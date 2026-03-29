package application

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"

	productdomain "alter0/internal/product/domain"
)

type Store interface {
	Load(ctx context.Context) ([]productdomain.Product, error)
	Save(ctx context.Context, products []productdomain.Product) error
}

type Service struct {
	mu      sync.RWMutex
	store   Store
	builtin map[string]productdomain.Product
	managed map[string]productdomain.Product
}

func NewService() *Service {
	return newService(nil)
}

func NewServiceWithStore(ctx context.Context, store Store) (*Service, error) {
	service := newService(store)
	if store == nil {
		return service, nil
	}
	items, err := store.Load(ctx)
	if err != nil {
		return nil, fmt.Errorf("load products: %w", err)
	}
	for _, item := range items {
		normalized := item.Normalized()
		if normalized.ID == "" {
			continue
		}
		if normalized.OwnerType == productdomain.OwnerTypeBuiltin {
			normalized.OwnerType = productdomain.OwnerTypeManaged
		}
		if err := normalized.Validate(); err != nil {
			return nil, fmt.Errorf("invalid product in store: %w", err)
		}
		service.managed[normalized.ID] = cloneProduct(normalized)
	}
	return service, nil
}

func newService(store Store) *Service {
	builtins := builtinProducts()
	builtinMap := make(map[string]productdomain.Product, len(builtins))
	for _, item := range builtins {
		normalized := item.Normalized()
		if normalized.OwnerType == "" {
			normalized.OwnerType = productdomain.OwnerTypeBuiltin
		}
		builtinMap[normalized.ID] = cloneProduct(normalized)
	}
	return &Service{
		store:   store,
		builtin: builtinMap,
		managed: map[string]productdomain.Product{},
	}
}

func (s *Service) IsBuiltinID(id string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.builtin[normalizeID(id)]
	return ok
}

func (s *Service) ResolveProduct(id string) (productdomain.Product, bool) {
	key := normalizeID(id)
	s.mu.RLock()
	defer s.mu.RUnlock()
	if item, ok := s.builtin[key]; ok {
		return cloneProduct(item), true
	}
	item, ok := s.managed[key]
	if !ok {
		return productdomain.Product{}, false
	}
	return cloneProduct(item), true
}

func (s *Service) ListProducts() []productdomain.Product {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.listLocked(false)
}

func (s *Service) ListPublicProducts() []productdomain.Product {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := s.listLocked(false)
	filtered := make([]productdomain.Product, 0, len(items))
	for _, item := range items {
		if item.Status != productdomain.StatusActive || item.Visibility != productdomain.VisibilityPublic {
			continue
		}
		filtered = append(filtered, item)
	}
	return filtered
}

func (s *Service) CreateProduct(product productdomain.Product) (productdomain.Product, error) {
	name := strings.TrimSpace(product.Name)
	if name == "" {
		return productdomain.Product{}, errors.New("product name is required")
	}
	base := productIDBase(product.Slug)
	if base == "" {
		base = productIDBase(name)
	}
	if base == "" {
		return productdomain.Product{}, errors.New("product id is required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	candidate := base
	for index := 2; ; index++ {
		if !s.productIDExistsLocked(candidate) {
			product.ID = candidate
			break
		}
		candidate = fmt.Sprintf("%s-%d", base, index)
	}
	product.OwnerType = productdomain.OwnerTypeManaged
	product.Version = productdomain.DefaultVersion
	normalized := product.Normalized()
	if err := normalized.Validate(); err != nil {
		return productdomain.Product{}, err
	}
	s.managed[normalized.ID] = cloneProduct(normalized)
	if err := s.storeLocked(); err != nil {
		delete(s.managed, normalized.ID)
		return productdomain.Product{}, err
	}
	return cloneProduct(normalized), nil
}

func (s *Service) SaveProduct(id string, product productdomain.Product) (productdomain.Product, error) {
	key := normalizeID(id)
	if key == "" {
		return productdomain.Product{}, errors.New("product id is required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.builtin[key]; ok {
		return productdomain.Product{}, errors.New("builtin products are managed by the service and cannot be overwritten")
	}
	previous, existed := s.managed[key]
	product.ID = key
	product.OwnerType = productdomain.OwnerTypeManaged
	if existed {
		product.Version = nextVersion(previous.Version)
	} else {
		product.Version = productdomain.DefaultVersion
	}
	normalized := product.Normalized()
	if err := normalized.Validate(); err != nil {
		return productdomain.Product{}, err
	}
	s.managed[key] = cloneProduct(normalized)
	if err := s.storeLocked(); err != nil {
		if existed {
			s.managed[key] = previous
		} else {
			delete(s.managed, key)
		}
		return productdomain.Product{}, err
	}
	return cloneProduct(normalized), nil
}

func (s *Service) DeleteProduct(id string) bool {
	key := normalizeID(id)
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.builtin[key]; ok {
		return false
	}
	previous, ok := s.managed[key]
	if !ok {
		return false
	}
	delete(s.managed, key)
	if err := s.storeLocked(); err != nil {
		s.managed[key] = previous
		return false
	}
	return true
}

func (s *Service) listLocked(managedOnly bool) []productdomain.Product {
	items := make([]productdomain.Product, 0, len(s.builtin)+len(s.managed))
	if !managedOnly {
		for _, item := range s.builtin {
			items = append(items, cloneProduct(item))
		}
	}
	for _, item := range s.managed {
		items = append(items, cloneProduct(item))
	}
	sort.Slice(items, func(i, j int) bool {
		left := items[i]
		right := items[j]
		if left.OwnerType != right.OwnerType {
			return left.OwnerType < right.OwnerType
		}
		if left.Name == right.Name {
			return left.ID < right.ID
		}
		return left.Name < right.Name
	})
	return items
}

func (s *Service) productIDExistsLocked(id string) bool {
	key := normalizeID(id)
	if key == "" {
		return false
	}
	if _, ok := s.builtin[key]; ok {
		return true
	}
	_, ok := s.managed[key]
	return ok
}

func (s *Service) storeLocked() error {
	if s == nil || s.store == nil {
		return nil
	}
	items := s.listLocked(true)
	return s.store.Save(context.Background(), items)
}

func cloneProduct(item productdomain.Product) productdomain.Product {
	cloned := item.Normalized()
	if len(item.WorkerAgents) > 0 {
		cloned.WorkerAgents = make([]productdomain.WorkerAgent, 0, len(item.WorkerAgents))
		for _, worker := range item.WorkerAgents {
			cloned.WorkerAgents = append(cloned.WorkerAgents, productdomain.WorkerAgent{
				AgentID:        normalizeID(worker.AgentID),
				Role:           strings.TrimSpace(worker.Role),
				Responsibility: strings.TrimSpace(worker.Responsibility),
				Capabilities:   cloneStringList(worker.Capabilities),
				Enabled:        worker.Enabled,
			})
		}
	}
	cloned.Tags = cloneStringList(item.Tags)
	cloned.ArtifactTypes = cloneStringList(item.ArtifactTypes)
	cloned.KnowledgeSources = cloneStringList(item.KnowledgeSources)
	return cloned
}

func cloneStringList(items []string) []string {
	if len(items) == 0 {
		return nil
	}
	out := make([]string, 0, len(items))
	for _, item := range items {
		trimmed := strings.TrimSpace(item)
		if trimmed == "" {
			continue
		}
		out = append(out, trimmed)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func normalizeID(raw string) string {
	return strings.ToLower(strings.TrimSpace(raw))
}

func productIDBase(raw string) string {
	trimmed := normalizeID(raw)
	if trimmed == "" {
		return ""
	}
	var builder strings.Builder
	lastHyphen := false
	for _, r := range trimmed {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			builder.WriteRune(r)
			lastHyphen = false
			continue
		}
		if !lastHyphen {
			builder.WriteByte('-')
			lastHyphen = true
		}
	}
	return strings.Trim(builder.String(), "-")
}

func nextVersion(raw string) string {
	trimmed := strings.TrimPrefix(strings.TrimSpace(raw), "v")
	parts := strings.Split(trimmed, ".")
	if len(parts) != 3 {
		return productdomain.DefaultVersion
	}
	major, err1 := strconv.Atoi(parts[0])
	minor, err2 := strconv.Atoi(parts[1])
	patch, err3 := strconv.Atoi(parts[2])
	if err1 != nil || err2 != nil || err3 != nil {
		return productdomain.DefaultVersion
	}
	patch++
	return fmt.Sprintf("v%d.%d.%d", major, minor, patch)
}
