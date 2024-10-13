package scheduler

import (
	"math/big"
	"runtime"
	"sync"
	"time"

	"github.com/kaiachain/kaia/blockchain"
	"github.com/kaiachain/kaia/blockchain/types"
	"github.com/kaiachain/kaia/blockchain/vm"
	"github.com/kaiachain/kaia/common"
	"github.com/kaiachain/kaia/consensus/misc"
	"github.com/kaiachain/kaia/log"
)

var logger = log.NewModuleLogger(log.Work)

type Scheduler struct {
	chain *blockchain.BlockChain

	tracers []*vm.CallTracer

	concurrencyLevel int
	batchSize        int

	threadWg sync.WaitGroup

	txs               []*types.Transaction
	txIndex           int
	processedTxs      []*types.Transaction
	processedReceipts map[int]*types.Receipt
	touchedAddrs        map[common.Address]struct{}

	done chan struct{}

	mu sync.RWMutex

	stateMu sync.RWMutex
}

func NewScheduler(chain *blockchain.BlockChain, txs []*types.Transaction) *Scheduler {
	concurrencyLevel := runtime.NumCPU()
	batchSize := (len(txs) + concurrencyLevel - 1) / concurrencyLevel

	return &Scheduler{
		chain:             chain,
		txs:               txs,
		concurrencyLevel:  concurrencyLevel,
		batchSize:         batchSize,
		processedTxs:      make([]*types.Transaction, 0),
		processedReceipts: make(map[int]*types.Receipt),
		touchedAddrs:        make(map[common.Address]struct{}),
		done:              make(chan struct{}),
	}
}

func (s *Scheduler) Schedule() (types.Receipts, []*types.Log, error) {
	// TODO: implement
	return nil, nil, nil
}

// postProcess merges parallelly processed states and usedGas
// Also sorts receipts and logs
func (s *Scheduler) postProcess() (types.Receipts, []*types.Log) {
	// TODO: implement
	receipts, logs := s.sortReceipts()

	return receipts, logs
}

func (s *Scheduler) sortReceipts() (types.Receipts, []*types.Log) {
	receipts := make(types.Receipts, 0, len(s.processedReceipts))
	logs := make([]*types.Log, 0, len(s.processedReceipts))
	for idx, receipt := range s.processedReceipts {
		receipts[idx] = receipt
	}
	for _, receipt := range receipts {
		logs = append(logs, receipt.Logs...)
	}

	return receipts, logs
}

func (s *Scheduler) prepareHeader() (*types.Header, error) {
	parent := s.chain.CurrentBlock()
	nextBlockNum := new(big.Int).Add(parent.Number(), common.Big1)
	header := &types.Header{
		ParentHash: parent.Hash(),
		Number:     nextBlockNum,
		Time:       big.NewInt(time.Now().Unix()),
	}

	if s.chain.Config().IsMagmaForkEnabled(nextBlockNum) {
		header.BaseFee = misc.NextMagmaBlockBaseFee(parent.Header(), s.chain.Config().Governance.KIP71)
	}
	if err := s.chain.Engine().Prepare(s.chain, header); err != nil {
		logger.Error("Failed to prepare header for mining", "err", err)
		return nil, err
	}

	return header, nil
}

func (s *Scheduler) preCheckParallelExecution() bool {
	addressSet := make(map[common.Address]byte, len(s.txs)*3)

	for _, tx := range s.txs {
		from, err := tx.From()
		if err != nil || addressSet[from]&1 != 0 {
			return false
		}
		addressSet[from] |= 1

		feePayer, err := tx.FeePayer()
		if err != nil || addressSet[feePayer]&2 != 0 {
			return false
		}
		addressSet[feePayer] |= 2

		to := tx.To()
		if to == nil || addressSet[*to]&4 != 0 {
			return false
		}
		addressSet[*to] |= 4
	}

	return true
}

func (s *Scheduler) setTouchedAddrs(touchedAddrs []common.Address) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, to := range touchedAddrs {
		s.touchedAddrs[to] = struct{}{}
	}
}

func (s *Scheduler) shouldRevert(touchedAddrs []common.Address) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, to := range touchedAddrs {
		if _, ok := s.touchedAddrs[to]; ok {
			return true
		}
	}

	return false
}

func getTouchedAddrs(result vm.CallFrame) []common.Address {
	touchedAddrs := make(map[common.Address]struct{})

	touchedAddrs[result.From] = struct{}{}
	if result.To != nil {
		touchedAddrs[*result.To] = struct{}{}
	}

	var traverse func(calls []vm.CallFrame)
	traverse = func(calls []vm.CallFrame) {
		for _, call := range calls {
			if call.To != nil {
				touchedAddrs[*call.To] = struct{}{}
			}
			traverse(call.Calls)
		}
	}

	traverse(result.Calls)

	uniquetouchedAddrs := make([]common.Address, 0, len(touchedAddrs))
	for addr := range touchedAddrs {
		uniquetouchedAddrs = append(uniquetouchedAddrs, addr)
	}

	return uniquetouchedAddrs
}
