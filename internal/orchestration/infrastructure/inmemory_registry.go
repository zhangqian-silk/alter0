package infrastructure

import (
	"errors"
	"sort"
	"strings"
	"sync"

	orchdomain "alter0/internal/orchestration/domain"
)

type InMemoryCommandRegistry struct {
	mu       sync.RWMutex
	handlers map[string]orchdomain.CommandHandler
}

func NewInMemoryCommandRegistry() *InMemoryCommandRegistry {
	return &InMemoryCommandRegistry{
		handlers: map[string]orchdomain.CommandHandler{},
	}
}

func (r *InMemoryCommandRegistry) Register(handler orchdomain.CommandHandler) error {
	name := strings.ToLower(strings.TrimSpace(handler.Name()))
	if name == "" {
		return errors.New("handler name cannot be empty")
	}

	keys := []string{name}
	for _, alias := range handler.Aliases() {
		normalized := strings.ToLower(strings.TrimSpace(alias))
		if normalized == "" {
			continue
		}
		keys = append(keys, normalized)
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	for _, key := range keys {
		if _, exists := r.handlers[key]; exists {
			return errors.New("handler already registered: " + key)
		}
	}

	for _, key := range keys {
		r.handlers[key] = handler
	}

	return nil
}

func (r *InMemoryCommandRegistry) Resolve(name string) (orchdomain.CommandHandler, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	handler, ok := r.handlers[strings.ToLower(strings.TrimSpace(name))]
	return handler, ok
}

func (r *InMemoryCommandRegistry) Exists(name string) bool {
	_, ok := r.Resolve(name)
	return ok
}

func (r *InMemoryCommandRegistry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	uniq := map[string]struct{}{}
	for _, handler := range r.handlers {
		uniq[handler.Name()] = struct{}{}
	}

	names := make([]string, 0, len(uniq))
	for name := range uniq {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
