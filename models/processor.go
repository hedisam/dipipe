package models

import "context"

// Processor is implemented by the users and defines an abstraction for the job that needs to be done by a worker.
type Processor interface {
	// Process runs the user-defined logic
	// TODO: complete the abstraction
	Process(ctx context.Context) error
}
