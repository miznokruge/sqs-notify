package main

import (
	"container/list"
	"sync"
)

type jobState int

const (
	jobStarted jobState = iota + 1
	jobRunning
	jobCompleted
)

func (s jobState) String() string {
	switch s {
	case jobStarted:
		return "started"
	case jobRunning:
		return "running"
	case jobCompleted:
		return "completed"
	default:
		return ""
	}
}

func parseJobState(s string) (jobState, bool) {
	switch s {
	case "started":
		return jobStarted, true
	case "running":
		return jobRunning, true
	case "completed":
		return jobCompleted, true
	default:
		return 0, false
	}
}

type jobs interface {
	StartTry(id string) jobState
	Fail(id string)
	Complete(id string)
	Close()
}

func newJobs(capacity int) (jobs, error) {
	return &jobManager{
		capacity: capacity,
		table:    make(map[string]*jobItem),
		keys:     list.New(),
	}, nil
}

type jobItem struct {
	el    *list.Element
	state jobState
}

type jobManager struct {
	capacity int
	lock     sync.Mutex
	table    map[string]*jobItem
	keys     *list.List
}

func (m *jobManager) StartTry(id string) jobState {
	if m.capacity <= 0 {
		return jobStarted
	}
	// get lock for table.
	m.lock.Lock()
	defer m.lock.Unlock()
	// search from cached key.
	s, ok := m.table[id]
	if ok {
		return s.state
	}
	// remove old entries, if over capacity.
	for m.keys.Len() >= m.capacity {
		f := m.keys.Front()
		delete(m.table, f.Value.(string))
		m.keys.Remove(f)
	}
	// add a key.
	el := m.keys.PushBack(id)
	m.table[id] = &jobItem{
		el:    el,
		state: jobRunning,
	}
	return jobStarted
}

func (m *jobManager) Fail(id string) {
	if m.capacity <= 0 {
		return
	}
	// get lock for table.
	m.lock.Lock()
	defer m.lock.Unlock()
	// search cached key.
	s, ok := m.table[id]
	if !ok {
		return
	}
	// remove a key.
	delete(m.table, id)
	m.keys.Remove(s.el)
}

func (m *jobManager) Complete(id string) {
	if m.capacity <= 0 {
		return
	}
	// get lock for table.
	m.lock.Lock()
	defer m.lock.Unlock()
	// search cached key.
	s, ok := m.table[id]
	if !ok {
		return
	}
	// remove a key.
	s.state = jobCompleted
}

func (m *jobManager) Close() {
	// nothing to do.
}
