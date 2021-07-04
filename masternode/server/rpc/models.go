package rpc

// Worker contains the info of a worker.
type Worker struct {
	// WorkerName of the worker
	WorkerName string
	// StageIndex the index of the stage in the pipeline
	StageIndex int
}

func (w Worker) Stage() int {
	return w.StageIndex
}

func (w Worker) Name() string {
	return w.WorkerName
}

// Output of a processed input by a Worker.
type Output struct {
	// Storage's name which is used to save the output.
	Storage string
	// OutputPath is the path to the output file saved in the Storage.
	OutputPath string
}

func (o Output) StorageName() string {
	return o.Storage
}

func (o Output) Path() string {
	return o.OutputPath
}

// JobDoneRequest is emitted by a worker when its done processing the input.
type JobDoneRequest struct {
	// Worker which has processed the input
	Worker Worker
	// Output of the processed input.
	Output Output
}

// WorkerStartedRequest is emitted by a worker when it starts
type WorkerStartedRequest struct {
	// Worker which has started.
	Worker Worker
}

// TTLRequest is sent by a healthy worker.
type TTLRequest struct {
	// Worker which is sending the TTL check.
	Worker Worker
}

// EmptyResponse is like an OK response without any data.
type EmptyResponse struct {}