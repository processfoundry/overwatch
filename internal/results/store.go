package results

import (
	"sync"

	"github.com/christianmscott/overwatch/pkg/spec"
)

type Store struct {
	mu      sync.RWMutex
	latest  map[string]spec.CheckResult
	history map[string][]spec.CheckResult
	maxHist int
}

func NewStore(maxHistory int) *Store {
	if maxHistory <= 0 {
		maxHistory = 100
	}
	return &Store{
		latest:  make(map[string]spec.CheckResult),
		history: make(map[string][]spec.CheckResult),
		maxHist: maxHistory,
	}
}

func (s *Store) Record(r spec.CheckResult) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.latest[r.CheckName] = r
	h := s.history[r.CheckName]
	if len(h) >= s.maxHist {
		h = h[1:]
	}
	s.history[r.CheckName] = append(h, r)
}

func (s *Store) Latest(checkName string) (spec.CheckResult, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	r, ok := s.latest[checkName]
	return r, ok
}

func (s *Store) All() map[string]spec.CheckResult {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make(map[string]spec.CheckResult, len(s.latest))
	for k, v := range s.latest {
		out[k] = v
	}
	return out
}
