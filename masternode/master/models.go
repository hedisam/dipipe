package master

import (
	"context"
)

type workerState int

const (
	// Idle describes a worker which is waiting for a job to work on.
	Idle workerState = iota
	// Busy describes a worker which is busy working on a job.
	Busy
	// Fatal describes a worker which has terminated abnormally and needs to be restarted.
	Fatal
	// NotStarted describes a spawned worker which has not confirmed that it has started.
	NotStarted
)

// StagePlugin is the path to the built plugin by the library which contains the user-defined code (aka the Processor)
// for a stage of the pipeline.
type StagePlugin string

// StageSpec is the specification for a stage of the pipeline.
type StageSpec interface {
	// Name of this stage
	Name() string
	// MaxWorkersNum that could be spawned for this stage.
	MaxWorkersNum() int
	// Plugin returns this stage's Plugin that contains the user-defined code for the Processor.
	Plugin() StagePlugin
	// Storages returns a list of storages which are accessible (mounted to) from the workers of this stage.
	Storages() []string
}

// WorkerBuilderFunc defines an abstraction to decouple a worker node instantiation.
// It expects a name for the worker, a stageIndex that shows which stage of the pipeline this worker belongs to, and
// a StagePlugin to be ran on the node.
type WorkerBuilderFunc func(name string, stageIndex int, plugin StagePlugin) Worker

// Worker represents a worker node.
type Worker interface {
	// Name of the worker.
	Name() string
	// Spawn the worker node.
	Spawn(ctx context.Context) error
	// Check the worker node if it's healthy.
	Check(ctx context.Context) (bool, error)
}

type WorkerInfo interface {
	// Name of the worker.
	Name() string
	// Stage returns the index of the worker's stage in the pipeline.
	Stage() int
}

// WorkerState holds a worker along with its state.
type WorkerState struct {
	worker Worker
	state  workerState
}

// Job needs to be done by a worker. It points to the input data which needs processing.
type Job struct {
	// Id is a unique id to know which worker is responsible for this job.
	Id string
	// StorageName name of the storage where the input data is saved
	StorageName string
	// Path to the input data
	Path string
}

// ExecutedJob represents an executed job along with the output.
type ExecutedJob interface {
	// ID returns the unique id of the job.
	ID() string
	// OutputStorage returns the name of the storage which the job's output has been saved in. It would be the name of
	// the volume mounted to the container in case of utilizing Docker.
	OutputStorage() string
	// OutputPath returns the path to the output file.
	OutputPath() string
}

// BlockingIdlesQueue is a thread-safe queue used to hold idle workers.
type BlockingIdlesQueue interface {
	// Enqueue a worker. It needs to be thread-safe.
	Enqueue(w Worker)
	// Dequeue a worker. It blocks until it gets an idle worker, unless the queue has been disposed.
	Dequeue() (w Worker, disposed bool)
	// TryDequeue tries to dequeue a worker, returns nil if there's no idle workers in the queue.
	TryDequeue() Worker
}

// IdlesQueue used to hold idle workers.
type IdlesQueue interface {
	// Enqueue a worker
	Enqueue(w Worker)
	// Dequeue a worker if there's any, otherwise return nil
	Dequeue() Worker
}

// UniqueIdGenerator abstracts the id generating process used for workers and jobs.
type UniqueIdGenerator interface {
	// Id returns a unique id.
	Id() string
}
