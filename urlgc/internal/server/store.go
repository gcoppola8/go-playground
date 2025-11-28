package server

import (
	"fmt"
	"sync"
)

// Store is an in-memory store for URL mappings
type Store struct {
	mu     sync.RWMutex
	byCode map[string]string // code -> targetURL
}

// NewStore creates a new in-memory store
func NewStore() *Store {
	return &Store{byCode: make(map[string]string)}
}

// Put stores a code-to-URL mapping in memory
func (s *Store) Put(code, target string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.byCode[code]; exists {
		return fmt.Errorf("code already exists")
	}
	s.byCode[code] = target
	return nil
}

// Get retrieves a target URL by code from memory
func (s *Store) Get(code string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.byCode[code]
	return v, ok
}

// GetAll returns a copy of all mappings
func (s *Store) GetAll() map[string]string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	result := make(map[string]string, len(s.byCode))
	for k, v := range s.byCode {
		result[k] = v
	}
	return result
}

// LoadAll replaces all mappings with the provided data
func (s *Store) LoadAll(data map[string]string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	s.byCode = make(map[string]string, len(data))
	for k, v := range data {
		s.byCode[k] = v
	}
}
