package multiversion

import (
	"testing"

	"github.com/hyeonLewis/go-pevm/chain"
	"github.com/hyeonLewis/go-pevm/storage"
	"github.com/kaiachain/kaia/blockchain/state"
	"github.com/kaiachain/kaia/common"
	"github.com/stretchr/testify/require"
)

func TestMultiVersionStore(t *testing.T) {
	store := NewMultiVersionStore(nil)

	// Test Set and GetLatest
	store.SetWriteset(1, 1, map[StorageKey]common.Hash{
		StorageKey("key1"): value1,
	})
	store.SetWriteset(2, 1, map[StorageKey]common.Hash{
		StorageKey("key1"): value2,
	})
	store.SetWriteset(3, 1, map[StorageKey]common.Hash{
		StorageKey("key2"): value3,
	})

	require.Equal(t, value2, store.GetLatest(StorageKey("key1")).Value())
	require.Equal(t, value3, store.GetLatest(StorageKey("key2")).Value())

	// Test SetEstimate
	store.SetEstimatedWriteset(4, 1, map[StorageKey]common.Hash{
		StorageKey("key1"): {},
	})
	require.True(t, store.GetLatest(StorageKey("key1")).IsEstimate())

	// Test Delete
	store.SetWriteset(5, 1, map[StorageKey]common.Hash{
		StorageKey("key1"): {},
	})
	require.True(t, store.GetLatest(StorageKey("key1")).IsDeleted())

	// Test GetLatestBeforeIndex
	store.SetWriteset(6, 1, map[StorageKey]common.Hash{
		StorageKey("key1"): value4,
	})
	require.True(t, store.GetLatestBeforeIndex(5, StorageKey("key1")).IsEstimate())
	require.Equal(t, value4, store.GetLatestBeforeIndex(7, StorageKey("key1")).Value())

	// Test Has
	require.True(t, store.Has(2, StorageKey("key1")))
	require.False(t, store.Has(0, StorageKey("key1")))
	require.False(t, store.Has(5, StorageKey("key4")))
}

func TestMultiVersionStoreHasLaterValue(t *testing.T) {
	store := NewMultiVersionStore(nil)

	store.SetWriteset(5, 1, map[StorageKey]common.Hash{
		StorageKey("key1"): value2,
	})

	require.Nil(t, store.GetLatestBeforeIndex(4, StorageKey("key1")))
	require.Equal(t, value2, store.GetLatestBeforeIndex(6, StorageKey("key1")).Value())
}

func TestMultiVersionStoreKeyDNE(t *testing.T) {
	store := NewMultiVersionStore(nil)

	require.Nil(t, store.GetLatest(StorageKey("key1")))
	require.Nil(t, store.GetLatestBeforeIndex(0, StorageKey("key1")))
	require.False(t, store.Has(0, StorageKey("key1")))
}

func prepareState() *state.StateDB {
	db := storage.NewInMemoryStorage()
	storage.InjectGenesis(db)

	chain := chain.NewBlockchain(db, db.ReadChainConfig(db.ReadCanonicalHash(0)))
	state, _ := chain.State()
	return state
}

var (
	key1Addr       = common.HexToAddress("0x0000000000000000000000000000000000000001")
	key2Addr       = common.HexToAddress("0x0000000000000000000000000000000000000002")
	key3Addr       = common.HexToAddress("0x0000000000000000000000000000000000000003")
	key4Addr       = common.HexToAddress("0x0000000000000000000000000000000000000004")
	key5Addr       = common.HexToAddress("0x0000000000000000000000000000000000000005")
	key6Addr       = common.HexToAddress("0x0000000000000000000000000000000000000006")
	keyDNEAddr     = common.HexToAddress("0x0000000000000000000000000000000000000007")
	deletedKeyAddr = common.HexToAddress("0x0000000000000000000000000000000000000008")

	key1 = common.BytesToHash([]byte("key1"))
	key2 = common.BytesToHash([]byte("key2"))
	key3 = common.BytesToHash([]byte("key3"))
	key4 = common.BytesToHash([]byte("key4"))
	key5 = common.BytesToHash([]byte("key5"))

	storageKey1 = ToStorageKey(key1Addr, key1)
	storageKey2 = ToStorageKey(key2Addr, key2)
	storageKey3 = ToStorageKey(key3Addr, key3)
	storageKey4 = ToStorageKey(key4Addr, key4)
	storageKey5 = ToStorageKey(key5Addr, key5)

	value0 = common.BytesToHash([]byte("value0"))
	value1 = common.BytesToHash([]byte("value1"))
	value2 = common.BytesToHash([]byte("value2"))
	value3 = common.BytesToHash([]byte("value3"))
	value4 = common.BytesToHash([]byte("value4"))
	value5 = common.BytesToHash([]byte("value5"))
	value6 = common.BytesToHash([]byte("value6"))
)

func TestMultiVersionStoreWriteToParent(t *testing.T) {
	state := prepareState()
	mvs := NewMultiVersionStore(state)

	state.SetState(key2Addr, key2, value0)
	state.SetState(key4Addr, key4, value4)

	mvs.SetWriteset(1, 1, map[StorageKey]common.Hash{
		storageKey1: value1,
		storageKey3: {},
		storageKey4: {},
	})
	mvs.SetWriteset(2, 1, map[StorageKey]common.Hash{
		storageKey1: value2,
	})
	mvs.SetWriteset(3, 1, map[StorageKey]common.Hash{
		storageKey2: value3,
	})

	mvs.WriteLatestToStore(state)

	// assert state in parent store
	require.Equal(t, value2, state.GetState(key1Addr, key1))
	require.Equal(t, value3, state.GetState(key2Addr, key2))
	require.False(t, storage.Has(state, key3Addr, key3))
	require.False(t, storage.Has(state, key4Addr, key4))

	// verify no-op if mvs contains ESTIMATE
	mvs.SetEstimatedWriteset(1, 2, map[StorageKey]common.Hash{
		storageKey1: value1,
		storageKey3: {},
		storageKey4: {},
		storageKey5: {},
	})
	mvs.WriteLatestToStore(state)
	require.False(t, storage.Has(state, key5Addr, key5))
}

func TestMultiVersionStoreWritesetSetAndInvalidate(t *testing.T) {
	mvs := NewMultiVersionStore(nil)

	writeset := make(map[StorageKey]common.Hash)
	writeset[storageKey1] = value1
	writeset[storageKey2] = value2
	writeset[storageKey3] = common.Hash{}

	mvs.SetWriteset(1, 2, writeset)
	require.Equal(t, value1, mvs.GetLatest(storageKey1).Value())
	require.Equal(t, value2, mvs.GetLatest(storageKey2).Value())
	require.True(t, mvs.GetLatest(storageKey3).IsDeleted())

	writeset2 := make(map[StorageKey]common.Hash)
	writeset2[storageKey1] = value3

	mvs.SetWriteset(2, 1, writeset2)
	require.Equal(t, value3, mvs.GetLatest(storageKey1).Value())

	// invalidate writeset1
	mvs.InvalidateWriteset(1, 2)

	// verify estimates
	require.True(t, mvs.GetLatestBeforeIndex(2, storageKey1).IsEstimate())
	require.True(t, mvs.GetLatestBeforeIndex(2, storageKey2).IsEstimate())
	require.True(t, mvs.GetLatestBeforeIndex(2, storageKey3).IsEstimate())

	// third writeset
	writeset3 := make(map[StorageKey]common.Hash)
	writeset3[storageKey4] = common.BytesToHash([]byte("foo"))
	writeset3[storageKey5] = common.Hash{}

	// write the writeset directly as estimate
	mvs.SetEstimatedWriteset(3, 1, writeset3)

	require.True(t, mvs.GetLatest(storageKey4).IsEstimate())
	require.True(t, mvs.GetLatest(storageKey5).IsEstimate())

	// try replacing writeset1 to verify old keys removed
	writeset1_b := make(map[StorageKey]common.Hash)
	writeset1_b[storageKey1] = value4

	mvs.SetWriteset(1, 2, writeset1_b)
	require.Equal(t, value4, mvs.GetLatestBeforeIndex(2, storageKey1).Value())
	require.Nil(t, mvs.GetLatestBeforeIndex(2, storageKey2))
	// verify that GetLatest for key3 returns nil - because of removal from writeset
	require.Nil(t, mvs.GetLatest(storageKey3))

	// verify output for GetAllWritesetKeys
	writesetKeys := mvs.GetAllWritesetKeys()
	// we have 3 writesets
	require.Equal(t, 3, len(writesetKeys))
	require.Equal(t, []StorageKey{storageKey1}, writesetKeys[1])
	require.Equal(t, []StorageKey{storageKey1}, writesetKeys[2])
	require.Equal(t, []StorageKey{storageKey4, storageKey5}, writesetKeys[3])
}

func TestMultiVersionStoreValidateState(t *testing.T) {
	state := prepareState()
	mvs := NewMultiVersionStore(state)

	state.SetState(key2Addr, key2, value0)
	state.SetState(key3Addr, key3, value3)
	state.SetState(key4Addr, key4, value4)
	state.SetState(key5Addr, key5, value5)

	writeset := make(WriteSet)
	writeset[storageKey1] = value1
	writeset[storageKey2] = value2
	writeset[storageKey3] = common.Hash{}
	mvs.SetWriteset(1, 2, writeset)

	readset := make(ReadSet)
	readset[storageKey1] = []common.Hash{value1}
	readset[storageKey2] = []common.Hash{value2}
	readset[storageKey3] = []common.Hash{common.Hash{}}
	readset[storageKey4] = []common.Hash{value4}
	readset[storageKey5] = []common.Hash{value5}
	mvs.SetReadset(5, readset)

	// assert no readset is valid
	valid, conflicts := mvs.ValidateTransactionState(4)
	require.True(t, valid)
	require.Empty(t, conflicts)

	// assert readset index 5 is valid
	valid, conflicts = mvs.ValidateTransactionState(5)
	require.True(t, valid)
	require.Empty(t, conflicts)

	// introduce conflict
	mvs.SetWriteset(2, 1, map[StorageKey]common.Hash{
		storageKey3: value6,
	})

	// expect failure with conflict of tx 2
	valid, conflicts = mvs.ValidateTransactionState(5)
	require.False(t, valid)
	require.Equal(t, []int{2}, conflicts)

	// add a conflict due to deletion
	mvs.SetWriteset(3, 1, map[StorageKey]common.Hash{
		storageKey1: common.Hash{},
	})

	// expect failure with conflict of tx 2 and 3
	valid, conflicts = mvs.ValidateTransactionState(5)
	require.False(t, valid)
	require.Equal(t, []int{2, 3}, conflicts)

	// add a conflict due to estimate
	mvs.SetEstimatedWriteset(4, 1, map[StorageKey]common.Hash{
		storageKey2: common.BytesToHash([]byte("test")),
	})

	// expect index 4 to be returned
	valid, conflicts = mvs.ValidateTransactionState(5)
	require.False(t, valid)
	require.Equal(t, []int{2, 3, 4}, conflicts)
}

func TestMultiVersionStoreParentValidationMismatch(t *testing.T) {
	parentState := prepareState()
	mvs := NewMultiVersionStore(parentState)

	parentState.SetState(key2Addr, key2, value0)
	parentState.SetState(key3Addr, key3, value3)
	parentState.SetState(key4Addr, key4, value4)
	parentState.SetState(key5Addr, key5, value5)

	writeset := make(WriteSet)
	writeset[storageKey1] = value1
	writeset[storageKey2] = value2
	writeset[storageKey3] = common.Hash{}
	mvs.SetWriteset(1, 2, writeset)

	readset := make(ReadSet)
	readset[storageKey1] = []common.Hash{value1}
	readset[storageKey2] = []common.Hash{value2}
	readset[storageKey3] = []common.Hash{common.Hash{}}
	readset[storageKey4] = []common.Hash{value4}
	readset[storageKey5] = []common.Hash{value5}
	mvs.SetReadset(5, readset)

	// assert no readset is valid
	valid, conflicts := mvs.ValidateTransactionState(4)
	require.True(t, valid)
	require.Empty(t, conflicts)

	// assert readset index 5 is valid
	valid, conflicts = mvs.ValidateTransactionState(5)
	require.True(t, valid)
	require.Empty(t, conflicts)

	// overwrite tx writeset for tx1 - no longer writes key1
	writeset2 := make(WriteSet)
	writeset2[storageKey2] = value2
	writeset2[storageKey3] = common.Hash{}
	mvs.SetWriteset(1, 3, writeset2)

	// assert readset index 5 is invalid - because of mismatch with parent store
	valid, conflicts = mvs.ValidateTransactionState(5)
	require.False(t, valid)
	require.Empty(t, conflicts)
}

func TestMultiVersionStoreMultipleReadsetValueValidationFailure(t *testing.T) {
	parentState := prepareState()
	mvs := NewMultiVersionStore(parentState)

	parentState.SetState(key2Addr, key2, value0)
	parentState.SetState(key3Addr, key3, value3)
	parentState.SetState(key4Addr, key4, value4)
	parentState.SetState(key5Addr, key5, value5)

	writeset := make(WriteSet)
	writeset[storageKey1] = value1
	writeset[storageKey2] = value2
	writeset[storageKey3] = common.Hash{}
	mvs.SetWriteset(1, 2, writeset)

	readset := make(ReadSet)
	readset[storageKey1] = []common.Hash{value1}
	readset[storageKey2] = []common.Hash{value2}
	readset[storageKey3] = []common.Hash{common.Hash{}}
	readset[storageKey4] = []common.Hash{value4}
	readset[storageKey5] = []common.Hash{value5, common.BytesToHash([]byte("value5b"))}
	mvs.SetReadset(5, readset)

	// assert readset index 5 is invalid due to multiple values in readset
	valid, conflicts := mvs.ValidateTransactionState(5)
	require.False(t, valid)
	require.Empty(t, conflicts)
}

func TestMVSValidationWithOnlyEstimate(t *testing.T) {
	parentState := prepareState()
	mvs := NewMultiVersionStore(parentState)

	parentState.SetState(key2Addr, key2, value0)
	parentState.SetState(key3Addr, key3, value3)
	parentState.SetState(key4Addr, key4, value4)
	parentState.SetState(key5Addr, key5, value5)

	writeset := make(WriteSet)
	writeset[storageKey1] = value1
	writeset[storageKey2] = value2
	writeset[storageKey3] = common.Hash{}
	mvs.SetWriteset(1, 2, writeset)

	readset := make(ReadSet)
	readset[storageKey1] = []common.Hash{value1}
	readset[storageKey2] = []common.Hash{value2}
	readset[storageKey3] = []common.Hash{common.Hash{}}
	readset[storageKey4] = []common.Hash{value4}
	readset[storageKey5] = []common.Hash{value5}
	mvs.SetReadset(5, readset)

	// add a conflict due to estimate
	mvs.SetEstimatedWriteset(4, 1, map[StorageKey]common.Hash{
		storageKey2: common.BytesToHash([]byte("test")),
	})

	valid, conflicts := mvs.ValidateTransactionState(5)
	require.True(t, valid)
	require.Equal(t, []int{4}, conflicts)

}
