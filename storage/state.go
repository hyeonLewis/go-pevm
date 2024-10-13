package storage

import (
	"reflect"
	"unsafe"

	"github.com/kaiachain/kaia/blockchain/state"
	"github.com/kaiachain/kaia/common"
)

type journalEntry interface {
	revert(*state.StateDB)

	dirtied() *common.Address
}

type Journal struct {
	entries []journalEntry         // Current changes tracked by the journal
	dirties map[common.Address]int // Dirty accounts and the number of changes
}

// GetJournal returns the journal of the StateDB.
// Only for internal use.
func GetJournal(s *state.StateDB) *Journal {
	// Create a new Journal instance
	rs := reflect.ValueOf(s).Elem()
	rf := rs.Field(21)
	rf = reflect.NewAt(rf.Type(), unsafe.Pointer(rf.UnsafeAddr())).Elem()

	entries := reflect.Indirect(rf).FieldByName("entries")
	dirties := reflect.Indirect(rf).FieldByName("dirties")

	// Unsafe operation to force access to unexported fields
	entriesPtr := unsafe.Pointer(entries.UnsafeAddr())
	dirtiesPtr := unsafe.Pointer(dirties.UnsafeAddr())

	journal := &Journal{
		entries: *(*[]journalEntry)(entriesPtr),
		dirties: *(*map[common.Address]int)(dirtiesPtr),
	}
	return journal
}

func (j *Journal) Entries() []journalEntry {
	return j.entries
}

func (j *Journal) Dirties() map[common.Address]int {
	return j.dirties
}

func (j *Journal) GetDirties() []common.Address {
	if j.dirties == nil {
		return nil
	}

	uniqueDirties := make([]common.Address, 0, len(j.dirties))
	for address := range j.dirties {
		uniqueDirties = append(uniqueDirties, address)
	}
	return uniqueDirties
}
