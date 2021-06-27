package master

import (
	"context"
	"github.com/hedisam/dipipe/models"
)

// StageSpec is the specification for a stage of the pipeline.
type StageSpec interface {
	// Weight is used to determine how many worker nodes out of the total available nodes should be assigned to
	// this stage.
	Weight() int
	// Processor returns the user-defined processor function of this stage.
	Processor() models.Processor
}

// WorkerBuilderFunc defines an abstraction to decouple a worker node instantiation.
type WorkerBuilderFunc func(processor models.Processor) Worker

// Worker represents a worker node.
type Worker interface {
	// Spawn the worker node.
	Spawn(ctx context.Context) error
	// Check the worker node if it's healthy.
	Check(ctx context.Context) (bool, error)
}
