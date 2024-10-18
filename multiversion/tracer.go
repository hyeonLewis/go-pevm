package multiversion

import (
	"errors"
	"fmt"
	"math/big"
	"sort"

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

	transactionIndex int
	incarnation      int

	abortedChannel chan Abort
}

func NewAccessListTracer(multiVersionStore MultiVersionStore, transactionIndex int, incarnation int, abortedChannel chan Abort) *AccessListTracer {
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
}

// CaptureState captures all opcodes that touch storage or addresses and adds them to the accesslist.
func (a *AccessListTracer) CaptureState(env *vm.EVM, pc uint64, op vm.OpCode, gas, cost, ccLeft, ccOpcode uint64, scope *vm.ScopeContext, depth int, err error) {
	stack := scope.Stack
	stackData := stack.Data()
	stackLen := len(stackData)

	switch op {
	case vm.SSTORE:
		if stackLen >= 2 {
			slot := common.Hash(stackData[stackLen-1].Bytes32())
			value := stackData[stackLen-2].Bytes()
			storageKey := ToStorageKey(scope.Contract.Address(), slot)
			a.writeset[storageKey] = common.BytesToHash(value)
		}
	case vm.SLOAD:
		if stackLen >= 1 {
			slot := common.Hash(stackData[stackLen-1].Bytes32())
			value := env.StateDB.GetState(scope.Contract.Address(), slot)
			storageKey := ToStorageKey(scope.Contract.Address(), slot)
			a.ValidGet(storageKey)
			a.UpdateReadSet(storageKey, value)
		}
	case vm.EXTCODECOPY:
		if stackLen >= 4 {
			addr := common.Address(stackData[stackLen-1].Bytes20())
			// Use codeHash instead of codeCopy
			codeHash := env.StateDB.GetCodeHash(addr)
			storageKey := ToStorageKey(addr, CodeCopyKey)
			a.ValidGet(storageKey)
			if _, ok := a.excl[addr]; !ok {
				a.UpdateReadSet(storageKey, codeHash)
			}
		}
	case vm.EXTCODEHASH:
		if stackLen >= 1 {
			addr := common.Address(stackData[stackLen-1].Bytes20())
			codeHash := env.StateDB.GetCodeHash(addr)
			if _, ok := a.excl[addr]; !ok {
				storageKey := ToStorageKey(addr, CodeHashKey)
				a.ValidGet(storageKey)
				a.UpdateReadSet(storageKey, codeHash)
			}
		}
	case vm.EXTCODESIZE:
		if stackLen >= 1 {
			addr := common.Address(stackData[stackLen-1].Bytes20())
			codeSize := env.StateDB.GetCodeSize(addr)
			if _, ok := a.excl[addr]; !ok {
				storageKey := ToStorageKey(addr, CodeSizeKey)
				a.ValidGet(storageKey)
				a.UpdateReadSet(storageKey, common.BigToHash(big.NewInt(int64(codeSize))))
			}
		}
	case vm.BALANCE:
		if stackLen >= 1 {
			addr := common.Address(stackData[stackLen-1].Bytes20())
			balance := env.StateDB.GetBalance(addr)
			if _, ok := a.excl[addr]; !ok {
				storageKey := ToStorageKey(addr, BalanceKey)
				a.ValidGet(storageKey)
				a.UpdateReadSet(storageKey, common.BigToHash(balance))
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

// vm.EXTCODEHASH, vm.EXTCODESIZE, vm.BALANCE, vm.SELFDESTRUCT

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

func (*AccessListTracer) CaptureTxEnd(restGas uint64) {}

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

// Get implements types.KVStore.
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
		} else {
			// This handles both detecting readset conflicts and updating readset if applicable
			a.parseValueAndUpdateReadset(key, mvsValue)
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
