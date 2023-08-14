package dev

import (
	"sync"

	"github.com/samber/lo"
)

type SafeSlice[T comparable] struct {
	mu    sync.Mutex
	slice []T
}

func (s *SafeSlice[T]) Append(values ...T) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.slice = append(s.slice, values...)
}

func (s *SafeSlice[T]) Remove(value T) {
	s.mu.Lock()
	defer s.mu.Unlock()

	iex := -1
	for i, v := range s.slice {
		if v == value {
			iex = i
		}
	}
	if iex != -1 {
		s.slice = append(s.slice[:iex], s.slice[iex+1:]...)
	}
}

func (s *SafeSlice[T]) Copy() []T {
	s.mu.Lock()
	defer s.mu.Unlock()

	return append([]T(nil), s.slice...)
}

func (s *SafeSlice[T]) Contains(value T) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	return lo.Contains(s.slice, value)
}
