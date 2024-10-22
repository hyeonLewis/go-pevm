package processor

import (
	"errors"

	"github.com/hyeonLewis/go-pevm/scheduler"
	"github.com/kaiachain/kaia/blockchain"
	"github.com/kaiachain/kaia/blockchain/state"
	"github.com/kaiachain/kaia/blockchain/types"
	"github.com/kaiachain/kaia/blockchain/vm"
	"github.com/kaiachain/kaia/params"
)

type Processor struct {
	txs     []*types.Transaction
	bc      *blockchain.BlockChain
	header  *types.Header
	state   *state.StateDB
	workers int
}

type Result struct {
	Receipt *types.Receipt
	Logs    []*types.Log
	UsedGas uint64
	Trace   *vm.InternalTxTrace
}

var ErrNilArgument = errors.New("nil argument")

func NewProcessor(txs []*types.Transaction, bc *blockchain.BlockChain, header *types.Header, state *state.StateDB, workers int) (*Processor, error) {
	if txs == nil || bc == nil || header == nil {
		return nil, ErrNilArgument
	}

	return &Processor{
		txs:     txs,
		bc:      bc,
		header:  header,
		state:   state,
		workers: workers,
	}, nil
}

func (p *Processor) Execute() []*scheduler.Response {
	if len(p.txs) == 0 {
		return nil
	}

	deliverTxEntries := make([]*scheduler.DeliverTxEntry, 0, len(p.txs))
	for idx, tx := range p.txs {
		deliverTxEntries = append(deliverTxEntries, &scheduler.DeliverTxEntry{Tx: tx, AbsoluteIndex: idx})
	}

	// If workers is set to 0, it will create goroutines equal to the number of txs by default
	scheduler := scheduler.NewScheduler(p.state, p.bc.Config(), p.header, p.workers, p.deliverTx)
	resp, err := scheduler.ProcessAll(deliverTxEntries)
	if err != nil {
		panic(err)
	}

	return resp
}

func (p *Processor) deliverTx(config *params.ChainConfig, header *types.Header, task *scheduler.DeliverTxTask) (*types.Receipt, *vm.InternalTxTrace, error) {
	snap := task.State.Snapshot()
	author, _ := p.bc.Engine().Author(header)
	p.state.SetTxContext(task.Tx.Hash(), header.Hash(), task.AbsoluteIndex)

	vmConfig := &vm.Config{
		Debug:  true,
		Tracer: task.VersionStores,
	}

	receipt, trace, err := p.bc.ApplyTransaction(config, &author, task.State, header, task.Tx, task.UsedGas, vmConfig)
	if err != nil {
		task.State.RevertToSnapshot(snap)
		return nil, nil, err
	}

	return receipt, trace, nil
}
