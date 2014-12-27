package main

type RoundRobinStrategy struct {
	backends []Backend
	idx      int
}

func (s *RoundRobinStrategy) NextBackend() Backend {
	n := len(s.backends)

	if n == 1 {
		return s.backends[0]
	} else {
		s.idx = (s.idx + 1) % n
		return s.backends[s.idx]
	}
}
