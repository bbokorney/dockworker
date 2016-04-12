package dockworker

import log "github.com/Sirupsen/logrus"

// StopService handles stopping jobs
type StopService interface {
	Stop(JobID) error
}

// NewStopService returns a new StopService
func NewStopService(stopEventChan chan JobID) StopService {
	return stopService{
		stopEventChan: stopEventChan,
	}
}

type stopService struct {
	stopEventChan chan JobID
}

func (s stopService) Stop(ID JobID) error {
	log.Debugf("StopService.Stop %d", ID)
	go s.notifyStop(ID)
	return nil
}

func (s stopService) notifyStop(jobID JobID) {
	log.Debugf("StopService.notifyStop %d before send", jobID)
	s.stopEventChan <- jobID
	log.Debugf("StopService.notifyStop %d after send", jobID)
}
