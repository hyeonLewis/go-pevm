package storage

import (
	"math/big"
	"testing"

	"github.com/hyeonLewis/go-pevm/chain"
	"github.com/kaiachain/kaia/common"
	"github.com/stretchr/testify/assert"
)

func TestNewInMemoryStorage(t *testing.T) {
	db := NewInMemoryStorage()
	assert.NotNil(t, db)
}

func TestInjectGenesis(t *testing.T) {
	db := NewInMemoryStorage()
	InjectGenesis(db)

	assert.NotNil(t, db)

	genesis, err := readGenesis("../data/genesis.json")
	assert.Nil(t, err)

	config := db.ReadChainConfig(db.ReadCanonicalHash(0))
	assert.Equal(t, genesis.Config, config)
}

func TestCommitPreState(t *testing.T) {
	db := NewInMemoryStorage()
	InjectGenesis(db)

	chain := chain.NewBlockchain(db, db.ReadChainConfig(db.ReadCanonicalHash(0)))
	state, _ := chain.State()
	CommitPreState("19932810", state)

	acc := state.GetAccount(common.HexToAddress("0x0000000000000000000000000000000000000001"))
	expectedBalance, _ := big.NewInt(0).SetString("313b2c58fcbd16c26", 16)
	assert.Equal(t, expectedBalance, acc.GetBalance())
}

func TestGetJournal(t *testing.T) {
	db := NewInMemoryStorage()
	InjectGenesis(db)

	chain := chain.NewBlockchain(db, db.ReadChainConfig(db.ReadCanonicalHash(0)))
	state, _ := chain.State()
	CommitPreState("19932810", state)

	state.SetBalance(common.HexToAddress("0x0000000000000000000000000000000000000001"), big.NewInt(1000000000000000000))
	state.SetBalance(common.HexToAddress("0x0000000000000000000000000000000000000001"), big.NewInt(3000000000000000000))
	journal := GetJournal(state)
	assert.Equal(t, 2, journal.Dirties()[common.HexToAddress("0x0000000000000000000000000000000000000001")])
}
