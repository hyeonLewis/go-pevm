package processor

import (
	"math/big"
	"testing"

	"github.com/hyeonLewis/go-pevm/chain"
	"github.com/hyeonLewis/go-pevm/constants"
	"github.com/hyeonLewis/go-pevm/storage"
	"github.com/kaiachain/kaia/blockchain/types"
	"github.com/kaiachain/kaia/blockchain/vm"
	"github.com/stretchr/testify/assert"
)

func TestProcessNewTxs(t *testing.T) {
	db := storage.NewInMemoryStorage()
	storage.InjectGenesis(db)
	config := db.ReadChainConfig(db.ReadCanonicalHash(0))
	bc := chain.NewBlockchain(db, config)

	tx := types.NewTransaction(0, constants.RandomAddress, big.NewInt(5), 3000000, big.NewInt(25*1e9), nil)
	err := tx.Sign(types.NewEIP155Signer(config.ChainID), constants.ValidatorPrivateKey)
	assert.NoError(t, err)

	processor, err := NewProcessor([]*types.Transaction{tx}, bc, bc.CurrentBlock().Header(), vm.Config{})
	assert.NoError(t, err)

	receipts, _, usedGas, err := processor.ProcessNewTxs()
	assert.NoError(t, err)
	assert.Equal(t, receipts[0].Status, types.ReceiptStatusSuccessful)
	assert.Equal(t, usedGas, uint64(21000))
}
