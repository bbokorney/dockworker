package dockworker

import "sync"

// StopEventListener handles requests to stop running jobs
type StopEventListener interface {
	Start() error
	Stop()
	RegisterListener(listener chan JobID)
	UnregisterListener(listener chan JobID)
}

// NewStopEventListener returns a new StopEventListener
func NewStopEventListener(stopEventChan chan JobID) StopEventListener {
	return &stopEventListener{
		lock:      &sync.RWMutex{},
		listeners: make(map[chan JobID]bool),
		eventChan: stopEventChan,
	}
}

type stopEventListener struct {
	lock      *sync.RWMutex
	listeners map[chan JobID]bool
	running   bool
	eventChan chan JobID
}

func (el *stopEventListener) Start() error {
	// TODO: proper locking
	go el.eventWorker()
	return nil
}

func (el *stopEventListener) Stop() {
	// TODO: implement
}

func (el *stopEventListener) RegisterListener(listener chan JobID) {
	el.lock.Lock()
	defer el.lock.Unlock()
	el.listeners[listener] = true
}

func (el *stopEventListener) UnregisterListener(listener chan JobID) {
	el.lock.Lock()
	defer el.lock.Unlock()
	delete(el.listeners, listener)
}

func (el *stopEventListener) eventWorker() {
	for {
		for event := range el.eventChan {
			el.lock.RLock()
			for listener := range el.listeners {
				go el.sendToListener(listener, event)
			}
			el.lock.RUnlock()
		}
	}
}

func (el *stopEventListener) sendToListener(listener chan JobID, event JobID) {
	listener <- event
}
