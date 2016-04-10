package dockworker

import (
	"fmt"
	"sync"

	log "github.com/Sirupsen/logrus"
	"github.com/fsouza/go-dockerclient"
)

// EventListener wraps the underly Docker event listener
type EventListener interface {
	Start() error
	Stop()
	RegisterListener(listener chan *docker.APIEvents)
	UnregisterListener(listener chan *docker.APIEvents)
}

// NewEventListener returns a new EventListener
func NewEventListener(client *docker.Client) EventListener {
	return &eventListener{
		client:    client,
		lock:      &sync.RWMutex{},
		listeners: make(map[chan *docker.APIEvents]bool),
	}
}

type eventListener struct {
	lock      *sync.RWMutex
	listeners map[chan *docker.APIEvents]bool
	client    *docker.Client
	running   bool
	eventChan chan *docker.APIEvents
}

func (el *eventListener) Start() error {
	// TODO: proper locking
	err := el.setupEventChan()
	if err != nil {
		return err
	}
	go el.eventWorker()
	return nil
}

func (el *eventListener) Stop() {
	// TODO: implement
}

func (el *eventListener) RegisterListener(listener chan *docker.APIEvents) {
	el.lock.Lock()
	defer el.lock.Unlock()
	if _, inMap := el.listeners[listener]; !inMap {
		el.listeners[listener] = true
	}
}
func (el *eventListener) UnregisterListener(listener chan *docker.APIEvents) {
	el.lock.Lock()
	if _, inMap := el.listeners[listener]; !inMap {
		delete(el.listeners, listener)
	}
	defer el.lock.Unlock()
}

func (el *eventListener) setupEventChan() error {

	el.eventChan = make(chan *docker.APIEvents)
	log.Debug("Adding event listener channel")
	if err := el.client.AddEventListener(el.eventChan); err != nil {
		err := fmt.Errorf("Failed to add event chan to event listener %s", err)
		log.Error(err)
		return err
	}
	log.Debug("Event listener channel added")
	return nil
}

func (el *eventListener) eventWorker() {
	for {
		for event := range el.eventChan {
			el.lock.RLock()
			for listener := range el.listeners {
				go sendToListener(listener, event)
			}
			el.lock.RUnlock()
		}
		log.Warn("Event chan closed, registering new one")
		el.setupEventChan()
	}
}

func sendToListener(listener chan *docker.APIEvents, event *docker.APIEvents) {
	listener <- event
}
