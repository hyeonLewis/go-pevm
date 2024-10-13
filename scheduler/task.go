package scheduler

import (
	"github.com/hyeonLewis/go-pevm/processor"
	"github.com/kaiachain/kaia/blockchain/types"
)

type ExecutionTask struct {
	Processor *processor.Processor
	Tx        *types.Transaction
	TxIndex   int

	resultCh chan *processor.Result
}

func NewExecutionTask(processor *processor.Processor, txIndex int, tx *types.Transaction, resultCh chan *processor.Result) *ExecutionTask {
	if processor == nil || tx == nil {
		return nil
	}
	return &ExecutionTask{Processor: processor, Tx: tx, TxIndex: txIndex, resultCh: resultCh}
}

func (t *ExecutionTask) Run(taskCh chan *ExecutionTask, cancelCh chan struct{}) {
	for {
		select {
		case <-cancelCh:
			return
		default:
			result, err := t.Processor.ProcessTx(t.TxIndex)
			if err != nil {
				t.resultCh <- result
			}
		}
	}
}

type ValidationResult struct {
	Valid bool
	Err   error
}

type ValidationTask struct {
	TxIndex int
	Tx      *types.Transaction
}

func NewValidationTask(txIndex int, tx *types.Transaction) *ValidationTask {
	return &ValidationTask{TxIndex: txIndex, Tx: tx}
}
