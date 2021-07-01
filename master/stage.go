package master

import (
	"context"
	"fmt"
	"github.com/hedisam/dipipe/models"
)

// stage holds a list of workers and their state for a specific stage of the pipeline.
type stage struct {
	// name is the name of this stage
	name string
	// proc is the user-defined processor for this stage
	proc models.Processor
	// workerBuilder instantiate and returns a worker with the given stage processor
	workerBuilder WorkerBuilderFunc
	// workers holds the state of running worker nodes.
	workers []Worker
}

// newStage returns an instance of stage which is responsible for holding the worker nodes' stage of the corresponding
// stage.
func newStage(name string, proc models.Processor, workerBuilder WorkerBuilderFunc, workersNum int) *stage {
	// instantiate the required worker nodes for this stage
	workers := make([]Worker, 0, workersNum)
	for i := 0; i < workersNum; i++ {
		workers = append(workers, workerBuilder(proc))
	}

	return &stage{
		name: name,
		proc:          proc,
		workerBuilder: workerBuilder,
		workers:       workers,
	}
}

// Run the stage by spawning the workers. It returns an error if it fails to spawn any of the worker nodes.
// TODO: (this behaviour should be configurable by the user since if there's a shortage of available nodes the user may
// be ok with carrying on with what she's got).
func (s *stage) Run(ctx context.Context) error {
	var err error

	for _, w := range s.workers {
		err = w.Spawn(ctx)
		if err != nil {
			// stop the nodes that have been spawned so far
			// TODO: log the error
			_ = s.Stop(ctx)
			return fmt.Errorf("stage %s: Run: failed to spawn a worker node: %w", s.name, err)
		}
	}

	return nil
}

func (s *stage) Stop(ctx context.Context) error {
	panic("implement stage.Stop")
}
