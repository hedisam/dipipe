package master

import (
	"context"
)

// master holds the state and manages the workers of each stage of the pipeline.
type master struct {
	// stages contains a list of pipeline's stages. Each stage could contain a list of workers for that stage of the
	// pipeline.
	stages []*stage
}

// New returns a new instance of a master node which is responsible for managing the workers.
// Each stage of the pipeline instantiates a number of worker nodes proportional to its weight using the provided
// WorkerBuilderFunc.
// totalNodesNum is the total number of nodes the user wants to be spawned and used for this pipeline.
// A slice of StageSpec needs to be provided to create the pipeline's stage.
// totalWeights is the total weight of the stages plus the weight of the source and sink of the pipeline.
func New(workerBuilder WorkerBuilderFunc, totalNodesNum, totalWeights int, specs ...StageSpec) *master {
	// creating the pipeline's stages
	// each stage spawns a bunch of worker nodes proportional to its weight
	var stages = make([]*stage, len(specs))
	for i, spec := range specs {
		workersNum := spec.Weight() / totalWeights * totalNodesNum
		stages[i] = newStage(spec.Name(), spec.Plugin(), workerBuilder, workersNum, newIdleQueue())
	}

	return &master{
		stages: stages,
	}
}

// Run the master node which then it will run the worker nodes.
func (m *master) Run(ctx context.Context) error {
	panic("implement master.Run")
}
