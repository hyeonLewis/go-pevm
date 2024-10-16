package scheduler

import (
	"testing"

	"github.com/hyeonLewis/go-pevm/chain"
	"github.com/hyeonLewis/go-pevm/storage"
	"github.com/kaiachain/kaia/blockchain"
	"github.com/kaiachain/kaia/blockchain/types"
	"github.com/kaiachain/kaia/blockchain/vm"
	"github.com/kaiachain/kaia/common"
	"github.com/kaiachain/kaia/params"
	"github.com/stretchr/testify/assert"
)

func generateCallFrame(depth int) vm.CallFrame {
	randomAddress := func() *common.Address {
		addr := common.BytesToAddress(common.MakeRandomBytes(20))
		return &addr
	}

	top := vm.CallFrame{
		From: common.Address{},
		To:   randomAddress(),
	}

	calls := make([]vm.CallFrame, 0, depth)
	// Total calls = 1 + depth + depth ^ 2 + 1 (from address)
	for i := 0; i < depth; i++ {
		calls = append(calls, vm.CallFrame{
			From: *randomAddress(),
			To:   randomAddress(),
		})

		calls[i].Calls = make([]vm.CallFrame, 0, depth)
		for j := 0; j < depth; j++ {
			calls[i].Calls = append(calls[i].Calls, vm.CallFrame{
				From: *randomAddress(),
				To:   randomAddress(),
			})
		}
	}

	top.Calls = calls

	return top
}

func TestGetTouchedAddrs(t *testing.T) {
	callFrame := generateCallFrame(10)
	touchedAddrs := getTouchedAddrs(callFrame)

	assert.Equal(t, len(touchedAddrs), 112)
}

func TestShouldRevert(t *testing.T) {
	callFrame := generateCallFrame(10)
	touchedAddrs := getTouchedAddrs(callFrame)

	chain := testBlockchain()
	scheduler := NewScheduler(chain, []*types.Transaction{})
	assert.False(t, scheduler.shouldRevert(touchedAddrs))

	scheduler.setTouchedAddrs(touchedAddrs)
	assert.True(t, scheduler.shouldRevert(touchedAddrs))
}

func testBlockchain() *blockchain.BlockChain {
	db := storage.NewInMemoryStorage()
	storage.InjectGenesis(db)
	config := params.MainnetChainConfig

	return chain.NewBlockchain(db, config)
}
