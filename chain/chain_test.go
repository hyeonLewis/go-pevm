package chain

import (
	"math/big"
	"testing"

	"github.com/hyeonLewis/go-pevm/constants"
	"github.com/hyeonLewis/go-pevm/storage"
	"github.com/kaiachain/kaia/blockchain/types"
	"github.com/kaiachain/kaia/blockchain/vm"
	"github.com/kaiachain/kaia/params"
	"github.com/stretchr/testify/assert"
)

func TestNewBlockchain(t *testing.T) {
	db := storage.NewInMemoryStorage()
	storage.InjectGenesis(db)
	config := params.MainnetChainConfig
	chain := NewBlockchain(db, config)

	assert.NotNil(t, chain)
	assert.Equal(t, chain.CurrentBlock().NumberU64(), uint64(0))
}

func TestNewTask(t *testing.T) {
	db := storage.NewInMemoryStorage()
	storage.InjectGenesis(db)
	config := params.MainnetChainConfig
	chain := NewBlockchain(db, config)

	state, err := chain.State()
	assert.NoError(t, err)

	task := NewTask(config, state, chain.CurrentBlock().Header())
	assert.NotNil(t, task)
}

func TestTask_CommitTransaction(t *testing.T) {
	db := storage.NewInMemoryStorage()
	storage.InjectGenesis(db)
	config := params.MainnetChainConfig
	chain := NewBlockchain(db, config)

	state, err := chain.State()
	assert.NoError(t, err)

	tx := types.NewTransaction(0, constants.RandomAddress, big.NewInt(5), 3000000, big.NewInt(25*1e9), nil)
	err = tx.Sign(types.NewEIP155Signer(config.ChainID), constants.ValidatorPrivateKey)
	assert.NoError(t, err)

	task := NewTask(config, state, chain.CurrentBlock().Header())

	err, _ = task.CommitTransaction(tx, chain, constants.DefaultRewardBase, &vm.Config{})
	assert.NoError(t, err)

	balance := state.GetBalance(constants.RandomAddress)
	assert.Equal(t, balance, big.NewInt(5))
}
