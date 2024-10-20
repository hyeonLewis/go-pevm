package scheduler

import (
	"context"
	"sort"
	"sync"

	"github.com/hyeonLewis/go-pevm/multiversion"
	"github.com/kaiachain/kaia/blockchain/state"
	"github.com/kaiachain/kaia/blockchain/types"
	"github.com/kaiachain/kaia/blockchain/vm"
	"github.com/kaiachain/kaia/log"
	"github.com/kaiachain/kaia/params"
)

var logger = log.NewModuleLogger(60)

type status string

const (
	// statusPending tasks are ready for execution
	// all executing tasks are in pending state
	statusPending status = "pending"
	// statusExecuted tasks are ready for validation
	// these tasks did not abort during execution
	statusExecuted status = "executed"
	// statusAborted means the task has been aborted
	// these tasks transition to pending upon next execution
	statusAborted status = "aborted"
	// statusValidated means the task has been validated
	// tasks in this status can be reset if an earlier task fails validation
	statusValidated status = "validated"
	// statusWaiting tasks are waiting for another tx to complete
	statusWaiting status = "waiting"
	// maximumIterations before we revert to sequential (for high conflict rates)
	maximumIterations = 10
)

type Response struct {
	Receipt *types.Receipt
	Trace   *vm.InternalTxTrace
	err     error
}

type DeliverTxEntry struct {
	Tx            *types.Transaction
	AbsoluteIndex int
}

type DeliverTxBatchRequest struct {
	TxEntries []*DeliverTxEntry
}

type deliverTxTask struct {
	AbortCh chan multiversion.Abort

	mx            sync.RWMutex
	Status        status
	Dependencies  map[int]struct{}
	Abort         *multiversion.Abort
	Incarnation   int
	Tx            *types.Transaction
	AbsoluteIndex int
	Response      *Response
	UsedGas       *uint64

	VersionStores *multiversion.AccessListTracer
}

// AppendDependencies appends the given indexes to the task's dependencies
func (dt *deliverTxTask) AppendDependencies(deps []int) {
	dt.mx.Lock()
	defer dt.mx.Unlock()
	for _, taskIdx := range deps {
		dt.Dependencies[taskIdx] = struct{}{}
	}
}

func (dt *deliverTxTask) IsStatus(s status) bool {
	dt.mx.RLock()
	defer dt.mx.RUnlock()
	return dt.Status == s
}

func (dt *deliverTxTask) SetStatus(s status) {
	dt.mx.Lock()
	defer dt.mx.Unlock()
	dt.Status = s
}

func (dt *deliverTxTask) Reset() {
	dt.SetStatus(statusPending)
	dt.Response = nil
	dt.Abort = nil
	dt.AbortCh = nil
	dt.VersionStores = nil
}

func (dt *deliverTxTask) Increment() {
	dt.Incarnation++
}

// Scheduler processes tasks concurrently
type Scheduler interface {
	ProcessAll(reqs []*DeliverTxEntry) ([]*Response, error)
}

type scheduler struct {
	deliverTx          func(config *params.ChainConfig, state *state.StateDB, header *types.Header, tx *types.Transaction, usedGas *uint64, vmConfig *vm.Config) (*types.Receipt, *vm.InternalTxTrace, error)
	workers            int
	multiVersionStores multiversion.MultiVersionStore
	allTasksMap        map[int]*deliverTxTask
	allTasks           []*deliverTxTask
	executeCh          chan func()
	validateCh         chan func()
	synchronous        bool // true if maxIncarnation exceeds threshold
	maxIncarnation     int  // current highest incarnation

	state       *state.StateDB
	chainConfig *params.ChainConfig
	header      *types.Header
}

type deliverTxFunc func(config *params.ChainConfig, state *state.StateDB, header *types.Header, tx *types.Transaction, usedGas *uint64, vmConfig *vm.Config) (*types.Receipt, *vm.InternalTxTrace, error)

// NewScheduler creates a new scheduler
func NewScheduler(state *state.StateDB, chainConfig *params.ChainConfig, header *types.Header, workers int, deliverTxFunc deliverTxFunc) Scheduler {
	return &scheduler{
		workers:     workers,
		deliverTx:   deliverTxFunc,
		state:       state,
		chainConfig: chainConfig,
		header:      header,
	}
}

func (s *scheduler) collectResponses(tasks []*deliverTxTask) []*Response {
	res := make([]*Response, 0, len(tasks))
	for _, t := range tasks {
		res = append(res, t.Response)
	}
	return res
}

func (s *scheduler) invalidateTask(task *deliverTxTask) {

	s.multiVersionStores.InvalidateWriteset(task.AbsoluteIndex, task.Incarnation)
	s.multiVersionStores.ClearReadset(task.AbsoluteIndex)
}

func start(ctx context.Context, ch chan func(), workers int) {
	for i := 0; i < workers; i++ {
		go func() {
			for {
				select {
				case <-ctx.Done():
					return
				case work := <-ch:
					work()
				}
			}
		}()
	}
}

func (s *scheduler) DoValidate(work func()) {
	if s.synchronous {
		work()
		return
	}
	s.validateCh <- work
}

func (s *scheduler) DoExecute(work func()) {
	if s.synchronous {
		work()
		return
	}
	s.executeCh <- work
}

func (s *scheduler) findConflicts(task *deliverTxTask) (bool, []int) {
	var conflicts []int
	uniq := make(map[int]struct{})
	valid := true
	ok, mvConflicts := s.multiVersionStores.ValidateTransactionState(task.AbsoluteIndex)
	for _, c := range mvConflicts {
		if _, ok := uniq[c]; !ok {
			conflicts = append(conflicts, c)
			uniq[c] = struct{}{}
		}
	}
	// any non-ok value makes valid false
	valid = valid && ok

	sort.Ints(conflicts)
	return valid, conflicts
}

func toTasks(reqs []*DeliverTxEntry) ([]*deliverTxTask, map[int]*deliverTxTask) {
	tasksMap := make(map[int]*deliverTxTask)
	allTasks := make([]*deliverTxTask, 0, len(reqs))
	for _, r := range reqs {
		task := &deliverTxTask{
			Tx:            r.Tx,
			AbsoluteIndex: r.AbsoluteIndex,
			Status:        statusPending,
			Dependencies:  map[int]struct{}{},
		}

		tasksMap[r.AbsoluteIndex] = task
		allTasks = append(allTasks, task)
	}
	return allTasks, tasksMap
}

func (s *scheduler) initMultiVersionStores() {
	if s.multiVersionStores != nil {
		return
	}
	mvs := multiversion.NewMultiVersionStore(s.state)
	s.multiVersionStores = mvs
}

func dependenciesValidated(tasksMap map[int]*deliverTxTask, deps map[int]struct{}) bool {
	for i := range deps {
		// because idx contains absoluteIndices, we need to fetch from map
		task := tasksMap[i]
		if !task.IsStatus(statusValidated) {
			return false
		}
	}
	return true
}

func filterTasks(tasks []*deliverTxTask, filter func(*deliverTxTask) bool) []*deliverTxTask {
	var res []*deliverTxTask
	for _, t := range tasks {
		if filter(t) {
			res = append(res, t)
		}
	}
	return res
}

func allValidated(tasks []*deliverTxTask) bool {
	for _, t := range tasks {
		if !t.IsStatus(statusValidated) {
			return false
		}
	}
	return true
}

func (s *scheduler) ProcessAll(reqs []*DeliverTxEntry) ([]*Response, error) {
	var iterations int
	ctx := context.Background()
	// initialize mutli-version stores if they haven't been initialized yet
	s.initMultiVersionStores()
	tasks, tasksMap := toTasks(reqs)
	s.allTasks = tasks
	s.allTasksMap = tasksMap
	s.executeCh = make(chan func(), len(tasks))
	s.validateCh = make(chan func(), len(tasks))

	// default to number of tasks if workers is negative or 0 by this point
	workers := s.workers
	if s.workers < 1 || len(tasks) < s.workers {
		workers = len(tasks)
	}

	workerCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	// execution tasks are limited by workers
	start(workerCtx, s.executeCh, workers)

	// validation tasks uses length of tasks to avoid blocking on validation
	start(workerCtx, s.validateCh, len(tasks))

	toExecute := tasks
	for !allValidated(tasks) {
		// if the max incarnation >= x, we should revert to synchronous
		if iterations >= maximumIterations {
			// process synchronously
			s.synchronous = true
			startIdx, anyLeft := s.findFirstNonValidated()
			if !anyLeft {
				break
			}
			toExecute = tasks[startIdx:]
		}

		// execute sets statuses of tasks to either executed or aborted
		if err := s.executeAll(toExecute); err != nil {
			return nil, err
		}

		// validate returns any that should be re-executed
		// note this processes ALL tasks, not just those recently executed
		var err error
		toExecute, err = s.validateAll(tasks)
		if err != nil {
			return nil, err
		}
		iterations++
	}

	s.multiVersionStores.WriteLatestToStore(s.state)

	return s.collectResponses(tasks), nil
}

func (s *scheduler) shouldRerun(task *deliverTxTask) bool {
	switch task.Status {

	case statusAborted, statusPending:
		return true

	// validated tasks can become unvalidated if an earlier re-run task now conflicts
	case statusExecuted, statusValidated:
		// With the current scheduler, we won't actually get to this step if a previous task has already been determined to be invalid,
		// since we choose to fail fast and mark the subsequent tasks as invalid as well.
		// TODO: in a future async scheduler that no longer exhaustively validates in order, we may need to carefully handle the `valid=true` with conflicts case
		if valid, conflicts := s.findConflicts(task); !valid {
			s.invalidateTask(task)
			task.AppendDependencies(conflicts)

			// if the conflicts are now validated, then rerun this task
			if dependenciesValidated(s.allTasksMap, task.Dependencies) {
				return true
			} else {
				// otherwise, wait for completion
				task.SetStatus(statusWaiting)
				return false
			}
		} else if len(conflicts) == 0 {
			// mark as validated, which will avoid re-validating unless a lower-index re-validates
			task.SetStatus(statusValidated)
			return false
		}
		// conflicts and valid, so it'll validate next time
		return false

	case statusWaiting:
		// if conflicts are done, then this task is ready to run again
		return dependenciesValidated(s.allTasksMap, task.Dependencies)
	}
	panic("unexpected status: " + task.Status)
}

func (s *scheduler) validateTask(task *deliverTxTask) bool {
	if s.shouldRerun(task) {
		return false
	}
	return true
}

func (s *scheduler) findFirstNonValidated() (int, bool) {
	for i, t := range s.allTasks {
		if t.Status != statusValidated {
			return i, true
		}
	}
	return 0, false
}

func (s *scheduler) validateAll(tasks []*deliverTxTask) ([]*deliverTxTask, error) {
	var mx sync.Mutex
	var res []*deliverTxTask

	startIdx, anyLeft := s.findFirstNonValidated()

	if !anyLeft {
		return nil, nil
	}

	wg := &sync.WaitGroup{}
	for i := startIdx; i < len(tasks); i++ {
		wg.Add(1)
		t := tasks[i]
		s.DoValidate(func() {
			defer wg.Done()
			if !s.validateTask(t) {
				mx.Lock()
				defer mx.Unlock()
				t.Reset()
				t.Increment()
				// update max incarnation for scheduler
				if t.Incarnation > s.maxIncarnation {
					s.maxIncarnation = t.Incarnation
				}
				res = append(res, t)
			}
		})
	}
	wg.Wait()

	return res, nil
}

func (s *scheduler) executeAll(tasks []*deliverTxTask) error {
	if len(tasks) == 0 {
		return nil
	}
	// validationWg waits for all validations to complete
	// validations happen in separate goroutines in order to wait on previous index
	wg := &sync.WaitGroup{}
	wg.Add(len(tasks))

	for _, task := range tasks {
		t := task
		s.DoExecute(func() {
			s.prepareAndRunTask(wg, t)
		})
	}

	wg.Wait()

	return nil
}

func (s *scheduler) prepareAndRunTask(wg *sync.WaitGroup, task *deliverTxTask) {
	s.executeTask(task)
	wg.Done()
}

func (s *scheduler) prepareTask(task *deliverTxTask) {
	abortCh := make(chan multiversion.Abort, 1)

	msg, err := task.Tx.AsMessageWithAccountKeyPicker(types.MakeSigner(s.chainConfig, s.header.Number), s.state, s.header.Number.Uint64())
	if err != nil {
		panic(err)
	}
	vs := s.multiVersionStores.VersionedIndexedStore(msg, task.AbsoluteIndex, task.Incarnation, abortCh)
	task.VersionStores = vs

	task.AbortCh = abortCh
}

func (s *scheduler) executeTask(task *deliverTxTask) {
	// in the synchronous case, we only want to re-execute tasks that need re-executing
	if s.synchronous {
		// if already validated, then this does another validation
		if task.IsStatus(statusValidated) {
			s.shouldRerun(task)
			if task.IsStatus(statusValidated) {
				return
			}
		}

		// waiting transactions may not yet have been reset
		// this ensures a task has been reset and incremented
		if !task.IsStatus(statusPending) {
			task.Reset()
			task.Increment()
		}
	}

	s.prepareTask(task)

	vmConfig := &vm.Config{
		Debug:  true,
		Tracer: task.VersionStores,
	}

	copiedState := s.state.Copy()
	receipt, trace, err := s.deliverTx(s.chainConfig, copiedState, s.header, task.Tx, task.UsedGas, vmConfig)
	if err != nil {
		panic(err)
	}
	// close the abort channel
	close(task.AbortCh)
	abort, ok := <-task.AbortCh
	if ok {
		// if there is an abort item that means we need to wait on the dependent tx
		task.SetStatus(statusAborted)
		task.Abort = &abort
		task.AppendDependencies([]int{abort.DependentTxIdx})
		// write from version store to multiversion stores
		task.VersionStores.WriteEstimatesToMultiVersionStore()
	}

	resp := &Response{
		Receipt: receipt,
		Trace:   trace,
	}
	task.SetStatus(statusExecuted)
	task.Response = resp

	// write from version store to multiversion stores
	task.VersionStores.WriteToMultiVersionStore()
}
