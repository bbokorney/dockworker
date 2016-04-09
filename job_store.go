package main

import "sync"

// JobStore stores jobs
type JobStore interface {
	Add(job Job) (Job, error)
	Find(ID JobID) (Job, error)
	Update(job Job) error
}

// NewJobStore creates a new JobStore
func NewJobStore() JobStore {
	return &inMemJobStore{
		lock:   &sync.RWMutex{},
		nextID: 0,
		data:   make(map[JobID]Job),
	}
}

type inMemJobStore struct {
	lock   *sync.RWMutex
	nextID JobID
	data   map[JobID]Job
}

func (store *inMemJobStore) Add(j Job) (Job, error) {
	store.lock.Lock()
	defer store.lock.Unlock()
	j.ID = store.nextID
	store.data[j.ID] = j
	store.nextID = store.nextID + 1
	return j, nil
}

func (store inMemJobStore) Find(ID JobID) (Job, error) {
	store.lock.RLock()
	defer store.lock.RUnlock()
	s, ok := store.data[ID]
	if !ok {
		return Job{}, ErrJobNotFound
	}
	return s, nil
}

func (store inMemJobStore) Update(job Job) error {
	store.lock.Lock()
	defer store.lock.Unlock()

	if _, ok := store.data[job.ID]; !ok {
		return ErrJobNotFound
	}
	store.data[job.ID] = job
	return nil
}
