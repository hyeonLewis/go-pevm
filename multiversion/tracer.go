package multiversion

import (
	"errors"
	"fmt"
	"math/big"
	"sort"

	"github.com/kaiachain/kaia/blockchain"
	"github.com/kaiachain/kaia/blockchain/types"
	"github.com/kaiachain/kaia/blockchain/vm"
	"github.com/kaiachain/kaia/common"
)

var (
	ErrReadEstimate            = errors.New("multiversion store value contains estimate, cannot read, aborting")
	ErrNeedSequentialExecution = errors.New("transaction must execute sequentially, aborting")
)

// Abort contains the information for a transaction's conflict
type Abort struct {
	DependentTxIdx int
	Err            error
}

func NewEstimateAbort(dependentTxIdx int) Abort {
	return Abort{
		DependentTxIdx: dependentTxIdx,
		Err:            ErrReadEstimate,
	}
}

// accessList is an accumulator for the set of accounts and storage slots an EVM
// contract execution touches.
type accessList map[common.Address]struct{}

type AccessListTracer struct {
	excl     map[common.Address]struct{}
	readset  map[StorageKey][]common.Hash
	writeset map[StorageKey]common.Hash

	multiVersionStore MultiVersionStore

	msg blockchain.Message

	gasPrice *big.Int

	transactionIndex int
	incarnation      int

	abortedChannel chan Abort
}

// TODO-kaia: add initial value transfer to writeset?
func NewAccessListTracer(multiVersionStore MultiVersionStore, msg blockchain.Message, transactionIndex int, incarnation int, abortedChannel chan Abort) *AccessListTracer {
	excl := make(map[common.Address]struct{})
	precompiles := vm.PrecompiledAddressCancun
	for _, addr := range precompiles {
		excl[addr] = struct{}{}
	}
	return &AccessListTracer{
		excl:              excl,
		readset:           make(map[StorageKey][]common.Hash),
		writeset:          make(map[StorageKey]common.Hash),
		multiVersionStore: multiVersionStore,
		msg:               msg,
		transactionIndex:  transactionIndex,
		incarnation:       incarnation,
		abortedChannel:    abortedChannel,
	}
}

// GetReadset returns the readset
func (a *AccessListTracer) GetReadset() map[StorageKey][]common.Hash {
	return a.readset
}

// GetWriteset returns the writeset
func (a *AccessListTracer) GetWriteset() map[StorageKey]common.Hash {
	return a.writeset
}

func (a *AccessListTracer) WriteAbort(abort Abort) {
	select {
	case a.abortedChannel <- abort:
	default:
		fmt.Println("WARN: abort channel full, discarding val")
	}
}

func (a *AccessListTracer) CaptureStart(env *vm.EVM, from common.Address, to common.Address, create bool, input []byte, gas uint64, value *big.Int) {
	if a.gasPrice == nil {
		a.gasPrice = env.GasPrice
	}

	// Gas fee related logic
	feePayer := a.msg.ValidatedFeePayer()
	sender := a.msg.ValidatedSender()
	feeRatio, isRatioTx := a.msg.FeeRatio()
	gasFee := new(big.Int).Mul(new(big.Int).SetUint64(a.msg.Gas()), a.gasPrice)
	if isRatioTx {
		a.handleRatioTransaction(feePayer, sender, feeRatio, gasFee)
	} else {
		a.handleRegularTransaction(feePayer, gasFee)
	}

	// Nonce related logic
	isContractSender := env.StateDB.IsProgramAccount(sender)
	if isContractSender {
		if create {
			a.updateAndSetNonce(sender)
		}
	} else {
		a.updateAndSetNonce(sender)
	}

	// Initial value transfer
	a.updateAndSetBalance(from, value)
	a.updateAndSetBalance(to, new(big.Int).Neg(value))
}

// CaptureState captures all opcodes that touch storage or addresses and adds them to the accesslist.
func (a *AccessListTracer) CaptureState(env *vm.EVM, pc uint64, op vm.OpCode, gas, cost, ccLeft, ccOpcode uint64, scope *vm.ScopeContext, depth int, err error) {
	stack := scope.Stack
	stackData := stack.Data()
	stackLen := len(stackData)

	switch op {
	case vm.SLOAD:
		if stackLen >= 1 {
			slot := common.Hash(stackData[stackLen-1].Bytes32())
			storageKey := ToStorageKey(scope.Contract.Address(), slot)
			a.Get(storageKey)
		}
	case vm.SSTORE:
		if stackLen >= 2 {
			slot := common.Hash(stackData[stackLen-1].Bytes32())
			value := stackData[stackLen-2].Bytes()
			storageKey := ToStorageKey(scope.Contract.Address(), slot)
			a.Set(storageKey, common.BytesToHash(value))
		}
	case vm.EXTCODECOPY:
		if stackLen >= 1 {
			addr := common.Address(stackData[stackLen-1].Bytes20())
			// Use codeHash instead of codeCopy
			storageKey := ToStorageKey(addr, CodeCopyKey)
			a.Get(storageKey)
		}
	case vm.EXTCODEHASH:
		if stackLen >= 1 {
			addr := common.Address(stackData[stackLen-1].Bytes20())
			storageKey := ToStorageKey(addr, CodeHashKey)
			a.Get(storageKey)
		}
	case vm.EXTCODESIZE:
		if stackLen >= 1 {
			addr := common.Address(stackData[stackLen-1].Bytes20())
			storageKey := ToStorageKey(addr, CodeSizeKey)
			a.Get(storageKey)
		}
	case vm.BALANCE:
		if stackLen >= 1 {
			addr := common.Address(stackData[stackLen-1].Bytes20())
			storageKey := ToStorageKey(addr, BalanceKey)
			a.Get(storageKey)
		}
	case vm.CALL, vm.CALLCODE:
		if stackLen >= 1 {
			fromAddr := scope.Contract.CallerAddress
			toAddr := common.Address(stackData[stackLen-1].Bytes20())
			value := scope.Contract.Value()
			if value.Sign() > 0 && fromAddr != toAddr {
				a.updateAndSetBalance(fromAddr, value)
				a.updateAndSetBalance(toAddr, new(big.Int).Neg(value))
			}

		}
	case vm.SELFDESTRUCT:
		// Need sequential execution to detect selfdestruct
		a.WriteAbort(Abort{
			DependentTxIdx: a.transactionIndex,
			Err:            ErrNeedSequentialExecution,
		})
	default:
		// do nothing for other opcodes
	}
}

func (a *AccessListTracer) UpdateReadSet(key StorageKey, value common.Hash) {
	if _, ok := a.readset[key]; !ok {
		// if the entry doesnt exist, make a new empty slice
		a.readset[key] = []common.Hash{}
	}
	for _, readsetVal := range a.readset[key] {
		if readsetVal == value {
			// this means we have already added this value to our readset, so we continue
			return
		}
	}
	// if we get here, that means we have a new readset val, so we append it to the slice
	a.readset[key] = append(a.readset[key], value)
}

// Delete implements types.KVStore.
func (a *AccessListTracer) Delete(key StorageKey) {
	// TODO: remove?
	// store.mtx.Lock()
	// defer store.mtx.Unlock()
	// defer telemetry.MeasureSince(time.Now(), "store", "mvkv", "delete")

	if key.Bytes() == nil {
		return
	}
	a.setValue(key, common.Hash{})
}

// Has implements types.KVStore.
func (a *AccessListTracer) Has(key StorageKey) bool {
	// necessary locking happens within store.Get
	return a.Get(key) != common.Hash{}
}

// Set implements types.KVStore.
func (a *AccessListTracer) Set(key StorageKey, value common.Hash) {
	// TODO: remove?
	// store.mtx.Lock()
	// defer store.mtx.Unlock()
	// defer telemetry.MeasureSince(time.Now(), "store", "mvkv", "set")

	if key.Bytes() == nil {
		return
	}
	a.setValue(key, value)
}

func (*AccessListTracer) CaptureFault(env *vm.EVM, pc uint64, op vm.OpCode, gas, cost, ccLeft, ccOpcode uint64, scope *vm.ScopeContext, depth int, err error) {
}

func (*AccessListTracer) CaptureEnd(output []byte, gasUsed uint64, err error) {}

func (*AccessListTracer) CaptureEnter(typ vm.OpCode, from common.Address, to common.Address, input []byte, gas uint64, value *big.Int) {
}

func (*AccessListTracer) CaptureExit(output []byte, gasUsed uint64, err error) {}

func (*AccessListTracer) CaptureTxStart(gasLimit uint64) {}

func (a *AccessListTracer) CaptureTxEnd(restGas uint64) {
	remaining := new(big.Int).Mul(new(big.Int).SetUint64(restGas), a.gasPrice)
	validatedFeePayer := a.msg.ValidatedFeePayer()
	validatedSender := a.msg.ValidatedSender()
	feeRatio, isRatioTx := a.msg.FeeRatio()
	if isRatioTx {
		feePayer, feeSender := types.CalcFeeWithRatio(feeRatio, remaining)
		a.updateAndSetBalance(validatedFeePayer, new(big.Int).Neg(feePayer))
		a.updateAndSetBalance(validatedSender, new(big.Int).Neg(feeSender))
	} else {
		a.updateAndSetBalance(validatedFeePayer, new(big.Int).Neg(remaining))
	}
}

// This function iterates over the readset, validating that the values in the readset are consistent with the values in the multiversion store and underlying parent store, and returns a boolean indicating validity
func (a *AccessListTracer) ValidateReadset() bool {
	// TODO: remove?
	// store.mtx.Lock()
	// defer store.mtx.Unlock()
	// defer telemetry.MeasureSince(time.Now(), "store", "mvkv", "validate_readset")

	// sort the readset keys - this is so we have consistent behavior when theres varying conflicts within the readset (eg. read conflict vs estimate)
	readsetKeys := make([]StorageKey, 0, len(a.readset))
	for key := range a.readset {
		readsetKeys = append(readsetKeys, key)
	}
	sort.Slice(readsetKeys, func(i, j int) bool {
		return readsetKeys[i] < readsetKeys[j]
	})

	// iterate over readset keys and values
	for _, key := range readsetKeys {
		valueArr := a.readset[key]
		if len(valueArr) != 1 {
			// if we have more than one value, we will fail the validation since we dedup when adding to readset
			return false
		}
		value := valueArr[0]
		mvsValue := a.multiVersionStore.GetLatestBeforeIndex(a.transactionIndex, key)
		if mvsValue != nil {
			if mvsValue.IsEstimate() {
				// if we see an estimate, that means that we need to abort and rerun
				a.WriteAbort(NewEstimateAbort(mvsValue.Index()))
				return false
			} else {
				if mvsValue.IsDeleted() {
					// check for `nil`
					if value != (common.Hash{}) {
						return false
					}
				} else {
					// check for equality
					if value != mvsValue.Value() {
						return false
					}
				}
			}
			continue // value is valid, continue to next key
		}

		// TODO-Kaia: Check if we need to care about this case
		parentValue := a.multiVersionStore.GetParentState().GetState(key.Address(), key.Slot())
		if parentValue != value {
			// this shouldnt happen because if we have a conflict it should always happen within multiversion store
			panic("we shouldn't ever have a readset conflict in parent store")
		}
		// value was correct, we can continue to the next value
	}
	return true
}

// Get implements types.KVStore.
func (a *AccessListTracer) Get(key StorageKey) common.Hash {
	if key.Bytes() == nil {
		return common.Hash{}
	}
	// first check the MVKV writeset, and return that value if present
	cacheValue, ok := a.writeset[key]
	if ok {
		// return the value from the cache, no need to update any readset stuff
		return cacheValue
	}
	// read the readset to see if the value exists - and return if applicable
	if readsetVal, ok := a.readset[key]; ok {
		// just return the first one, if there is more than one, we will fail the validation anyways
		return readsetVal[0]
	}

	// if we didn't find it, then we want to check the multivalue store + add to readset if applicable
	mvsValue := a.multiVersionStore.GetLatestBeforeIndex(a.transactionIndex, key)
	if mvsValue != nil {
		if mvsValue.IsEstimate() {
			abort := NewEstimateAbort(mvsValue.Index())
			a.WriteAbort(abort)
			panic(abort)
		} else {
			// This handles both detecting readset conflicts and updating readset if applicable
			return a.parseValueAndUpdateReadset(key, mvsValue)
		}
	}
	parentValue := key.GetValue(a.multiVersionStore.GetParentState())
	a.UpdateReadSet(key, parentValue)
	return parentValue
}

func (a *AccessListTracer) ValidGet(key StorageKey) {
	if key.Bytes() == nil {
		return
	}

	// if we didn't find it, then we want to check the multivalue store + add to readset if applicable
	mvsValue := a.multiVersionStore.GetLatestBeforeIndex(a.transactionIndex, key)
	if mvsValue != nil {
		if mvsValue.IsEstimate() {
			abort := NewEstimateAbort(mvsValue.Index())
			a.WriteAbort(abort)
			panic(abort)
		}
	}
}

func (a *AccessListTracer) parseValueAndUpdateReadset(key StorageKey, mvsValue MultiVersionValueItem) common.Hash {
	value := mvsValue.Value()
	if mvsValue.IsDeleted() {
		value = (common.Hash{})
	}
	a.UpdateReadSet(key, value)
	return value
}

// Only entrypoint to mutate writeset
func (a *AccessListTracer) setValue(key StorageKey, value common.Hash) {
	if key.Bytes() == nil {
		return
	}

	a.writeset[key] = value
}

func (a *AccessListTracer) WriteEstimatesToMultiVersionStore() {
	// TODO: remove?
	// store.mtx.Lock()
	// defer store.mtx.Unlock()
	// defer telemetry.MeasureSince(time.Now(), "store", "mvkv", "write_mvs")
	a.multiVersionStore.SetEstimatedWriteset(a.transactionIndex, a.incarnation, a.writeset)
	// TODO: do we need to write readset and iterateset in this case? I don't think so since if this is called it means we aren't doing validation
}

func (a *AccessListTracer) WriteToMultiVersionStore() {
	// TODO: remove?
	// store.mtx.Lock()
	// defer store.mtx.Unlock()
	// defer telemetry.MeasureSince(time.Now(), "store", "mvkv", "write_mvs")
	a.multiVersionStore.SetWriteset(a.transactionIndex, a.incarnation, a.writeset)
	a.multiVersionStore.SetReadset(a.transactionIndex, a.readset)
}

func (a *AccessListTracer) handleRatioTransaction(feePayer, sender common.Address, feeRatio types.FeeRatio, gasFee *big.Int) {
	feePayerFee, senderFee := types.CalcFeeWithRatio(feeRatio, gasFee)

	a.updateAndSetBalance(feePayer, feePayerFee)
	a.updateAndSetBalance(sender, senderFee)
}

func (a *AccessListTracer) handleRegularTransaction(feePayer common.Address, gasFee *big.Int) {
	a.updateAndSetBalance(feePayer, gasFee)
}

func (a *AccessListTracer) updateAndSetBalance(address common.Address, value *big.Int) {
	balance := hashToBig(a.Get(ToStorageKey(address, BalanceKey)))
	newBalance := new(big.Int).Sub(balance, value)
	a.Set(ToStorageKey(address, BalanceKey), common.BigToHash(newBalance))
}

func (a *AccessListTracer) updateAndSetNonce(address common.Address) {
	nonce := hashToBig(a.Get(ToStorageKey(address, NonceKey)))
	a.Set(ToStorageKey(address, NonceKey), common.BigToHash(nonce.Add(nonce, big.NewInt(1))))
}

// getData returns a slice from the data based on the start and size and pads
// up to size with zero's. This function is overflow safe.
func getData(data []byte, start uint64, size uint64) []byte {
	length := uint64(len(data))
	if start > length {
		start = length
	}
	end := start + size
	if end > length {
		end = length
	}
	return common.RightPadBytes(data[start:end], int(size))
}

func hashToBig(hash common.Hash) *big.Int {
	return new(big.Int).SetBytes(hash.Bytes())
}
