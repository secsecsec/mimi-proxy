package main

import (
	"errors"
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

	log.Printf("%d", n)
	for _, b := range s.backends {
		log.Printf("%v %d %d", b, b.Id, b.Url)
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

func (s *RoundRobinStrategy) DeleteBackend(id string) {
	var index int
	for i, backend := range s.backends {
		if backend.Id == id {
			index = i
			break
		}
	}

	s.backends = append(s.backends[:index], s.backends[index+1:]...)
}

type BackendStrategy interface {
	NextBackend() (Backend, error)
	AddBackend(backend Backend)
	DeleteBackend(id string)
	SetBackends(backends []Backend)
}
