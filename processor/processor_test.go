package processor

import (
	"context"
	"fmt"
	"math/big"
	"math/rand"
	"testing"
	"time"

	"github.com/hyeonLewis/go-pevm/chain"
	"github.com/hyeonLewis/go-pevm/constants"
	test "github.com/hyeonLewis/go-pevm/contracts"
	"github.com/hyeonLewis/go-pevm/multiversion"
	"github.com/hyeonLewis/go-pevm/storage"
	"github.com/kaiachain/kaia"
	"github.com/kaiachain/kaia/accounts/abi/bind/backends"
	"github.com/kaiachain/kaia/blockchain"
	"github.com/kaiachain/kaia/blockchain/state"
	"github.com/kaiachain/kaia/blockchain/types"
	"github.com/kaiachain/kaia/blockchain/vm"
	"github.com/kaiachain/kaia/common"
	"github.com/kaiachain/kaia/common/hexutil"
	"github.com/kaiachain/kaia/crypto"
	"github.com/stretchr/testify/assert"
)

const (
	defaultWorkers = 20
	numTxs         = 500
	numContractTxs = 100
	benchmarkLoop  = 10
)

var (
	contractAddress = common.HexToAddress("0x0000000000000000000000000000000000000500")
	testByteCode    = hexutil.MustDecode("0x" + test.TestBinRuntime)
)

type ContractCallerForTest struct {
	state  *state.StateDB               // the state that is under process
	chain  backends.BlockChainForCaller // chain containing the blockchain information
	header *types.Header                // the header of a new block that is under process
}

func (caller *ContractCallerForTest) CodeAt(ctx context.Context, contract common.Address, blockNumber *big.Int) ([]byte, error) {
	return contractAddress[:], nil
}

func (caller *ContractCallerForTest) CallContract(ctx context.Context, call kaia.CallMsg, blockNumber *big.Int) ([]byte, error) {
	gasPrice := big.NewInt(0) // execute call regardless of the balance of the sender
	gasLimit := uint64(1e8)   // enough gas limit to execute multicall contract functions
	intrinsicGas := uint64(0) // read operation doesn't require intrinsicGas

	// call.From: zero address will be assigned if nothing is specified
	// call.To: the target contract address will be assigned by `BoundContract`
	// call.Value: nil value is acceptable for `types.NewMessage`
	// call.Data: a proper value will be assigned by `BoundContract`
	// No need to handle access list here

	msg := types.NewMessage(call.From, call.To, caller.state.GetNonce(call.From),
		call.Value, gasLimit, gasPrice, nil, nil, call.Data, false, intrinsicGas, nil, nil)

	blockContext := blockchain.NewEVMBlockContext(caller.header, caller.chain, nil)
	txContext := blockchain.NewEVMTxContext(msg, caller.header, caller.chain.Config())
	txContext.GasPrice = gasPrice                                                                // set gasPrice again if baseFee is assigned
	evm := vm.NewEVM(blockContext, txContext, caller.state, caller.chain.Config(), &vm.Config{}) // no additional vm config required

	result, err := blockchain.ApplyMessage(evm, msg)
	return result.Return(), err
}

func prepareChain() *blockchain.BlockChain {
	db := storage.NewInMemoryStorage()
	storage.InjectGenesis(db)

	bc := chain.NewBlockchain(db, db.ReadChainConfig(db.ReadCanonicalHash(0)))

	return bc
}

func prepareTestContract(bc *blockchain.BlockChain, state *state.StateDB) (*test.TestCaller, error) {
	caller := &ContractCallerForTest{
		state:  state,
		chain:  bc,
		header: bc.CurrentHeader(),
	}

	return test.NewTestCaller(contractAddress, caller)
}

// Same sender and receiver with different nonce
func prepareSimpleValueTransferTx(bc *blockchain.BlockChain, value *big.Int, num int) ([]*types.Transaction, error) {
	config := bc.Config()
	txs := make([]*types.Transaction, num)
	for i := 0; i < num; i++ {
		tx := types.NewTransaction(uint64(i), constants.RandomAddress, value, 3000000, big.NewInt(25*1e9), nil)
		err := tx.Sign(types.NewEIP155Signer(config.ChainID), constants.ValidatorPrivateKey)
		if err != nil {
			return nil, err
		}
		txs[i] = tx
	}
	return txs, nil
}

func prepareValueTransferTxWithSender(bc *blockchain.BlockChain, value *big.Int, num int) ([]*types.Transaction, []common.Address, []common.Address, error) {
	config := bc.Config()
	txs := make([]*types.Transaction, num)
	senderAddrs := make([]common.Address, num)
	receiverAddrs := make([]common.Address, num)
	for i := 0; i < num; i++ {
		senderPrivKey, _ := crypto.GenerateKey()
		senderAddrs[i] = crypto.PubkeyToAddress(senderPrivKey.PublicKey)
		receiverPrivKey, _ := crypto.GenerateKey()
		receiverAddrs[i] = crypto.PubkeyToAddress(receiverPrivKey.PublicKey)
		tx := types.NewTransaction(0, receiverAddrs[i], value, 3000000, big.NewInt(25*1e9), nil)
		err := tx.Sign(types.NewEIP155Signer(config.ChainID), senderPrivKey)
		if err != nil {
			return nil, nil, nil, err
		}
		txs[i] = tx
	}
	return txs, senderAddrs, receiverAddrs, nil
}

func prepareContractTx(bc *blockchain.BlockChain, num int, len int) ([]*types.Transaction, []common.Address, error) {
	config := bc.Config()
	txs := make([]*types.Transaction, num)
	senderAddrs := make([]common.Address, num)
	abi, _ := test.TestMetaData.GetAbi()
	input, _ := abi.Pack("setArr", big.NewInt(int64(len)))
	for i := 0; i < num; i++ {
		senderPrivKey, _ := crypto.GenerateKey()
		senderAddrs[i] = crypto.PubkeyToAddress(senderPrivKey.PublicKey)
		tx := types.NewTransaction(0, contractAddress, big.NewInt(0), 3000000, big.NewInt(25*1e9), input)
		err := tx.Sign(types.NewEIP155Signer(config.ChainID), senderPrivKey)
		if err != nil {
			return nil, nil, err
		}
		txs[i] = tx
	}
	return txs, senderAddrs, nil
}

func prepareIndependentContractTx(bc *blockchain.BlockChain, num int) ([]*types.Transaction, []common.Address, error) {
	config := bc.Config()
	txs := make([]*types.Transaction, num)
	senderAddrs := make([]common.Address, num)
	abi, _ := test.TestMetaData.GetAbi()
	for i := 0; i < num; i++ {
		input, _ := abi.Pack("setMap", big.NewInt(int64(i)))
		senderPrivKey, _ := crypto.GenerateKey()
		senderAddrs[i] = crypto.PubkeyToAddress(senderPrivKey.PublicKey)
		tx := types.NewTransaction(0, contractAddress, big.NewInt(0), 30_000_000, big.NewInt(25*1e9), input)
		err := tx.Sign(types.NewEIP155Signer(config.ChainID), senderPrivKey)
		if err != nil {
			return nil, nil, err
		}
		txs[i] = tx
	}
	return txs, senderAddrs, nil
}

func executeTxsSequential(txs []*types.Transaction, bc *blockchain.BlockChain, state *state.StateDB) (common.Hash, []*types.Receipt, error) {
	executor := chain.NewExecutor(bc.Config(), state, types.CopyHeader(bc.CurrentHeader()))
	author, _ := bc.Engine().Author(bc.CurrentHeader())
	receipts := make([]*types.Receipt, len(txs))
	for i, tx := range txs {
		state.SetTxContext(tx.Hash(), common.Hash{}, i)
		tracer := multiversion.NewAccessListTracer(multiversion.NewMultiVersionStore(state), tx, i, 0, nil)
		err, resp := executor.CommitTransaction(tx, bc, author, &vm.Config{Debug: true, Tracer: tracer})
		if err != nil {
			return common.Hash{}, nil, err
		}
		receipts[i] = resp
	}
	return state.IntermediateRoot(false), receipts, nil
}

func TestValueTransferSingleTx(t *testing.T) {
	bc := prepareChain()
	header := bc.CurrentHeader()

	state, _ := bc.State()
	stateCopy := state.Copy()
	value := big.NewInt(10000)
	txs, err := prepareSimpleValueTransferTx(bc, value, 1)
	if err != nil {
		t.Fatal(err)
	}

	processor, err := NewProcessor(txs, bc, header, state, 0)
	if err != nil {
		t.Fatal(err)
	}
	resp := processor.Execute()

	rootSequential, _, err := executeTxsSequential(txs, bc, stateCopy)
	if err != nil {
		t.Fatal(err)
	}

	root := state.IntermediateRoot(false)

	assert.Equal(t, root, rootSequential)
	assert.Equal(t, len(resp), 1)
	assert.Equal(t, resp[0].Receipt.Status, uint(1))
	assert.Equal(t, state.GetBalance(constants.RandomAddress), value)
	assert.Equal(t, state.GetNonce(constants.ValidatorAddress), uint64(1))
}

func TestDependentValueTransferMultipleTxs(t *testing.T) {
	bc := prepareChain()
	header := bc.CurrentHeader()

	state, _ := bc.State()
	stateCopy := state.Copy()
	value := big.NewInt(10000)
	txs, err := prepareSimpleValueTransferTx(bc, value, 50)
	if err != nil {
		t.Fatal(err)
	}

	processor, err := NewProcessor(txs, bc, header, state, 0)
	if err != nil {
		t.Fatal(err)
	}
	resp := processor.Execute()

	rootSequential, _, err := executeTxsSequential(txs, bc, stateCopy)
	if err != nil {
		t.Fatal(err)
	}

	root := state.IntermediateRoot(false)

	assert.Equal(t, root, rootSequential)
	assert.Equal(t, len(resp), 50)
	for _, r := range resp {
		assert.Equal(t, r.Receipt.Status, uint(1))
	}
	assert.Equal(t, state.GetBalance(constants.RandomAddress), new(big.Int).Mul(value, big.NewInt(50)))
	assert.Equal(t, state.GetNonce(constants.ValidatorAddress), uint64(50))
}

func TestValueTransferMultipleTxsConcurrent(t *testing.T) {
	bc := prepareChain()
	header := bc.CurrentHeader()

	value := big.NewInt(10000)
	txs, senders, receivers, err := prepareValueTransferTxWithSender(bc, value, 50)
	state, _ := bc.State()
	for _, sender := range senders {
		state.SetBalance(sender, big.NewInt(1000000000000000000))
	}
	stateCopy := state.Copy()
	if err != nil {
		t.Fatal(err)
	}
	processor, err := NewProcessor(txs, bc, header, state, 0)
	if err != nil {
		t.Fatal(err)
	}
	resp := processor.Execute()

	rootSequential, _, err := executeTxsSequential(txs, bc, stateCopy)
	if err != nil {
		t.Fatal(err)
	}

	root := state.IntermediateRoot(false)

	assert.Equal(t, root, rootSequential)
	assert.Equal(t, len(resp), 50)
	for _, r := range resp {
		assert.Equal(t, r.Receipt.Status, uint(1))
	}
	for _, receiver := range receivers {
		assert.Equal(t, state.GetBalance(receiver), value)
	}
	for _, sender := range senders {
		assert.Equal(t, state.GetNonce(sender), uint64(1))
	}
}

func TestExecutionContractTx(t *testing.T) {
	bc := prepareChain()
	header := bc.CurrentHeader()

	state, _ := bc.State()
	caller, _ := prepareTestContract(bc, state)

	txs, senders, err := prepareContractTx(bc, 1, 10)
	if err != nil {
		t.Fatal(err)
	}

	for _, sender := range senders {
		state.SetBalance(sender, big.NewInt(1000000000000000000))
	}
	stateCopy := state.Copy()

	processor, err := NewProcessor(txs, bc, header, state, 0)
	if err != nil {
		t.Fatal(err)
	}
	resp := processor.Execute()

	rootSequential, _, err := executeTxsSequential(txs, bc, stateCopy)
	if err != nil {
		t.Fatal(err)
	}

	root := state.IntermediateRoot(false)

	assert.Equal(t, root, rootSequential)

	assert.Equal(t, len(resp), 1)
	assert.Equal(t, resp[0].Receipt.Status, uint(1))
	for i := 0; i < 10; i++ {
		arr, err := caller.Arr(nil, big.NewInt(int64(i)))
		if err != nil {
			t.Fatal(err)
		}
		assert.Equal(t, arr.Uint64(), uint64(i))
	}
}

func TestExecutionContractTxs(t *testing.T) {
	bc := prepareChain()

	state, _ := bc.State()
	caller, _ := prepareTestContract(bc, state)

	loop := 10
	txsNum := 100
	txs, senders, err := prepareContractTx(bc, txsNum, loop)
	if err != nil {
		t.Fatal(err)
	}

	for _, sender := range senders {
		state.SetBalance(sender, big.NewInt(1000000000000000000))
	}
	stateCopy := state.Copy()
	callerCopy, _ := prepareTestContract(bc, stateCopy)

	processor, err := NewProcessor(txs, bc, bc.CurrentHeader(), state, 0)
	if err != nil {
		t.Fatal(err)
	}
	resp := processor.Execute()
	root := state.IntermediateRoot(false)

	rootSequential, _, err := executeTxsSequential(txs, bc, stateCopy)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, root, rootSequential)

	assert.Equal(t, len(resp), txsNum)
	for _, r := range resp {
		assert.Equal(t, r.Receipt.Status, uint(1))
	}
	for i := 0; i < loop; i++ {
		arr, err := caller.Arr(nil, big.NewInt(int64(i)))
		arrCopy, err2 := callerCopy.Arr(nil, big.NewInt(int64(i)))
		if err != nil {
			t.Fatal(err)
		}
		if err2 != nil {
			fmt.Println(err2)
			t.Fatal(err2)
		}
		assert.Equal(t, uint64(i), arr.Uint64())
		assert.Equal(t, uint64(i), arrCopy.Uint64())
	}
}

func TestExecutionIndependentContractTxs(t *testing.T) {
	bc := prepareChain()

	state, _ := bc.State()
	caller, _ := prepareTestContract(bc, state)

	txsNum := 100
	txs, senders, err := prepareIndependentContractTx(bc, txsNum)
	if err != nil {
		t.Fatal(err)
	}

	for _, sender := range senders {
		state.SetBalance(sender, big.NewInt(1000000000000000000))
	}
	stateCopy := state.Copy()
	callerCopy, _ := prepareTestContract(bc, stateCopy)

	processor, err := NewProcessor(txs, bc, bc.CurrentHeader(), state, 0)
	if err != nil {
		t.Fatal(err)
	}
	resp := processor.Execute()
	root := state.IntermediateRoot(false)

	rootSequential, _, err := executeTxsSequential(txs, bc, stateCopy)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, root, rootSequential)

	assert.Equal(t, len(resp), txsNum)
	for _, r := range resp {
		assert.Equal(t, r.Receipt.Status, uint(1))
	}
	for i := 0; i < txsNum; i++ {
		arr, err := caller.Map(nil, senders[i])
		arrCopy, err2 := callerCopy.Map(nil, senders[i])
		if err != nil {
			t.Fatal(err)
		}
		if err2 != nil {
			fmt.Println(err2)
			t.Fatal(err2)
		}
		assert.Equal(t, uint64(i*500), arr.Uint64())
		assert.Equal(t, uint64(i*500), arrCopy.Uint64())
	}
}

// Benchmark
func BenchmarkDependentValueTransferTxsConcurrent(b *testing.B) {
	benchmarkDependentValueTransferTxsConcurrent(b, defaultWorkers)
}

func BenchmarkDependentValueTransferTxsSequential(b *testing.B) {
	benchmarkDependentValueTransferTxsSequential(b)
}

func BenchmarkValueTransferMultipleTxsConcurrent(b *testing.B) {
	benchmarkValueTransferMultipleTxsConcurrent(b, defaultWorkers)
}

func BenchmarkValueTransferMultipleTxsSequential(b *testing.B) {
	benchmarkValueTransferMultipleTxsSequential(b)
}

func BenchmarkDependentSmartContractConcurrent(b *testing.B) {
	benchmarkSmartContractConcurrent(b, defaultWorkers)
}

func BenchmarkDependentSmartContractSequential(b *testing.B) {
	benchmarkSmartContractSequential(b)
}

func BenchmarkIndependentSmartContractConcurrent(b *testing.B) {
	benchmarkIndependentSmartContractConcurrent(b, defaultWorkers)
}

func BenchmarkIndependentSmartContractSequential(b *testing.B) {
	benchmarkIndependentSmartContractSequential(b)
}

func BenchmarkIndependentAndDependentSmartContractConcurrent(b *testing.B) {
	benchmarkIndependentAndDependentSmartContractConcurrent(b, defaultWorkers)
}

func BenchmarkIndependentAndDependentSmartContractSequential(b *testing.B) {
	benchmarkIndependentAndDependentSmartContractSequential(b)
}

func benchmarkDependentValueTransferTxsConcurrent(b *testing.B, workers int) {
	for i := 0; i < b.N; i++ {
		bc := prepareChain()
		header := bc.CurrentHeader()

		state, _ := bc.State()
		// This is to warm up the cache
		state.Exist(constants.ValidatorAddress)
		value := big.NewInt(10000)
		txs, _ := prepareSimpleValueTransferTx(bc, value, numTxs)

		if workers > len(txs) {
			workers = len(txs)
		}
		processor, _ := NewProcessor(txs, bc, header, state, workers)

		processor.Execute()
		state.IntermediateRoot(false)
	}
}

func benchmarkDependentValueTransferTxsSequential(b *testing.B) {
	for i := 0; i < b.N; i++ {
		bc := prepareChain()
		state, _ := bc.State()
		value := big.NewInt(10000)
		txs, _ := prepareSimpleValueTransferTx(bc, value, numTxs)

		executeTxsSequential(txs, bc, state)
	}
}

func benchmarkValueTransferMultipleTxsConcurrent(b *testing.B, workers int) {
	for i := 0; i < b.N; i++ {
		bc := prepareChain()
		header := bc.CurrentHeader()

		value := big.NewInt(10000)
		txs, senders, _, _ := prepareValueTransferTxWithSender(bc, value, numTxs)
		state, _ := bc.State()
		for _, sender := range senders {
			state.SetBalance(sender, big.NewInt(1000000000000000000))
		}

		if workers > numTxs {
			workers = numTxs
		}
		processor, _ := NewProcessor(txs, bc, header, state, workers)
		processor.Execute()
		state.IntermediateRoot(false)
	}
}

func benchmarkValueTransferMultipleTxsSequential(b *testing.B) {
	for i := 0; i < b.N; i++ {
		bc := prepareChain()

		value := big.NewInt(10000)
		txs, senders, _, _ := prepareValueTransferTxWithSender(bc, value, numTxs)
		state, _ := bc.State()
		for _, sender := range senders {
			state.SetBalance(sender, big.NewInt(1000000000000000000))
		}

		executeTxsSequential(txs, bc, state)
	}
}

func benchmarkSmartContractConcurrent(b *testing.B, workers int) {
	for i := 0; i < b.N; i++ {
		bc := prepareChain()

		state, _ := bc.State()

		txs, senders, _ := prepareContractTx(bc, numContractTxs, benchmarkLoop)

		for _, sender := range senders {
			state.SetBalance(sender, big.NewInt(1000000000000000000))
		}

		processor, _ := NewProcessor(txs, bc, bc.CurrentHeader(), state, workers)
		processor.Execute()
		state.IntermediateRoot(false)
	}
}

func benchmarkSmartContractSequential(b *testing.B) {
	for i := 0; i < b.N; i++ {
		bc := prepareChain()

		state, _ := bc.State()

		txs, senders, _ := prepareContractTx(bc, numContractTxs, benchmarkLoop)

		for _, sender := range senders {
			state.SetBalance(sender, big.NewInt(1000000000000000000))
		}

		executeTxsSequential(txs, bc, state)
	}
}

func benchmarkIndependentSmartContractConcurrent(b *testing.B, workers int) {
	for i := 0; i < b.N; i++ {
		bc := prepareChain()

		state, _ := bc.State()

		txs, senders, _ := prepareIndependentContractTx(bc, numTxs)

		for _, sender := range senders {
			state.SetBalance(sender, big.NewInt(1000000000000000000))
		}

		processor, _ := NewProcessor(txs, bc, bc.CurrentHeader(), state, workers)
		processor.Execute()
		state.IntermediateRoot(false)
	}
}

func benchmarkIndependentSmartContractSequential(b *testing.B) {
	for i := 0; i < b.N; i++ {
		bc := prepareChain()

		state, _ := bc.State()

		txs, senders, _ := prepareIndependentContractTx(bc, numTxs)

		for _, sender := range senders {
			state.SetBalance(sender, big.NewInt(1000000000000000000))
		}

		executeTxsSequential(txs, bc, state)
	}
}

func benchmarkIndependentAndDependentSmartContractConcurrent(b *testing.B, workers int) {
	for i := 0; i < b.N; i++ {
		bc := prepareChain()
		state, _ := bc.State()

		independentTxCount := 30
		independentTxs, senders, _ := prepareIndependentContractTx(bc, independentTxCount)

		for _, sender := range senders {
			state.SetBalance(sender, big.NewInt(1000000000000000000))
		}

		dependentTxCount := 70
		dependentTxs, senders, _ := prepareContractTx(bc, dependentTxCount, benchmarkLoop)

		for _, sender := range senders {
			state.SetBalance(sender, big.NewInt(1000000000000000000))
		}

		txs := append(dependentTxs, independentTxs...)
		shuffle := func(txs []*types.Transaction) {
			r := rand.New(rand.NewSource(time.Now().UnixNano()))
			n := len(txs)
			for i := n - 1; i > 0; i-- {
				j := r.Intn(i + 1)
				txs[i], txs[j] = txs[j], txs[i]
			}
		}
		shuffle(txs)

		processor, _ := NewProcessor(txs, bc, bc.CurrentHeader(), state, workers)
		processor.Execute()
		state.IntermediateRoot(false)
	}
}

func benchmarkIndependentAndDependentSmartContractSequential(b *testing.B) {
	for i := 0; i < b.N; i++ {
		bc := prepareChain()
		state, _ := bc.State()

		independentTxCount := 30
		independentTxs, senders, _ := prepareIndependentContractTx(bc, independentTxCount)

		for _, sender := range senders {
			state.SetBalance(sender, big.NewInt(1000000000000000000))
		}

		dependentTxCount := 70
		dependentTxs, senders, _ := prepareContractTx(bc, dependentTxCount, benchmarkLoop)

		for _, sender := range senders {
			state.SetBalance(sender, big.NewInt(1000000000000000000))
		}

		txs := append(dependentTxs, independentTxs...)
		shuffle := func(txs []*types.Transaction) {
			r := rand.New(rand.NewSource(time.Now().UnixNano()))
			n := len(txs)
			for i := n - 1; i > 0; i-- {
				j := r.Intn(i + 1)
				txs[i], txs[j] = txs[j], txs[i]
			}
		}
		shuffle(txs)

		executeTxsSequential(txs, bc, state)
	}
}
