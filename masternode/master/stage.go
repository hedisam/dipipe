package master

import (
	"context"
	"fmt"
	"log"
	"strings"
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
	// idles is a queue which keeps track of idle workers.
	idles BlockingIdlesQueue
	// idGen generates unique names for the workers.
	idGen UniqueIdGenerator
	// jobs holds a list of this stage's jobs and their corresponding workers responsible for them.
	jobs map[string]string
	// jMutex controls concurrent access to jobs.
	jMutex sync.RWMutex
}

// newStage returns an instance of stageRunner which is responsible for running and maintaining the worker nodes of a
// stage in the pipeline.
// spec is the specification of the stage (e.g. the plugin of the stage, max workers allowed, name).
// index of this stage in the pipeline (e.g. 1th, 5th)
// workerBuilder builds and returns a worker node which will be managed by this stage.
// idGen generates unique ids used for creating unique names for the workers and ids the jobs.
func newStage(spec StageSpec, index int, workerBuilder WorkerBuilderFunc, idGen UniqueIdGenerator) *stageRunner {
	return &stageRunner{
		spec:          spec,
		index:         index,
		workerBuilder: workerBuilder,
		workers:       make(map[string]*WorkerState),
		idles:         newThreadSafeIdlesQueue(context.TODO(), newIdleQueue()),
		idGen:         idGen,
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

	// how this worker could be processing when it's not marked as BUSY? there must be an inconsistency, for instance,
	// the worker could've faced a timeout while sending ttl checks therefore the master has marked it unhealthy
	if worker.state != Busy {
		s.wMutex.Unlock()
		return fmt.Errorf("stageRunner %s: MarkIdle: worker %s's current state is not STATE_BUSY - its work "+
			"output should be omitted", s.spec.Name(), workerName)
	}

	// mark as idle
	worker.state = Idle
	s.wMutex.Unlock()

	// put the worker in the idles queue to be reused for further jobs
	s.idles.Enqueue(worker.worker)

	return nil
}

// ValidateExecutedJob checks whether this job has been reassigned or not, if so, then this job's output/result must be
// ignored.
func (s *stageRunner) ValidateExecutedJob(workerName, jobId string) error {
	s.jMutex.RLock()
	// get the responsible worker for this job
	respWorker, ok := s.jobs[jobId]
	s.jMutex.RUnlock()

	// first we need to have a record of this job
	if !ok {
		return fmt.Errorf("stageRunner %s: ValidateExecutedJob: job %s not found",
			s.spec.Name(), jobId)
	}

	// the worker's name that is responsible for this job has to match with the provided workerName
	if respWorker != workerName {
		return fmt.Errorf("stageRunner %s: ValidateExecutedJob: worker %s is not responsible for job %s",
			s.spec.Name(), workerName, jobId)
	}

	return nil
}

// Process the given input. It assigns the job to an idle worker if there's any, otherwise spawns a new one.
func (s *stageRunner) Process(job Job) error {
	err := s.validate(job)
	if err != nil {
		return fmt.Errorf("stageRunner %s: Process: invalid job: %w", s.spec.Name(), err)
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

// validate determines whether a job can be done by this stage.
func (s *stageRunner) validate(job Job) error {
	// check if the storage which the job's input is saved on is accessible to the workers of this stage
	for _, stg := range s.spec.Storages() {
		if stg == job.StorageName {
			return nil
		}
	}

	return fmt.Errorf("validate: this stage's workers don't have access to the job's input file storage: "+
		"stage's mounted storages: %v, job: %+v", s.spec.Storages(), job)
}

// schedule a job to be executed whenever there's an available worker.
func (s *stageRunner) schedule(job Job) error {
	go func() {
		// wait and block for an idle worker. the returned worker is guaranteed to be non-nil unless the queue has
		// been disposed
		w, disposed := s.idles.Dequeue()
		if disposed {
			return
		}
		err := s.assignJob(w, job)
		if err != nil {
			// todo: should we schedule the job again??
			log.Printf("stageRunner: schedule: failed to assign job %+v to the worker %+v, err: %v", job, w, err)
		}
	}()

	return nil
}

// spawnWorker creates and runs a new worker. It doesn't know about the type and the way that the worker gets created
// and spawned; it could be an executable program or a Docker container.
func (s *stageRunner) spawnWorker(ctx context.Context) error {
	// a unique name of the worker
	name := fmt.Sprintf("stage:%d:%s:worker:%s", s.index, s.spec.Name(), s.idGen.Id())
	// instantiate and spawn a worker node
	worker := s.workerBuilder(name, s.index, s.spec.Plugin())
	err := worker.Spawn(ctx)
	if err != nil {
		return fmt.Errorf("spawnWorker: failed to spawn a new worker: %w", err)
	}

	atomic.AddInt32(&s.runningWorkersCnt, 1)

	// index the workers to keep track of them and making it possible to access them by their name
	s.wMutex.Lock()
	s.workers[name] = &WorkerState{worker: worker, state: NotStarted}
	s.wMutex.Unlock()

	return nil
}

// assignJob assigns the provided job to be executed by the specified worker. It keeps track of the assigned worker to
// the job.
func (s *stageRunner) assignJob(worker Worker, job Job) error {
	// a unique job id is needed to know which worker is responsible for executing it
	// only assign a new unique id if this a new and fresh job
	var ok bool
	if strings.TrimSpace(job.Id) != "" {
		s.jMutex.RLock()
		_, ok = s.jobs[job.Id]
		s.jMutex.RUnlock()
	}

	if !ok {
		// it's a new/fresh job
		job.Id = "job:" + s.idGen.Id()
	}

	err := worker.Assign(context.TODO(), job)
	if err != nil {
		return fmt.Errorf("assignJob: failed to assign job %+v to worker %s: %w", job, worker.Name(), err)
	}

	// keep track of the worker that is responsible for this job
	s.jMutex.Lock()
	s.jobs[job.Id] = worker.Name()
	s.jMutex.Unlock()

	return nil
}
