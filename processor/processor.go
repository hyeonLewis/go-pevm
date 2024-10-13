package processor

import (
	"errors"

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
}

type Result struct {
	Receipt *types.Receipt
	Logs    []*types.Log
	UsedGas uint64
	Trace   *vm.InternalTxTrace
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

	return &Processor{
		txs:      txs,
		bc:       bc,
		header:   header,
		state:    state,
		vmConfig: vmConfig,
	}, nil
}

func (p *Processor) ProcessTx(txIndex int) (*Result, error) {
	var (
		usedGas = new(uint64)
		tx      = p.txs[txIndex]
	)

	// Extract author from the header
	author, _ := p.bc.Engine().Author(p.header) // Ignore error, we're past header validation

	// Iterate over and process the individual transactions
	p.state.SetTxContext(tx.Hash(), p.header.Hash(), txIndex)
	receipt, trace, err := p.bc.ApplyTransaction(p.bc.Config(), &author, p.state, p.header, tx, usedGas, &p.vmConfig)
	if err != nil {
		return nil, err
	}
	logs := receipt.Logs

	// Finalize the block, applying any consensus engine specific extras (e.g. block rewards)
	if _, err := p.bc.Engine().Finalize(p.bc, p.header, p.state, []*types.Transaction{tx}, []*types.Receipt{receipt}); err != nil {
		return nil, err
	}

	return &Result{Receipt: receipt, Logs: logs, UsedGas: *usedGas, Trace: trace}, nil
}

func (p *Processor) ProcessTxsSequentially() (types.Receipts, []*types.Log, uint64, []*vm.InternalTxTrace, error) {
	var (
		receipts         types.Receipts
		usedGas          = new(uint64)
		allLogs          []*types.Log
		internalTxTraces []*vm.InternalTxTrace
	)

	// Extract author from the header
	author, _ := p.bc.Engine().Author(p.header) // Ignore error, we're past header validation

	// Iterate over and process the individual transactions
	for i, tx := range p.txs {
		p.state.SetTxContext(tx.Hash(), p.header.Hash(), i)
		receipt, trace, err := p.bc.ApplyTransaction(p.bc.Config(), &author, p.state, p.header, tx, usedGas, &p.vmConfig)
		if err != nil {
			return nil, nil, 0, nil, err
		}
		receipts = append(receipts, receipt)
		allLogs = append(allLogs, receipt.Logs...)
		internalTxTraces = append(internalTxTraces, trace)
	}

	// Finalize the block, applying any consensus engine specific extras (e.g. block rewards)
	if _, err := p.bc.Engine().Finalize(p.bc, p.header, p.state, p.txs, receipts); err != nil {
		return nil, nil, 0, nil, err
	}

	return receipts, allLogs, *usedGas, internalTxTraces, nil
}
