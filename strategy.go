package main

import (
	"errors"
	"fmt"
	"log"
)

type RoundRobinStrategy struct {
	backends []Backend
	idx      int
}

func (s *RoundRobinStrategy) NextBackend() (Backend, error) {
	n := len(s.backends)

	if n == 0 {
		return Backend{}, errors.New("Backends not found. Skipping.")
	}

	if n == 1 {
		return s.backends[0], nil
	} else {
		s.idx = (s.idx + 1) % n
		return s.backends[s.idx], nil
	}
}

func (s *RoundRobinStrategy) AddBackend(backend Backend) {
	s.backends = append(s.backends, backend)
}

func (s *RoundRobinStrategy) SetBackends(backends []Backend) {
	s.backends = backends
}

func (s *RoundRobinStrategy) DeleteBackend(id string) error {
	log.Printf("Strategy: delete backend")
	for i, backend := range s.backends {
		if backend.Id == id {
			s.backends = append(s.backends[:i], s.backends[i+1:]...)
			return nil
		}
	}

	return errors.New(fmt.Sprintf("Unknown backend id: %s", id))
}

type BackendStrategy interface {
	NextBackend() (Backend, error)
	AddBackend(backend Backend)
	DeleteBackend(id string) error
	SetBackends(backends []Backend)
}
