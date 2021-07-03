package master

import (
	"context"
	"fmt"
	"sync"
)

// stageRunner holds a list of workers and their state for a specific stage of the pipeline.
type stageRunner struct {
	// name of this stage
	name string
	// plugin is the user-defined code to be ran on this stage.
	plugin StagePlugin
	// workerBuilder instantiate and returns a worker.
	workerBuilder WorkerBuilderFunc
	// workersNum is the number of worker nodes required for this stage.
	workersNum int
	// workers holds the running workers along with their state.
	workers map[string]*WorkerState
	// wMutex controls concurrent access to the workers slice.
	wMutex sync.RWMutex
	// idles is a queue which keeps track of idle workers
	idles IdlesQueue
}

// newStage returns an instance of stageRunner which is responsible for running and maintaining the worker nodes of a
// stage of the pipeline.
// name represents the name of this stage.
// plugin is the user-defined code to be ran by the worker nodes of this stage.
// workerBuilder builds and returns a worker node which will be managed by this stage.
// workersNum specified the number of worker nodes that should be spawned for this stage.
func newStage(name string, plugin StagePlugin, workerBuilder WorkerBuilderFunc, workersNum int, idles IdlesQueue) *stageRunner {
	return &stageRunner{
		name:          name,
		plugin:        plugin,
		workerBuilder: workerBuilder,
		workersNum:    workersNum,
		workers:       make(map[string]*WorkerState),
		idles:         idles,
	}
}

// Setup the stage by spawning the workers. It returns an error if it fails to spawn any of the worker nodes.
// TODO: (this behaviour should be configurable by the user since if there's a shortage of available nodes the user may
// be ok with carrying on with what she's got).
func (s *stageRunner) Setup(ctx context.Context) error {
	wg := &sync.WaitGroup{}
	errorCh := make(chan error)

	spawnCtx, cancelCtx := context.WithCancel(ctx)
	defer cancelCtx()

	// spawn all the worker nodes concurrently
	for i := 1; i <= s.workersNum; i++ {
		wg.Add(1)
		workerName := fmt.Sprintf("%s:worker:%d", s.name, i)
		go s.spawnWorker(spawnCtx, workerName, wg, errorCh)
	}

	go func() {
		wg.Wait()
		// all the nodes have been spawned successfully, we can safely close the error channel as there's no one left
		// to write to it. also closing it signals that all the nodes are spawned and running.
		close(errorCh)
	}()

	select {
	case <-ctx.Done():
		return nil
	case err, ok := <-errorCh:
		if ok {
			// the channel is not closed so we have received an error message.
			err = fmt.Errorf("stageRunner %s: Run: failed to spawn one of the worker nodes: %w", s.name, err)
			cancelCtx()
			wg.Wait()
			return err
		}
		// if we're here it means that all of the worker nodes have bee spawned successfully
	}

	return nil
}

func (s *stageRunner) spawnWorker(ctx context.Context, name string, wg *sync.WaitGroup, errorCh chan error) {
	defer wg.Done()

	// instantiate a worker node
	worker := s.workerBuilder(name, s.plugin)
	// spawn an executable or a docker container for the worker
	err := worker.Spawn(ctx)
	if err != nil {
		select {
		case <-ctx.Done():
			return
		case errorCh <- fmt.Errorf("stageRunner: Run: failed to spawn the worker %s: %w", name, err):
		}
	}

	s.wMutex.Lock()
	s.workers[name] = &WorkerState{worker: worker, state: workerIDLE}
	s.wMutex.Unlock()

	// keeping track of idle workers
	s.idles.Enqueue(ctx, worker)
}

func (s *stageRunner) Dispose() {
	s.idles.Dispose()
}
