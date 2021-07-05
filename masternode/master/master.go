package master

import (
	"log"
)

// Master holds the state and manages the workers of each stage of the pipeline.
type Master struct {
	// stages contains a list of pipeline's stages. Each stageRunner contains a list of workers for that stage of the
	// pipeline.
	stages []*stageRunner
}

// New returns a new instance of a Master node which is responsible for managing the workers.
// Each stage of the pipeline instantiates a number of worker nodes proportional to its weight using the provided
// WorkerBuilderFunc.
// specs is an ordered slice of StageSpec, each describing one stage of the pipeline.
func New(workerBuilder WorkerBuilderFunc, specs ...StageSpec) *Master {
	// creating the pipeline's stages
	// each stageRunner spawns a bunch of worker nodes proportional to its weight

	// a unique id generator is needed for building unique names for the workers
	idGen, err := newWorkerIdGen()
	if err != nil {
		panic(err)
	}

	// instantiate and run each stage
	var stages = make([]*stageRunner, len(specs)+1)
	// reserve the first stage for the input-source which is the library running on the user's machine that is providing
	// us the original input data
	stages[0] = nil
	for i, spec := range specs {
		stages[i+1] = newStage(spec, i+1, workerBuilder, idGen)
	}

	return &Master{
		stages: stages,
	}
}

// JobDone a worker has finished its job. It lets the next stage to work on the output as its input. Nothing is done if
// the worker belongs to the last stage of the pipeline.
func (m *Master) JobDone(worker WorkerInfo, execJob ExecutedJob) {
	// validate the worker
	if worker.Stage() >= len(m.stages) || worker.Stage() < 0 {
		// worker's stage not valid
		log.Println("[!] Master: JobDone: worker's stage not valid: index out of range:", worker.Stage())
		return
	}

	// unless the worker doesn't belong to the first stage (the input source), mark the worker as STATE_IDLE and also,
	// validate the executed job because it could be reassigned to another worker.
	if worker.Stage() > 0 {
		stage := m.stages[worker.Stage()]
		// mark as idle
		err := stage.MarkIdle(worker.Name())
		if err != nil {
			log.Printf("[!] Master: JobDone: failed to mark worker %s as idle: omitting its work output, err: %v",
				worker.Name(), err)
			return
		}
		// validate the executed job
		err = stage.ValidateExecutedJob(worker.Name(), execJob.ID())
		if err != nil {
			log.Printf("[!] Master: JobDone: invalid executed job: %v", err)
			return
		}
	}

	// the last stage is the pipeline's sink. for now nothing's needed if the worker belongs to a sink.
	// TODO: SINK'S OUTPUT'S HERE, DO STH WITH IT.
	if worker.Stage() == len(m.stages)-1 {
		log.Printf("[!] Master: JobDone: sink %s processed an input & saved output at %s:%s", worker.Name(),
			execJob.OutputStorage(), execJob.OutputPath())
		return
	}

	// tell the next stage to process the output
	nextStage := m.stages[worker.Stage()+1] // notice how we've reserved the first (0th) stage for the input-source
	err := nextStage.Process(Job{StorageName: execJob.OutputStorage(), Path: execJob.OutputPath()})
	if err != nil {
		log.Printf("[!] Master: JobDone: stage %s failed to process the output of the previous one: %v",
			nextStage.spec.Name(), err)
	}
}

func (m *Master) WorkerStarted(worker WorkerInfo) {
	panic("implement me")
}

func (m *Master) TTLCheck(worker WorkerInfo) {
	panic("implement me")
}
