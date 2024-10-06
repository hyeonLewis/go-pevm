package scheduler

import (
	"context"
	"math"
	"math/big"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/hyeonLewis/go-pevm/chain"
	"github.com/hyeonLewis/go-pevm/constants"
	"github.com/kaiachain/kaia/blockchain"
	"github.com/kaiachain/kaia/blockchain/state"
	"github.com/kaiachain/kaia/blockchain/types"
	"github.com/kaiachain/kaia/blockchain/vm"
	"github.com/kaiachain/kaia/common"
	"github.com/kaiachain/kaia/consensus/misc"
	"github.com/kaiachain/kaia/log"
)

var logger = log.NewModuleLogger(log.Work)


type SchedulerStatus uint64

const (
	timeout = 5 * time.Second
)

const (
	SchedulerStatusReadyToStart SchedulerStatus = iota
	SchedulerStatusRunning
	SchedulerStatusFinished
	SchedulerStatusReverted // need to re-execute sequentially
	SchedulerStatusFailed
)

type Scheduler struct {
	chain *blockchain.BlockChain

	states  []*state.StateDB
	tracers []*vm.CallTracer

	concurrencyLevel int
	batchSize        int

	threadWg sync.WaitGroup

	txs           []*types.Transaction
	txIndex       int
	processedTxs  []*types.Transaction
	processedLogs map[int][]*types.Log
	touchedTos    map[common.Address]struct{}

	status SchedulerStatus

	ctx    context.Context
	cancel context.CancelFunc
	done   chan struct{}

	mu sync.RWMutex

	stateMu sync.RWMutex
}

func NewScheduler(chain *blockchain.BlockChain, txs []*types.Transaction) *Scheduler {
	concurrencyLevel := runtime.NumCPU()
	batchSize := (len(txs) + concurrencyLevel - 1) / concurrencyLevel

	ctx, cancel := context.WithTimeout(context.Background(), timeout)

	return &Scheduler{
		chain:            chain,
		txs:              txs,
		concurrencyLevel: concurrencyLevel,
		batchSize:        batchSize,
		status:           SchedulerStatusReadyToStart,
		processedTxs:     make([]*types.Transaction, 0),
		processedLogs:    make(map[int][]*types.Log),
		touchedTos:       make(map[common.Address]struct{}),
		ctx:              ctx,
		cancel:           cancel,
		done:             make(chan struct{}),
	}
}

func (s *Scheduler) Peek() *types.Transaction {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.txIndex >= len(s.txs) {
		return nil
	}

	tx := s.txs[s.txIndex]
	s.txIndex++

	return tx
}

func (s *Scheduler) Start() SchedulerStatus {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.status = SchedulerStatusRunning
	s.threadWg.Add(s.concurrencyLevel)

	header, err := s.prepareHeader()
	if err != nil {
		return SchedulerStatusFailed
	}

	for i := 0; i < s.concurrencyLevel; i++ {
		go s.run(i, header)
	}

	go func() {
		s.threadWg.Wait()
		close(s.done)
	}()

	select {
	case <-s.ctx.Done():
		s.status = SchedulerStatusReverted
	case <-s.done:
		if s.status == SchedulerStatusRunning {
			s.status = SchedulerStatusFinished
		}
	}

	return s.status
}

func (s *Scheduler) run(i int, header *types.Header) {
	defer s.threadWg.Done()

	start := i * s.batchSize
	end := int(math.Min(float64(start+s.batchSize), float64(len(s.txs))))

	for start < end {
		select {
		case <-s.ctx.Done():
			return
		default:
		}

		if atomic.LoadUint64((*uint64)(&s.status)) == uint64(SchedulerStatusReverted) {
			break
		}

		tx := s.Peek()
		if tx == nil {
			break
		}

		// TODO: use PrunableStateAt
		state, _ := s.chain.State()
		task := chain.NewTask(s.chain.Config(), state, header)
		tracer := vm.NewCallTracer()

		s.stateMu.Lock()
		s.states = append(s.states, state)
		s.tracers = append(s.tracers, tracer)
		s.stateMu.Unlock()

		err, logs := task.CommitTransaction(tx, s.chain, constants.DefaultRewardBase, &vm.Config{
			Tracer: tracer,
		})
		
		s.mu.Lock()
		s.processedTxs = append(s.processedTxs, tx)
		s.processedLogs[len(s.processedTxs)-1] = logs
		s.mu.Unlock()

		// TODO: better error handling
		if err != nil {
			continue
		}

		result, err := tracer.GetResult()
		if err != nil {
			break
		}

		touchedTos := getTouchedTos(result)
		if s.shouldRevert(touchedTos) {
			atomic.StoreUint64((*uint64)(&s.status), uint64(SchedulerStatusReverted))
			break
		}
	}
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

func (s *Scheduler) setTouchedTos(touchedTos []common.Address) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, to := range touchedTos {
		s.touchedTos[to] = struct{}{}
	}
}

func (s *Scheduler) shouldRevert(touchedTos []common.Address) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, to := range touchedTos {
		if _, ok := s.touchedTos[to]; ok {
			return true
		}
	}

	return false
}

func getTouchedTos(result vm.CallFrame) []common.Address {
	touchedTos := make(map[common.Address]struct{})

	touchedTos[*result.To] = struct{}{}

	var traverse func(calls []vm.CallFrame)
	traverse = func(calls []vm.CallFrame) {
		for _, call := range calls {
			if call.To != nil {
				touchedTos[*call.To] = struct{}{}
			}
			traverse(call.Calls)
		}
	}

	traverse(result.Calls)

	uniqueTouchedTos := make([]common.Address, 0, len(touchedTos))
	for addr := range touchedTos {
		uniqueTouchedTos = append(uniqueTouchedTos, addr)
	}

	return uniqueTouchedTos
}
