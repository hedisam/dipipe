package master

import (
	"github.com/hedisam/dipipe/masternode/server"
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

	// instantiate and run each stage
	var stages = make([]*stageRunner, len(specs)+1)
	// reserve the first stage for the input-source which is the library running on the user's machine that is providing
	// us the original input data
	stages[0] = nil
	for i, spec := range specs {
		stages[i+1] = newStage(spec, i+1, workerBuilder)
	}

	return &Master{
		stages: stages,
	}
}

// JobDone a worker has finished its job. It lets the next stage to work on the output as its input. Nothing is done if
// the worker belongs to the last stage of the pipeline.
func (m *Master) JobDone(worker server.WorkerInfo, output server.JobOutput) {
	// validate the worker
	if worker.Stage() >= len(m.stages) {
		// worker's stage not valid
		log.Println("[!] Master: JobDone: worker's stage not valid: index out of range:", worker.Stage())
		return
	}

	// mark the worker as IDLE
	stage := m.stages[worker.Stage()]
	err := stage.MarkIdle(worker.Name())
	if err != nil {
		log.Printf("[!] Master: JobDone: failed to mark worker %s as idle: omitting its work output, err: %v",
			worker.Name(), err)
		return
	}

	// the last stage is the pipeline's sink. for now nothing's needed if the worker belongs to a sink.
	// TODO: SINK'S OUTPUT'S HERE, DO STH WITH IT.
	if worker.Stage() == len(m.stages) - 1 {
		log.Printf("[!] Master: JobDone: sink %s processed an input & saved output at %s:%s", worker.Name(),
			output.StorageName(), output.Path())
		return
	}

	// tell the next stage to process the output
	nextStage := m.stages[worker.Stage()+1] // notice how we've reserved the first (0th) stage for the input-source
	err = nextStage.Process(Job{StorageName: output.StorageName(), Path: output.Path()})
	if err != nil {
		log.Printf("[!] Master: JobDone: stage %s failed to process the output of the previous one: %v",
			nextStage.spec.Name(), err)
	}
}

func (m *Master) WorkerStarted(worker server.WorkerInfo) {
	panic("implement me")
}

func (m *Master) TTLCheck(worker server.WorkerInfo) {
	panic("implement me")
}
