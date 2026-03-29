package localfile

import (
	"context"
	"path/filepath"
	"sync"

	productapp "alter0/internal/product/application"
	productdomain "alter0/internal/product/domain"
)

type productDraftState struct {
	Items []productdomain.ProductDraft `json:"items"`
}

type ProductDraftStore struct {
	path   string
	format Format
	mu     sync.Mutex
}

func NewProductDraftStore(baseDir string, format Format) *ProductDraftStore {
	return &ProductDraftStore{
		path:   filepath.Join(baseDir, "product_drafts."+extension(format)),
		format: format,
	}
}

var _ productapp.DraftStore = (*ProductDraftStore)(nil)

func (s *ProductDraftStore) Load(_ context.Context) ([]productdomain.ProductDraft, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	raw, ok, err := readIfExists(s.path)
	if err != nil {
		return nil, err
	}
	if !ok {
		return []productdomain.ProductDraft{}, nil
	}

	state := productDraftState{}
	if err := unmarshalPayload(s.format, raw, &state); err != nil {
		return nil, err
	}
	if len(state.Items) == 0 {
		return []productdomain.ProductDraft{}, nil
	}
	items := make([]productdomain.ProductDraft, 0, len(state.Items))
	items = append(items, state.Items...)
	return items, nil
}

func (s *ProductDraftStore) Save(_ context.Context, drafts []productdomain.ProductDraft) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	items := make([]productdomain.ProductDraft, 0, len(drafts))
	items = append(items, drafts...)
	raw, err := marshalPayload(s.format, "alter0 product drafts", productDraftState{Items: items})
	if err != nil {
		return err
	}
	return writeFile(s.path, raw)
}
