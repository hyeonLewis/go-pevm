package multiversion

import (
	"sort"
	"sync"

	"github.com/kaiachain/kaia/blockchain"
	"github.com/kaiachain/kaia/blockchain/state"
	"github.com/kaiachain/kaia/common"
)

type MultiVersionStore interface {
	GetLatest(key StorageKey) (value MultiVersionValueItem)
	GetLatestBeforeIndex(index int, key StorageKey) (value MultiVersionValueItem)
	GetParentState() *state.StateDB
	Has(index int, key StorageKey) bool
	WriteLatestToStore(state *state.StateDB)
	WriteLatestToStoreUntil(lastStoreIndex int, startIndex int, state *state.StateDB)
	SetWriteset(index int, incarnation int, writeset WriteSet)
	InvalidateWriteset(index int, incarnation int)
	SetEstimatedWriteset(index int, incarnation int, writeset WriteSet)
	GetAllWritesetKeys() map[int][]StorageKey
	SetReadset(index int, readset ReadSet)
	GetReadset(index int) ReadSet
	ClearReadset(index int)
	ValidateTransactionState(index int) (bool, []int)
	VersionedIndexedStore(msg blockchain.Message, index int, incarnation int, abortChannel chan Abort) *AccessListTracer
}

type WriteSet map[StorageKey]common.Hash
type ReadSet map[StorageKey][]common.Hash

var _ MultiVersionStore = (*Store)(nil)

type Store struct {
	// map that stores the key string -> MultiVersionValue mapping for accessing from a given key
	multiVersionMap *sync.Map
	// TODO: do we need to support iterators as well similar to how cachekv does it - yes

	txWritesetKeys *sync.Map // map of tx index -> writeset keys []StorageKey
	txReadSets     *sync.Map // map of tx index -> readset ReadSet
	txIterateSets  *sync.Map // map of tx index -> iterateset Iterateset

	// parent ParentStore
	parentStore *state.StateDB
}

func NewMultiVersionStore(parentStore *state.StateDB) *Store {
	return &Store{
		multiVersionMap: &sync.Map{},
		txWritesetKeys:  &sync.Map{},
		txReadSets:      &sync.Map{},
		txIterateSets:   &sync.Map{},
		parentStore:     parentStore,
	}
}

func (s *Store) VersionedIndexedStore(msg blockchain.Message, index int, incarnation int, abortChannel chan Abort) *AccessListTracer {
	return NewAccessListTracer(s, msg, index, incarnation, abortChannel)
}

func (s *Store) GetParentState() *state.StateDB {
	return s.parentStore
}

// GetLatest implements MultiVersionStore.
func (s *Store) GetLatest(key StorageKey) (value MultiVersionValueItem) {
	mvVal, found := s.multiVersionMap.Load(key)
	// if the key doesn't exist in the overall map, return nil
	if !found {
		return nil
	}
	latestVal, found := mvVal.(MultiVersionValue).GetLatest()
	if !found {
		return nil // this is possible IF there is are writeset that are then removed for that key
	}
	return latestVal
}

// GetLatestBeforeIndex implements MultiVersionStore.
func (s *Store) GetLatestBeforeIndex(index int, key StorageKey) (value MultiVersionValueItem) {
	mvVal, found := s.multiVersionMap.Load(key)
	// if the key doesn't exist in the overall map, return nil
	if !found {
		return nil
	}
	val, found := mvVal.(MultiVersionValue).GetLatestBeforeIndex(index)
	// otherwise, we may have found a value for that key, but its not written before the index passed in
	if !found {
		return nil
	}
	// found a value prior to the passed in index, return that value (could be estimate OR deleted, but it is a definitive value)
	return val
}

// Has implements MultiVersionStore. It checks if the key exists in the multiversion store at or before the specified index.
func (s *Store) Has(index int, key StorageKey) bool {
	mvVal, found := s.multiVersionMap.Load(key)
	// if the key doesn't exist in the overall map, return nil
	if !found {
		return false // this is okay because the caller of this will THEN need to access the parent store to verify that the key doesnt exist there
	}
	_, foundVal := mvVal.(MultiVersionValue).GetLatestBeforeIndex(index)
	return foundVal
}

func (s *Store) removeOldWriteset(index int, newWriteSet WriteSet) {
	writeset := make(map[StorageKey]common.Hash)
	if newWriteSet != nil {
		// if non-nil writeset passed in, we can use that to optimize removals
		writeset = newWriteSet
	}
	// if there is already a writeset existing, we should remove that fully
	oldKeys, loaded := s.txWritesetKeys.LoadAndDelete(index)
	if loaded {
		keys := oldKeys.([]StorageKey)
		// we need to delete all of the keys in the writeset from the multiversion store
		for _, key := range keys {
			// small optimization to check if the new writeset is going to write this key, if so, we can leave it behind
			if _, ok := writeset[key]; ok {
				// we don't need to remove this key because it will be overwritten anyways - saves the operation of removing + rebalancing underlying btree
				continue
			}
			// remove from the appropriate item if present in multiVersionMap
			mvVal, found := s.multiVersionMap.Load(key)
			// if the key doesn't exist in the overall map, return nil
			if !found {
				continue
			}
			mvVal.(MultiVersionValue).Remove(index)
		}
	}
}

// SetWriteset sets a writeset for a transaction index, and also writes all of the multiversion items in the writeset to the multiversion store.
// TODO: returns a list of NEW keys added
func (s *Store) SetWriteset(index int, incarnation int, writeset WriteSet) {
	// TODO: add telemetry spans
	// remove old writeset if it exists
	s.removeOldWriteset(index, writeset)

	writeSetKeys := make([]StorageKey, 0, len(writeset))
	for key, value := range writeset {
		writeSetKeys = append(writeSetKeys, key)
		loadVal, _ := s.multiVersionMap.LoadOrStore(key, NewMultiVersionItem()) // init if necessary
		mvVal := loadVal.(MultiVersionValue)
		if value == (common.Hash{}) {
			// delete if nil value
			// TODO: sync map
			mvVal.Delete(index, incarnation)
		} else {
			mvVal.Set(index, incarnation, value)
		}
	}
	sort.Slice(writeSetKeys, func(i, j int) bool {
		return writeSetKeys[i].String() < writeSetKeys[j].String()
	}) // TODO: if we're sorting here anyways, maybe we just put it into a btree instead of a slice
	s.txWritesetKeys.Store(index, writeSetKeys)
}

// InvalidateWriteset iterates over the keys for the given index and incarnation writeset and replaces with ESTIMATEs
func (s *Store) InvalidateWriteset(index int, incarnation int) {
	keysAny, found := s.txWritesetKeys.Load(index)
	if !found {
		return
	}
	keys := keysAny.([]StorageKey)
	for _, key := range keys {
		// invalidate all of the writeset items - is this suboptimal? - we could potentially do concurrently if slow because locking is on an item specific level
		val, _ := s.multiVersionMap.LoadOrStore(key, NewMultiVersionItem())
		val.(MultiVersionValue).SetEstimate(index, incarnation)
	}
	// we leave the writeset in place because we'll need it for key removal later if/when we replace with a new writeset
}

// SetEstimatedWriteset is used to directly write estimates instead of writing a writeset and later invalidating
func (s *Store) SetEstimatedWriteset(index int, incarnation int, writeset WriteSet) {
	// remove old writeset if it exists
	s.removeOldWriteset(index, writeset)

	writeSetKeys := make([]StorageKey, 0, len(writeset))
	// still need to save the writeset so we can remove the elements later:
	for key := range writeset {
		writeSetKeys = append(writeSetKeys, key)

		mvVal, _ := s.multiVersionMap.LoadOrStore(key, NewMultiVersionItem()) // init if necessary
		mvVal.(MultiVersionValue).SetEstimate(index, incarnation)
	}
	sort.Slice(writeSetKeys, func(i, j int) bool {
		return writeSetKeys[i].String() < writeSetKeys[j].String()
	})
	s.txWritesetKeys.Store(index, writeSetKeys)
}

// GetAllWritesetKeys implements MultiVersionStore.
func (s *Store) GetAllWritesetKeys() map[int][]StorageKey {
	writesetKeys := make(map[int][]StorageKey)
	// TODO: is this safe?
	s.txWritesetKeys.Range(func(key, value interface{}) bool {
		index := key.(int)
		keys := value.([]StorageKey)
		writesetKeys[index] = keys
		return true
	})

	return writesetKeys
}

func (s *Store) SetReadset(index int, readset ReadSet) {
	s.txReadSets.Store(index, readset)
}

func (s *Store) GetReadset(index int) ReadSet {
	readsetAny, found := s.txReadSets.Load(index)
	if !found {
		return nil
	}
	return readsetAny.(ReadSet)
}

func (s *Store) ClearReadset(index int) {
	s.txReadSets.Delete(index)
}

func (s *Store) checkReadsetAtIndex(index int) (bool, []int) {
	conflictSet := make(map[int]struct{})
	valid := true

	readSetAny, found := s.txReadSets.Load(index)
	if !found {
		return true, []int{}
	}
	readset := readSetAny.(ReadSet)
	// iterate over readset and check if the value is the same as the latest value relateive to txIndex in the multiversion store
	for key, valueArr := range readset {
		if len(valueArr) != 1 {
			valid = false
			continue
		}
		value := valueArr[0]
		// get the latest value from the multiversion store
		latestValue := s.GetLatestBeforeIndex(index, key)
		if latestValue == nil {
			// this is possible if we previously read a value from a transaction write that was later reverted, so this time we read from parent store
			parentVal := key.GetValue(s.parentStore)
			if parentVal != value {
				valid = false
			}
			continue
		} else {
			// if estimate, mark as conflict index
			// TODO-kaia: Re-evaluate if we should be invalidating here
			if latestValue.IsEstimate() {
				conflictSet[latestValue.Index()] = struct{}{}
				valid = false
			} else if latestValue.IsDeleted() {
				if value != (common.Hash{}) {
					// conflict
					// TODO: would we want to return early?
					conflictSet[latestValue.Index()] = struct{}{}
					valid = false
				}
			} else if latestValue.Value() != value {
				conflictSet[latestValue.Index()] = struct{}{}
				valid = false
			}
		}
	}

	conflictIndices := make([]int, 0, len(conflictSet))
	for index := range conflictSet {
		conflictIndices = append(conflictIndices, index)
	}

	sort.Ints(conflictIndices)

	return valid, conflictIndices
}

// TODO: do we want to return bool + []int where bool indicates whether it was valid and then []int indicates only ones for which we need to wait due to estimates? - yes i think so?
func (s *Store) ValidateTransactionState(index int) (bool, []int) {
	// defer telemetry.MeasureSince(time.Now(), "store", "mvs", "validate")

	readsetValid, conflictIndices := s.checkReadsetAtIndex(index)

	return readsetValid, conflictIndices
}

func (s *Store) WriteLatestToStoreUntil(lastStoreIndex int, startIndex int, finalState *state.StateDB) {
	// sort the keys
	keys := []StorageKey{}
	s.multiVersionMap.Range(func(key, value interface{}) bool {
		keys = append(keys, key.(StorageKey))
		return true
	})
	sort.Slice(keys, func(i, j int) bool {
		return keys[i].String() < keys[j].String()
	})
	for _, key := range keys {
		val, ok := s.multiVersionMap.Load(key)
		if !ok {
			continue
		}
		mvValue, found := val.(MultiVersionValue).GetLatestNonEstimate()
		if !found {
			// this means that at some point, there was an estimate, but we have since removed it so there isn't anything writeable at the key, so we can skip
			continue
		}
		// we shouldn't have any ESTIMATE values when performing the write, because we read the latest non-estimate values only
		if mvValue.IsEstimate() {
			panic("should not have any estimate values when writing to parent store")
		}
		// valIdx := mvValue.Index()
		// if valIdx < lastStoreIndex || valIdx >= startIndex {
		// 	continue
		// }
		// if the value is deleted, then delete it from the parent store
		if mvValue.IsDeleted() {
			// We use []byte(key) instead of conv.UnsafeStrToBytes because we cannot
			// be sure if the underlying store might do a save with the byteslice or
			// not. Once we get confirmation that .Delete is guaranteed not to
			// save the byteslice, then we can assume only a read-only copy is sufficient.
			key.SetValue(finalState, common.Hash{})
			continue
		}
		if mvValue.Value() != (common.Hash{}) {
			key.SetValue(finalState, mvValue.Value())
		}
	}
	
	finalState.Finalise(true, false)
}

func (s *Store) WriteLatestToStore(finalState *state.StateDB) {
	// sort the keys
	keys := []StorageKey{}
	s.multiVersionMap.Range(func(key, value interface{}) bool {
		keys = append(keys, key.(StorageKey))
		return true
	})
	sort.Slice(keys, func(i, j int) bool {
		return keys[i].String() < keys[j].String()
	})

	for _, key := range keys {
		val, ok := s.multiVersionMap.Load(key)
		if !ok {
			continue
		}
		mvValue, found := val.(MultiVersionValue).GetLatestNonEstimate()
		if !found {
			// this means that at some point, there was an estimate, but we have since removed it so there isn't anything writeable at the key, so we can skip
			continue
		}
		// we shouldn't have any ESTIMATE values when performing the write, because we read the latest non-estimate values only
		if mvValue.IsEstimate() {
			panic("should not have any estimate values when writing to parent store")
		}
		// if the value is deleted, then delete it from the parent store
		if mvValue.IsDeleted() {
			// We use []byte(key) instead of conv.UnsafeStrToBytes because we cannot
			// be sure if the underlying store might do a save with the byteslice or
			// not. Once we get confirmation that .Delete is guaranteed not to
			// save the byteslice, then we can assume only a read-only copy is sufficient.
			key.SetValue(finalState, common.Hash{})
			continue
		}
		if mvValue.Value() != (common.Hash{}) {
			key.SetValue(finalState, mvValue.Value())
		}
	}

	finalState.Finalise(true, false)
}
