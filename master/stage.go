package master

import (
	"github.com/hedisam/dipipe/models"
)

// stage holds a list of workers and their state for a specific stage of the pipeline.
type stage struct {
	// proc is the user-defined processor for this stage
	proc models.Processor
	// workerBuilder instantiate and returns a worker with the given stage processor
	workerBuilder WorkerBuilderFunc
	// workers holds the state of running worker nodes.
	workers []Worker
}

// newStage returns an instance of stage which is responsible for holding the worker nodes' stage of the corresponding
// stage.
func newStage(proc models.Processor, workerBuilder WorkerBuilderFunc, workersNum int) *stage {
	// instantiate the required worker nodes for this stage
	workers := make([]Worker, 0, workersNum)
	for i := 0; i < workersNum; i++ {
		workers = append(workers, workerBuilder(proc))
	}

	return &stage{
		proc:          proc,
		workerBuilder: workerBuilder,
		workers:       workers,
	}
}
