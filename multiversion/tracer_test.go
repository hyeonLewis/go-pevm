package multiversion

import (
	"testing"

	"github.com/kaiachain/kaia/common"
	"github.com/stretchr/testify/require"
)

func TestVersionIndexedStoreGetters(t *testing.T) {
	parentState := prepareState()
	mvs := NewMultiVersionStore(parentState)
	// initialize a new VersionIndexedStore
	vis := NewAccessListTracer(mvs, 1, 2, make(chan Abort, 1))

	// mock a value in the parent store
	parentState.SetState(key1Addr, key1, value1)

	// read key that doesn't exist
	val := vis.Get(storageKey2)
	require.Equal(t, common.Hash{}, val)
	require.False(t, vis.Has(storageKey2))

	// read key that falls down to parent store
	val2 := vis.Get(storageKey1)
	require.Equal(t, value1, val2)
	require.True(t, vis.Has(storageKey1))
	// verify value now in readset
	require.Equal(t, []common.Hash{value1}, vis.GetReadset()[storageKey1])

	// read the same key that should now be served from the readset (can be verified by setting a different value for the key in the parent store)
	parentState.SetState(key1Addr, key1, value2) // realistically shouldn't happen, modifying to verify readset access
	val3 := vis.Get(storageKey1)
	require.True(t, vis.Has(storageKey1))
	require.Equal(t, value1, val3)

	// test deleted value written to MVS but not parent store
	mvs.SetWriteset(0, 2, map[StorageKey]common.Hash{
		ToStorageKey(key1Addr, common.BytesToHash([]byte("delKey"))): common.Hash{},
	})
	parentState.SetState(key1Addr, common.BytesToHash([]byte("delKey")), value4)
	valDel := vis.Get(ToStorageKey(key1Addr, common.BytesToHash([]byte("delKey"))))
	require.Equal(t, common.Hash{}, valDel)
	require.False(t, vis.Has(ToStorageKey(key1Addr, common.BytesToHash([]byte("delKey")))))

	// set different key in MVS - for various indices
	mvs.SetWriteset(0, 2, map[StorageKey]common.Hash{
		ToStorageKey(key1Addr, common.BytesToHash([]byte("delKey"))): common.Hash{},
		ToStorageKey(key1Addr, key3):                                 value3,
	})
	mvs.SetWriteset(2, 1, map[StorageKey]common.Hash{
		ToStorageKey(key1Addr, key3): value4,
	})
	mvs.SetEstimatedWriteset(5, 0, map[StorageKey]common.Hash{
		ToStorageKey(key1Addr, key3): common.Hash{},
	})

	// read the key that falls down to MVS
	val4 := vis.Get(ToStorageKey(key1Addr, key3))
	// should equal value3 because value4 is later than the key in question
	require.Equal(t, value3, val4)
	require.True(t, vis.Has(ToStorageKey(key1Addr, key3)))

	// try a read that falls through to MVS with a later tx index
	vis2 := NewAccessListTracer(mvs, 3, 2, make(chan Abort, 1))
	val5 := vis2.Get(ToStorageKey(key1Addr, key3))
	// should equal value3 because value4 is later than the key in question
	require.Equal(t, value4, val5)
	require.True(t, vis2.Has(ToStorageKey(key1Addr, key3)))

	// test estimate values writing to abortChannel
	abortChannel := make(chan Abort, 1)
	vis3 := NewAccessListTracer(mvs, 6, 2, abortChannel)
	require.Panics(t, func() {
		vis3.Get(ToStorageKey(key1Addr, key3))
	})
	abort := <-abortChannel // read the abort from the channel
	require.Equal(t, 5, abort.DependentTxIdx)
	// require.Equal(t, scheduler.ErrReadEstimate, abort.Err)

	vis.Set(ToStorageKey(key1Addr, key4), value4)
	// verify proper response for GET
	val6 := vis.Get(ToStorageKey(key1Addr, key4))
	require.True(t, vis.Has(ToStorageKey(key1Addr, key4)))
	require.Equal(t, value4, val6)
	// verify that its in the writeset
	require.Equal(t, value4, vis.GetWriteset()[ToStorageKey(key1Addr, key4)])
	// verify that its not in the readset
	require.Empty(t, vis.GetReadset()[ToStorageKey(key1Addr, key4)])
}

func TestVersionIndexedStoreSetters(t *testing.T) {
	parentState := prepareState()
	mvs := NewMultiVersionStore(parentState)
	// initialize a new VersionIndexedStore
	vis := NewAccessListTracer(mvs, 1, 2, make(chan Abort, 1))

	// test simple set
	vis.Set(storageKey1, value1)
	require.Equal(t, value1, vis.GetWriteset()[storageKey1])

	mvs.SetWriteset(0, 1, map[StorageKey]common.Hash{
		ToStorageKey(key1Addr, key2): value2,
	})
	vis.Delete(ToStorageKey(key1Addr, key2))
	require.Equal(t, common.Hash{}, vis.Get(ToStorageKey(key1Addr, key2)))
	// because the delete should be at the writeset level, we should not have populated the readset
	require.Zero(t, len(vis.GetReadset()))

	// try setting the value again, and then read
	vis.Set(ToStorageKey(key1Addr, key2), value3)
	require.Equal(t, value3, vis.Get(ToStorageKey(key1Addr, key2)))
	require.Zero(t, len(vis.GetReadset()))
}

func TestVersionIndexedStoreWriteEstimates(t *testing.T) {
	parentState := prepareState()
	mvs := NewMultiVersionStore(parentState)
	// initialize a new VersionIndexedStore
	vis := NewAccessListTracer(mvs, 1, 2, make(chan Abort, 1))

	mvs.SetWriteset(0, 1, map[StorageKey]common.Hash{
		storageKey3: value3,
	})

	require.False(t, mvs.Has(3, storageKey1))
	require.False(t, mvs.Has(3, storageKey2))
	require.True(t, mvs.Has(3, storageKey3))

	// write some keys
	vis.Set(storageKey1, value1)
	vis.Set(storageKey2, value2)
	vis.Delete(storageKey3)

	vis.WriteEstimatesToMultiVersionStore()

	require.True(t, mvs.GetLatest(storageKey1).IsEstimate())
	require.True(t, mvs.GetLatest(storageKey2).IsEstimate())
	require.True(t, mvs.GetLatest(storageKey3).IsEstimate())
}

func TestVersionIndexedStoreValidation(t *testing.T) {
	parentState := prepareState()
	mvs := NewMultiVersionStore(parentState)
	// initialize a new VersionIndexedStore
	abortC := make(chan Abort, 1)
	vis := NewAccessListTracer(mvs, 2, 2, abortC)
	// set some initial values
	parentState.SetState(key4Addr, key4, value4)
	parentState.SetState(key5Addr, key5, value5)
	parentState.SetState(deletedKeyAddr, common.BytesToHash([]byte("deletedKey")), common.BytesToHash([]byte("foo")))

	mvs.SetWriteset(0, 1, map[StorageKey]common.Hash{
		storageKey1: value1,
		storageKey2: value2,
		ToStorageKey(deletedKeyAddr, common.BytesToHash([]byte("deletedKey"))): common.Hash{},
	})

	// load those into readset
	vis.Get(storageKey1)
	vis.Get(storageKey2)
	vis.Get(storageKey4)
	vis.Get(storageKey5)
	vis.Get(ToStorageKey(keyDNEAddr, common.BytesToHash([]byte("keyDNE"))))
	vis.Get(ToStorageKey(deletedKeyAddr, common.BytesToHash([]byte("deletedKey"))))

	// everything checks out, so we should be able to validate successfully
	require.True(t, vis.ValidateReadset())
	// modify underlying transaction key that is unrelated
	mvs.SetWriteset(1, 1, map[StorageKey]common.Hash{
		storageKey3: value3,
	})
	// should still have valid readset
	require.True(t, vis.ValidateReadset())

	// modify underlying transaction key that is related
	mvs.SetWriteset(1, 1, map[StorageKey]common.Hash{
		storageKey3: value3,
		storageKey1: common.BytesToHash([]byte("value1_b")),
	})
	// should now have invalid readset
	require.False(t, vis.ValidateReadset())
	// reset so readset is valid again
	mvs.SetWriteset(1, 1, map[StorageKey]common.Hash{
		storageKey3: value3,
		storageKey1: value1,
	})
	require.True(t, vis.ValidateReadset())

	// mvs has a value that was initially read from parent
	mvs.SetWriteset(1, 1, map[StorageKey]common.Hash{
		storageKey3: value3,
		storageKey1: value1,
		storageKey4: common.BytesToHash([]byte("value4_b")),
	})
	require.False(t, vis.ValidateReadset())
	// reset key
	mvs.SetWriteset(1, 1, map[StorageKey]common.Hash{
		storageKey3: value3,
		storageKey1: value1,
		storageKey4: value4,
	})
	require.True(t, vis.ValidateReadset())

	// mvs has a value that was initially read from parent - BUT in a later tx index
	mvs.SetWriteset(4, 2, map[StorageKey]common.Hash{
		storageKey4: common.BytesToHash([]byte("value4_c")),
	})
	// readset should remain valid
	require.True(t, vis.ValidateReadset())

	// mvs has an estimate
	mvs.SetEstimatedWriteset(1, 1, map[StorageKey]common.Hash{
		storageKey2: common.Hash{},
	})
	// readset should be invalid now - but via abort channel write
	go func() {
		vis.ValidateReadset()
	}()
	abort := <-abortC // read the abort from the channel
	require.Equal(t, 1, abort.DependentTxIdx)

	// test key deleted later
	mvs.SetWriteset(1, 1, map[StorageKey]common.Hash{
		storageKey3: value3,
		storageKey1: value1,
		storageKey4: value4,
		storageKey2: common.Hash{},
	})
	require.False(t, vis.ValidateReadset())
	// reset key2
	mvs.SetWriteset(1, 1, map[StorageKey]common.Hash{
		storageKey3: value3,
		storageKey1: value1,
		storageKey4: value4,
		storageKey2: value2,
	})

	// lastly verify panic if parent kvstore has a conflict - this shouldn't happen but lets assert that it would panic
	parentState.SetState(keyDNEAddr, common.BytesToHash([]byte("keyDNE")), common.BytesToHash([]byte("foobar")))
	require.Equal(t, common.BytesToHash([]byte("foobar")), parentState.GetState(keyDNEAddr, common.BytesToHash([]byte("keyDNE"))))
	require.Panics(t, func() {
		vis.ValidateReadset()
	})
}
