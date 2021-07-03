package server

// Master represents a master object which manages the pipeline's stages and workers.
type Master interface {
	// JobDone tells the master that the worker has processed its input.
	JobDone(worker WorkerInfo, output JobOutput)
	// WorkerStarted notifies the master that a worker has started.
	WorkerStarted(worker WorkerInfo)
	// TTLCheck tells the master about receiving a ttl check from the worker.
	TTLCheck(worker WorkerInfo)
}

type WorkerInfo interface {
	// Stage is the index of the stage in the pipeline (e.g. 1st, 2nd, 10th).
	Stage() int
	// Name returns the name of the worker.
	Name() string
}

type JobOutput interface {
	// StorageName returns the name of the storage; it would be the name of the volume mounted to the container in
	// case of utilizing Docker.
	StorageName() string
	// Path to the output file.
	Path() string
}

