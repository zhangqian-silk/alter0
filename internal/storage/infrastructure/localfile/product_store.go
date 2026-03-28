package localfile

import (
	"context"
	"path/filepath"
	"sync"

	productapp "alter0/internal/product/application"
	productdomain "alter0/internal/product/domain"
)

type productState struct {
	Items []productdomain.Product `json:"items"`
}

type ProductStore struct {
	path   string
	format Format
	mu     sync.Mutex
}

func NewProductStore(baseDir string, format Format) *ProductStore {
	return &ProductStore{
		path:   filepath.Join(baseDir, "products."+extension(format)),
		format: format,
	}
}

var _ productapp.Store = (*ProductStore)(nil)

func (s *ProductStore) Load(_ context.Context) ([]productdomain.Product, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	raw, ok, err := readIfExists(s.path)
	if err != nil {
		return nil, err
	}
	if !ok {
		return []productdomain.Product{}, nil
	}

	state := productState{}
	if err := unmarshalPayload(s.format, raw, &state); err != nil {
		return nil, err
	}
	if len(state.Items) == 0 {
		return []productdomain.Product{}, nil
	}
	items := make([]productdomain.Product, 0, len(state.Items))
	for _, item := range state.Items {
		items = append(items, item)
	}
	return items, nil
}

func (s *ProductStore) Save(_ context.Context, products []productdomain.Product) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	items := make([]productdomain.Product, 0, len(products))
	for _, item := range products {
		items = append(items, item)
	}
	raw, err := marshalPayload(s.format, "alter0 products", productState{Items: items})
	if err != nil {
		return err
	}
	return writeFile(s.path, raw)
}
