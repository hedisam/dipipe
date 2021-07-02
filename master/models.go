package master

import (
	"context"
)

// StagePlugin is the path to the built plugin by the library which contains the user-defined code (aka the Processor)
// for a stage of the pipeline.
type StagePlugin string

// StageSpec is the specification for a stage of the pipeline.
type StageSpec interface {
	// Name of this stage
	Name() string
	// Weight is used to determine how many worker nodes out of the total available nodes should be assigned to
	// this stage.
	Weight() int
	// Plugin returns this stage's Plugin that contains the user-defined code for the Processor.
	Plugin() StagePlugin
}

// WorkerBuilderFunc defines an abstraction to decouple a worker node instantiation. It expects a name for the worker
// and a StagePlugin to be ran on the node.
type WorkerBuilderFunc func(name string, plugin StagePlugin) Worker

// Worker represents a worker node.
type Worker interface {
	// Name of the worker.
	Name() string
	// Spawn the worker node.
	Spawn(ctx context.Context) error
	// Check the worker node if it's healthy.
	Check(ctx context.Context) (bool, error)
}

// WorkerState holds a worker along with its state.
type WorkerState struct {
	worker Worker
	state int
}

const (
	// workerIDLE describes a worker which is waiting for a job to work on.
	workerIDLE = iota
	// workerBUSY describes a worker which is busy working on a job.
	workerBUSY
	// workerDONE describes a worker which has just finished working on its job.
	workerDONE
	// workerFATAL describes a worker which has terminated abnormally and needs to be restarted.
	workerFATAL
)

type IdlesQueue interface {
	// Enqueue a worker
	Enqueue(ctx context.Context, worker Worker)
	// Dequeue a worker. Blocks forever if no idle workers are available, unless the context gets cancelled.
	Dequeue(ctx context.Context) Worker
	Dispose()
}
