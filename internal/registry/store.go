package registry

import "strings"

// Parser is the minimal parser contract needed for registration and lookup.
type Parser interface {
	Bank() string
	CanParse(text string) bool
}

type scoredParser interface {
	DetectionScore(text string) int
}

// Store keeps an ordered list of parsers for detection and explicit lookup.
type Store[T Parser] struct {
	items []T
}

// New creates a new Store with optional initial parsers.
func New[T Parser](items ...T) *Store[T] {
	store := &Store[T]{}
	store.Add(items...)
	return store
}

// Add appends parsers preserving registration order.
func (s *Store[T]) Add(items ...T) {
	s.items = append(s.items, items...)
}

// Len reports the number of registered parsers.
func (s *Store[T]) Len() int {
	return len(s.items)
}

// FindByText returns the first parser that can parse the given text.
func (s *Store[T]) FindByText(text string) (T, bool) {
	parser, _, ok := s.FindByTextWithScore(text)
	return parser, ok
}

// FindByTextWithScore returns the strongest parser match and its score.
func (s *Store[T]) FindByTextWithScore(text string) (T, int, bool) {
	var zero T
	bestIndex := -1
	bestScore := 0

	for i, item := range s.items {
		score := parserScore(item, text)
		if score > bestScore {
			bestIndex = i
			bestScore = score
		}
	}

	if bestIndex == -1 {
		return zero, 0, false
	}

	return s.items[bestIndex], bestScore, true
}

// FindByBank returns the parser whose bank id matches the provided name.
func (s *Store[T]) FindByBank(bank string) (T, bool) {
	var zero T
	normalizedBank := normalizeBank(bank)

	for _, item := range s.items {
		if normalizeBank(item.Bank()) == normalizedBank {
			return item, true
		}
	}

	return zero, false
}

func normalizeBank(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func parserScore[T Parser](item T, text string) int {
	if scored, ok := any(item).(scoredParser); ok {
		return scored.DetectionScore(text)
	}
	if item.CanParse(text) {
		return 1
	}
	return 0
}
