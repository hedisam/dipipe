package rpc

import "github.com/hedisam/dipipe/masternode/server"

// Service is a native-go rpc service. It handles the rpc messages sent by the workers.
type Service struct {
	master server.Master
}

// NewService returns an instance of rpc Service which is responsible for handling messages from the workers.
func NewService(m server.Master) *Service {
	return &Service{
		master: m,
	}
}

// JobDone is received from a worker that has finished processing the input.
func (s *Service) JobDone(req JobDoneRequest, _ *EmptyResponse) error {
	s.master.JobDone(req.Worker, req.Output)
	return nil
}

// WorkerStarted is received from a worker that has just got started.
func (s *Service) WorkerStarted(req WorkerStartedRequest, _ *EmptyResponse) error {
	s.master.WorkerStarted(req.Worker)
	return nil
}

// TTL received from a worker that wants to claim a healthy state.
func (s *Service) TTL(req TTLRequest, _ *EmptyResponse) error {
	s.master.TTLCheck(req.Worker)
	return nil
}


