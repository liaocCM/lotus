package store

import (
	"fmt"
	"sync"
	"time"

	"example.com/crud-api/internal/model"
)

// MemoryStore is an in-memory product store. Use this for the benchmark
// instead of a real database — the task is to implement the handler layer.
type MemoryStore struct {
	mu       sync.RWMutex
	products map[string]model.Product
	nextID   int
}

func New() *MemoryStore {
	return &MemoryStore{
		products: make(map[string]model.Product),
	}
}

func (s *MemoryStore) List(offset, limit int) ([]model.Product, int) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	all := make([]model.Product, 0, len(s.products))
	for _, p := range s.products {
		all = append(all, p)
	}
	total := len(all)
	if offset >= total {
		return nil, total
	}
	end := offset + limit
	if end > total {
		end = total
	}
	return all[offset:end], total
}

func (s *MemoryStore) Get(id string) (model.Product, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	p, ok := s.products[id]
	return p, ok
}

func (s *MemoryStore) Create(p model.Product) model.Product {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.nextID++
	p.ID = fmt.Sprintf("%d", s.nextID)
	p.CreatedAt = time.Now()
	p.UpdatedAt = time.Now()
	s.products[p.ID] = p
	return p
}

func (s *MemoryStore) Update(id string, p model.Product) (model.Product, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	existing, ok := s.products[id]
	if !ok {
		return model.Product{}, false
	}
	p.ID = id
	p.CreatedAt = existing.CreatedAt
	p.UpdatedAt = time.Now()
	s.products[id] = p
	return p, true
}

func (s *MemoryStore) Delete(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.products[id]
	if ok {
		delete(s.products, id)
	}
	return ok
}
