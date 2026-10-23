// Package store provides a thread-safe, in-memory storage layer for tasks.
//
// Data lives only in memory, so it is lost when the server stops. To add
// persistence, write another implementation with the same methods (for
// example one backed by SQLite).
package store

import (
	"errors"
	"sort"
	"sync"
	"time"
)

// ErrNotFound is returned when a task with the requested ID does not exist.
var ErrNotFound = errors.New("task not found")

// Task is a single to-do item.
type Task struct {
	ID        int       `json:"id"`
	Title     string    `json:"title"`
	Done      bool      `json:"done"`
	CreatedAt time.Time `json:"created_at"`
}

// Store holds tasks in a map guarded by a RWMutex, so it is safe to use
// from many HTTP handlers (goroutines) at once.
type Store struct {
	mu     sync.RWMutex
	tasks  map[int]Task
	nextID int
}

// New returns an empty Store. IDs start at 1.
func New() *Store {
	return &Store{tasks: make(map[int]Task), nextID: 1}
}

// List returns all tasks sorted by ID.
func (s *Store) List() []Task {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]Task, 0, len(s.tasks))
	for _, t := range s.tasks {
		out = append(out, t)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// Get returns the task with the given ID, or ErrNotFound.
func (s *Store) Get(id int) (Task, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	t, ok := s.tasks[id]
	if !ok {
		return Task{}, ErrNotFound
	}
	return t, nil
}

// Create adds a new task with the given title and returns it.
func (s *Store) Create(title string) Task {
	s.mu.Lock()
	defer s.mu.Unlock()

	t := Task{ID: s.nextID, Title: title, CreatedAt: time.Now().UTC()}
	s.tasks[t.ID] = t
	s.nextID++
	return t
}

// Update changes the title and/or done flag of a task. A nil pointer
// means "leave that field unchanged".
func (s *Store) Update(id int, title *string, done *bool) (Task, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	t, ok := s.tasks[id]
	if !ok {
		return Task{}, ErrNotFound
	}
	if title != nil {
		t.Title = *title
	}
	if done != nil {
		t.Done = *done
	}
	s.tasks[id] = t
	return t, nil
}

// Delete removes a task, or returns ErrNotFound.
func (s *Store) Delete(id int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.tasks[id]; !ok {
		return ErrNotFound
	}
	delete(s.tasks, id)
	return nil
}
