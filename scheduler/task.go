package scheduler

import (
	"github.com/hyeonLewis/go-pevm/chain"
	"github.com/kaiachain/kaia/blockchain/types"
)

type ExecutionTask struct {
	Executor *chain.Executor
	TxIndex int
	Tx      *types.Transaction
}

type ValidationTask struct {
	TxIndex int
	Tx      *types.Transaction
}
