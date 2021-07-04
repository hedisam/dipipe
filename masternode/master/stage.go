package master

import (
	"context"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
)

// stageRunner holds a list of workers and their state for a specific stage of the pipeline.
type stageRunner struct {
	// spec is the specification of this stage.
	spec StageSpec
	// index of this stage in the pipeline (e.g. 1st, 3th, 10th).
	index int
	// workerBuilder instantiate and returns a worker.
	workerBuilder WorkerBuilderFunc
	// workers holds the running workers along with their state.
	workers map[string]*WorkerState
	// runningWorkersCnt keeps track of the total count of running workers which are busy and alive.
	runningWorkersCnt int32
	// wMutex controls concurrent access to the workers.
	wMutex sync.RWMutex
	// idles is a queue which keeps track of idle workers
	idles BlockingIdlesQueue
}

// newStage returns an instance of stageRunner which is responsible for running and maintaining the worker nodes of a
// stage in the pipeline.
// spec is the specification of the stage (e.g. the plugin of the stage, max workers allowed, name).
// workerBuilder builds and returns a worker node which will be managed by this stage.
func newStage(spec StageSpec, index int, workerBuilder WorkerBuilderFunc) *stageRunner {
	return &stageRunner{
		spec: spec,
		index: index,
		workerBuilder: workerBuilder,
		workers:       make(map[string]*WorkerState),
		idles:         newThreadSafeIdlesQueue(newIdleQueue()),
	}
}

// MarkIdle marks a worker as idle to be ready for further jobs.
func (s *stageRunner) MarkIdle(workerName string) error {
	s.wMutex.Lock()

	worker, ok := s.workers[workerName]
	if !ok {
		s.wMutex.Unlock()
		return fmt.Errorf("stageRunner %s: MarkIdle: worker %s not found", s.spec.Name(), workerName)
	}

	// for instance, how this worker could be processing when it's not marked as BUSY? there must be an inconsistency.
	if worker.state != Busy {
		s.wMutex.Unlock()
		return fmt.Errorf("stageRunner %s: MarkIdle: worker %s's current state is not STATE_BUSY - its work " +
			"output should be omitted", s.spec.Name(), workerName)
	}

	// mark as idle
	worker.state = Idle
	s.wMutex.Unlock()

	// put the worker in the idles queue to be reused for further jobs
	s.idles.Enqueue(worker.worker)

	return nil
}

// Process the given input. It assigns the job to an idle worker if there's any, otherwise spawns a new one.
func (s *stageRunner) Process(job Job) error {
	err := s.validate(job)
	if err != nil {
		return fmt.Errorf("stageRunner %s: Process: %w", s.spec.Name(), err)
	}

	// see if there's any idle worker
	worker := s.idles.TryDequeue()
	if worker != nil {
		err = s.assignJob(worker, job)
		if err != nil {
			// todo: should we try to assign the job to another worker?
			return fmt.Errorf("stageRunner %s: Process: failed to assign the job to worker %s: %w", s.spec.Name(), worker.Name(), err)
		}
		return nil
	}

	// no idle workers, we have to schedule the job
	err = s.schedule(job)
	if err != nil {
		return fmt.Errorf("stageRunner %s: Process: failed to schedule job %+v: %w", s.spec.Name(), job, err)
	}

	// see if we can spawn a fresh worker to come alive and process the scheduled job
	cnt := atomic.LoadInt32(&s.runningWorkersCnt)
	if int(cnt) >= s.spec.MaxWorkersNum() {
		// we have reached the max running workers that this stage can have, we have to wait for a BUSY worker to
		// become IDLE
		return nil
	}

	// we've still got room for another running worker, spawn it, so it can process the job as soon as it gets started
	err = s.spawnWorker(context.TODO())
	if err != nil {
		return fmt.Errorf("stageRunner %s: Process: %w", s.spec.Name(), err)
	}

	return nil
}

func (s *stageRunner) validate(job Job) error {
	panic("implement stageRunner.validate")
}

func (s *stageRunner) schedule(job Job) error {
	go func() {
		// todo: make this blocking process cancellable (e.g. cancel if a context got cancelled)
		// wait and block for an idle worker. the returned worker is guaranteed to be non-nil
		w := s.idles.Dequeue()
		err := s.assignJob(w, job)
		if err != nil {
			// todo: should we schedule the job again??
			log.Printf("stageRunner: schedule: failed to assign job %+v to the worker %+v, err: %v", job, w, err)
		}
	}()


	return nil
}

func (s *stageRunner) spawnWorker(ctx context.Context) error {
	// a unique name for our new worker
	// todo: create a unique name
	var name string

	// instantiate and spawn a worker node
	worker := s.workerBuilder(name, s.index, s.spec.Plugin())
	err := worker.Spawn(ctx)
	if err != nil {
		return fmt.Errorf("spawnWorker: failed to spawn a new worker: %w", err)
	}

	atomic.AddInt32(&s.runningWorkersCnt, 1)

	s.wMutex.Lock()
	s.workers[name] = &WorkerState{worker: worker, state: NotStarted}
	s.wMutex.Unlock()

	return nil
}

func (s *stageRunner) assignJob(worker Worker, job Job) error {
	panic("implement stageRunner.assignJob")
}