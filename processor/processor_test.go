package processor

import (
	"math/big"
	"testing"

	"github.com/hyeonLewis/go-pevm/chain"
	"github.com/hyeonLewis/go-pevm/constants"
	test "github.com/hyeonLewis/go-pevm/contracts"
	"github.com/hyeonLewis/go-pevm/multiversion"
	"github.com/hyeonLewis/go-pevm/storage"
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
	numTxs         = 50
)

var (
	contractAddress = common.HexToAddress("0x0000000000000000000000000000000000000500")
	testByteCode = hexutil.MustDecode("0x" + test.TestBinRuntime)
)


func prepareChain() *blockchain.BlockChain {
	db := storage.NewInMemoryStorage()
	storage.InjectGenesis(db)

	bc := chain.NewBlockchain(db, db.ReadChainConfig(db.ReadCanonicalHash(0)))

	return bc
}

func prepareTestContract(bc *blockchain.BlockChain, state *state.StateDB) (*test.TestCaller) {
	state.SetCode(contractAddress, testByteCode)

	caller, _ := test.NewTestCaller(contractAddress, backends.NewBlockchainContractBackend(bc, nil, nil))

	return caller
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

func executeTxsSequential(txs []*types.Transaction, bc *blockchain.BlockChain, state *state.StateDB) (common.Hash, error) {
	executor := chain.NewExecutor(bc.Config(), state, types.CopyHeader(bc.CurrentHeader()))
	for i, tx := range txs {
		state.SetTxContext(tx.Hash(), common.Hash{}, i)
		tracer := multiversion.NewAccessListTracer(multiversion.NewMultiVersionStore(state), tx, i, 0, nil)
		err, _ := executor.CommitTransaction(tx, bc, constants.DefaultRewardBase, &vm.Config{Debug: true, Tracer: tracer})
		if err != nil {
			return common.Hash{}, err
		}
	}
	return state.IntermediateRoot(false), nil
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

	rootSequential, err := executeTxsSequential(txs, bc, stateCopy)
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

	rootSequential, err := executeTxsSequential(txs, bc, stateCopy)
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

	rootSequential, err := executeTxsSequential(txs, bc, stateCopy)
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

// TODO: Fix this test
func TestExecutionContract(t *testing.T) {
	bc := prepareChain()
	header := bc.CurrentHeader()

	state, _ := bc.State()
	caller := prepareTestContract(bc, state)

	txs, senders, err := prepareContractTx(bc, 1, 10)
	if err != nil {
		t.Fatal(err)
	}

	for _, sender := range senders {
		state.SetBalance(sender, big.NewInt(1000000000000000000))
	}

	processor, err := NewProcessor(txs, bc, header, state, 0)
	if err != nil {
		t.Fatal(err)
	}
	resp := processor.Execute()

	assert.Equal(t, len(resp), 1)
	assert.Equal(t, resp[0].Receipt.Status, uint(1))
	for i := 0; i < 10; i++ {
		arr, err := caller.Arr(nil, big.NewInt(int64(i)))
		if err != nil {
			t.Fatal(err)
		}
		assert.Equal(t, arr, big.NewInt(int64(i)))
	}
}

// Benchmark

// Before avoiding validation:
// 1000000000	         0.01709 ns/op	       0 B/op	       0 allocs/op
// After avoiding validation:
// 1000000000	         0.007191 ns/op	       0 B/op	       0 allocs/op
func BenchmarkDependentValueTransferTxsConcurrent(b *testing.B) {
	benchmarkDependentValueTransferTxsConcurrent(b, defaultWorkers)
}

// 1000000000	         0.006375 ns/op	       0 B/op	       0 allocs/op
func BenchmarkDependentValueTransferTxsSequential(b *testing.B) {
	benchmarkDependentValueTransferTxsSequential(b)
}

// 1000000000	         0.008027 ns/op	       0 B/op	       0 allocs/op
func BenchmarkValueTransferMultipleTxsConcurrent(b *testing.B) {
	benchmarkValueTransferMultipleTxsConcurrent(b, defaultWorkers)
}

// 1000000000	         0.01039 ns/op	       0 B/op	       0 allocs/op
func BenchmarkValueTransferMultipleTxsSequential(b *testing.B) {
	benchmarkValueTransferMultipleTxsSequential(b)
}


func benchmarkDependentValueTransferTxsConcurrent(b *testing.B, workers int) {
	bc := prepareChain()
	header := bc.CurrentHeader()

	state, _ := bc.State()
	// This is to warm up the cache
	state.Exist(constants.ValidatorAddress)
	value := big.NewInt(10000)
	txs, _ := prepareSimpleValueTransferTx(bc, value, 50)

	if workers > len(txs) {
		workers = len(txs)
	}
	processor, _ := NewProcessor(txs, bc, header, state, workers)

	processor.Execute()
	state.IntermediateRoot(false)
}

func benchmarkDependentValueTransferTxsSequential(b *testing.B) {
	bc := prepareChain()
	state, _ := bc.State()
	value := big.NewInt(10000)
	txs, _ := prepareSimpleValueTransferTx(bc, value, 50)

	executeTxsSequential(txs, bc, state)
}

func benchmarkValueTransferMultipleTxsConcurrent(b *testing.B, workers int) {
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

func benchmarkValueTransferMultipleTxsSequential(b *testing.B) {
	bc := prepareChain()

	value := big.NewInt(10000)
	txs, senders, _, _ := prepareValueTransferTxWithSender(bc, value, numTxs)
	state, _ := bc.State()
	for _, sender := range senders {
		state.SetBalance(sender, big.NewInt(1000000000000000000))
	}

	executeTxsSequential(txs, bc, state)
}
