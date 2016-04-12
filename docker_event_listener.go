package dockworker

import (
	"fmt"
	"sync"

	log "github.com/Sirupsen/logrus"
	"github.com/fsouza/go-dockerclient"
)

// DockerEventListener wraps the underlying Docker event listener
type DockerEventListener interface {
	Start() error
	Stop()
	RegisterListener(listener chan *docker.APIEvents)
	UnregisterListener(listener chan *docker.APIEvents)
}

// TODO: There is probably a way to combine these two event listeners
// or maybe they're fine separate

// NewEventListener returns a new EventListener
func NewEventListener(client *docker.Client) DockerEventListener {
	return &dockerEventListener{
		client:    client,
		lock:      &sync.RWMutex{},
		listeners: make(map[chan *docker.APIEvents]bool),
	}
}

type dockerEventListener struct {
	lock      *sync.RWMutex
	listeners map[chan *docker.APIEvents]bool
	client    *docker.Client
	running   bool
	eventChan chan *docker.APIEvents
}

func (el *dockerEventListener) Start() error {
	// TODO: proper locking
	err := el.setupEventChan()
	if err != nil {
		return err
	}
	go el.eventWorker()
	return nil
}

func (el *dockerEventListener) Stop() {
	// TODO: implement
}

func (el *dockerEventListener) RegisterListener(listener chan *docker.APIEvents) {
	el.lock.Lock()
	defer el.lock.Unlock()
	if _, inMap := el.listeners[listener]; !inMap {
		el.listeners[listener] = true
	}
}

func (el *dockerEventListener) UnregisterListener(listener chan *docker.APIEvents) {
	el.lock.Lock()
	defer el.lock.Unlock()
	if _, inMap := el.listeners[listener]; inMap {
		// TODO: flush and close channel
		delete(el.listeners, listener)
	}
}

func (el *dockerEventListener) setupEventChan() error {
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

func (el *dockerEventListener) eventWorker() {
	for {
		for event := range el.eventChan {
			el.lock.RLock()
			for listener := range el.listeners {
				go el.sendToListener(listener, event)
			}
			el.lock.RUnlock()
		}
		log.Warn("Event chan closed, registering new one")
		el.setupEventChan()
	}
}

func (el *dockerEventListener) sendToListener(listener chan *docker.APIEvents, event *docker.APIEvents) {
	listener <- event
}
