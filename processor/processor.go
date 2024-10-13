package processor

import (
	"errors"

	"github.com/hyeonLewis/go-pevm/scheduler"
	"github.com/kaiachain/kaia/blockchain"
	"github.com/kaiachain/kaia/blockchain/state"
	"github.com/kaiachain/kaia/blockchain/types"
	"github.com/kaiachain/kaia/blockchain/vm"
)

type Processor struct {
	txs      []*types.Transaction
	bc       *blockchain.BlockChain
	header   *types.Header
	state    *state.StateDB
	vmConfig vm.Config

	scheduler *scheduler.Scheduler
}

var ErrNilArgument = errors.New("nil argument")

func NewProcessor(txs []*types.Transaction, bc *blockchain.BlockChain, header *types.Header, vmConfig vm.Config) (*Processor, error) {
	if txs == nil || bc == nil || header == nil {
		return nil, ErrNilArgument
	}
	// In actual blockchain, it processes txs in header, so we need to get the state of the parent block.
	// But in simulation, we assume that transactions are executed for a next block,
	// so we use the state of the current block.
	state, err := bc.StateAt(header.Root)
	if err != nil {
		return nil, err
	}

	scheduler := scheduler.NewScheduler(bc, txs)

	return &Processor{
		txs:       txs,
		bc:        bc,
		header:    header,
		state:     state,
		vmConfig:  vmConfig,
		scheduler: scheduler,
	}, nil
}

// func (p *Processor) Execute() (types.Receipts, []*types.Log, uint64, []*vm.InternalTxTrace, error)  {

// }

func (p *Processor) ProcessTx(txIndex int) (*types.Receipt, []*types.Log, uint64, error) {
	var (
		usedGas = new(uint64)
		tx      = p.txs[txIndex]
	)

	// Extract author from the header
	author, _ := p.bc.Engine().Author(p.header) // Ignore error, we're past header validation

	// Iterate over and process the individual transactions
	p.state.SetTxContext(tx.Hash(), p.header.Hash(), txIndex)
	receipt, _, err := p.bc.ApplyTransaction(p.bc.Config(), &author, p.state, p.header, tx, usedGas, &p.vmConfig)
	if err != nil {
		return nil, nil, 0, err
	}
	logs := receipt.Logs

	// Finalize the block, applying any consensus engine specific extras (e.g. block rewards)
	if _, err := p.bc.Engine().Finalize(p.bc, p.header, p.state, []*types.Transaction{tx}, []*types.Receipt{receipt}); err != nil {
		return nil, nil, 0, err
	}

	return receipt, logs, *usedGas, nil
}

func (p *Processor) ProcessTxsSequentially() (types.Receipts, []*types.Log, uint64, error) {
	var (
		receipts types.Receipts
		usedGas  = new(uint64)
		allLogs  []*types.Log
	)

	// Extract author from the header
	author, _ := p.bc.Engine().Author(p.header) // Ignore error, we're past header validation

	// Iterate over and process the individual transactions
	for i, tx := range p.txs {
		p.state.SetTxContext(tx.Hash(), p.header.Hash(), i)
		receipt, _, err := p.bc.ApplyTransaction(p.bc.Config(), &author, p.state, p.header, tx, usedGas, &p.vmConfig)
		if err != nil {
			return nil, nil, 0, err
		}
		receipts = append(receipts, receipt)
		allLogs = append(allLogs, receipt.Logs...)
	}

	// Finalize the block, applying any consensus engine specific extras (e.g. block rewards)
	if _, err := p.bc.Engine().Finalize(p.bc, p.header, p.state, p.txs, receipts); err != nil {
		return nil, nil, 0, err
	}

	return receipts, allLogs, *usedGas, nil
}
