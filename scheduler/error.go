package scheduler

import "errors"

var (
	ErrNeedSequentialExecution = errors.New("need sequential execution")
	ErrSchedulerNotReady       = errors.New("scheduler not ready")
	ErrSchedulerRunning        = errors.New("scheduler running")
	ErrSchedulerFinished       = errors.New("scheduler finished")
	ErrSchedulerFailed         = errors.New("scheduler failed")
	ErrPrepareHeader           = errors.New("prepare header failed")
	ErrSchedulerTimeout        = errors.New("scheduler timeout")
)
