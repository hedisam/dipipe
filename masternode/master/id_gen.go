package master

import (
	"fmt"
	"github.com/teris-io/shortid"
	"time"
)

// workerIdGen implements WorkerIdGenerator.
type workerIdGen struct {
	shId *shortid.Shortid
}

func newWorkerIdGen() (*workerIdGen, error) {
	shId, err := shortid.New(1, shortid.DefaultABC, uint64(time.Now().Unix()))
	if err != nil {
		return nil, fmt.Errorf("newWrokerIdGen: failed to instantiate a shortid generator: %w", err)
	}

	return &workerIdGen{shId: shId}, nil
}

// Id implements WorkerIdGenerator.
func (w *workerIdGen) Id() (string, error) {
	id, err := w.shId.Generate()
	if err != nil {
		return "", fmt.Errorf("workerIdGen: Id: failed to generate a unique id: %w", err)
	}

	return id, nil
}

