package set

import "sync"

type Set[T comparable] struct {
	safe bool
	mu   sync.RWMutex
	m    map[T]bool
}

// no op
func nop() {}

func (s *Set[T]) prepare(w bool) func() {
	if !s.safe {
		return nop
	}
	if w {
		s.mu.Lock()
		return func() {
			s.mu.Unlock()
		}
	}
	s.mu.RLock()
	return func() {
		s.mu.RUnlock()
	}
}

func (s *Set[_]) Len() int {
	fn := s.prepare(false)
	defer fn()

	return len(s.m)
}

func (s *Set[T]) Has(el T) bool {
	fn := s.prepare(false)
	defer fn()

	return s.m[el]
}

func (s *Set[T]) Add(el T) bool {
	fn := s.prepare(true)
	defer fn()

	if s.m[el] {
		return false
	}
	s.m[el] = true
	return true
}

func (s *Set[T]) Remove(el T) bool {
	fn := s.prepare(true)
	defer fn()

	if !s.m[el] {
		return false
	}
	delete(s.m, el)
	return true
}

func (s *Set[T]) Clear() {
	fn := s.prepare(true)
	defer fn()

	s.m = map[T]bool{}
}

func (s *Set[T]) For(fn func(el T)) {
	f := s.prepare(true)
	defer f()

	for el := range s.m {
		fn(el)
	}
}

func (s *Set[T]) List() []T {
	l := make([]T, len(s.m))
	i := 0
	for el := range s.m {
		l[i] = el
		i++
	}
	return l
}

func (s *Set[T]) Clone() *Set[T] {
	fn := s.prepare(false)
	defer fn()

	m := make(map[T]bool)
	for el := range s.m {
		m[el] = true
	}
	return &Set[T]{
		safe: s.safe,
		m:    m,
	}
}

type Option[T comparable] interface {
	Apply(s *Set[T])
}

type OptionFunc[T comparable] func(s *Set[T])

func (f OptionFunc[T]) Apply(s *Set[T]) {
	f(s)
}

func New[T comparable](options ...Option[T]) *Set[T] {
	s := &Set[T]{
		m: make(map[T]bool),
	}
	for _, opt := range options {
		opt.Apply(s)
	}
	return s
}

func WithLock[T comparable]() OptionFunc[T] {
	return func(s *Set[T]) {
		s.safe = true
	}
}

func WithElements[T comparable](els []T) OptionFunc[T] {
	return func(s *Set[T]) {
		for _, el := range els {
			s.m[el] = true
		}
	}
}
