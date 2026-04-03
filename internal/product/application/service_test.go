package application

import (
	"context"
	"testing"

	productdomain "alter0/internal/product/domain"
)

type memoryStore struct {
	items []productdomain.Product
}

func (m *memoryStore) Load(context.Context) ([]productdomain.Product, error) {
	out := make([]productdomain.Product, 0, len(m.items))
	out = append(out, m.items...)
	return out, nil
}

func (m *memoryStore) Save(_ context.Context, items []productdomain.Product) error {
	m.items = append([]productdomain.Product(nil), items...)
	return nil
}

func TestServiceIncludesBuiltinTravel(t *testing.T) {
	service := NewService()
	product, ok := service.ResolveProduct("travel")
	if !ok {
		t.Fatalf("expected builtin travel product")
	}
	if product.MasterAgentID != "travel-master" {
		t.Fatalf("unexpected product: %+v", product)
	}
	if len(product.WorkerAgents) != 0 {
		t.Fatalf("expected builtin travel to run without worker agents, got %+v", product.WorkerAgents)
	}
}

func TestServiceCreateManagedProduct(t *testing.T) {
	store := &memoryStore{}
	service, err := NewServiceWithStore(context.Background(), store)
	if err != nil {
		t.Fatalf("new service failed: %v", err)
	}
	created, err := service.CreateProduct(productdomain.Product{
		Name:          "Research Product",
		Slug:          "research-product",
		Status:        productdomain.StatusDraft,
		Visibility:    productdomain.VisibilityPrivate,
		MasterAgentID: "product-builder",
	})
	if err != nil {
		t.Fatalf("create product failed: %v", err)
	}
	if created.ID != "research-product" {
		t.Fatalf("unexpected id: %+v", created)
	}
	if len(store.items) != 1 || store.items[0].ID != "research-product" {
		t.Fatalf("unexpected persisted items: %+v", store.items)
	}
}

func TestServiceRejectsBuiltinOverwrite(t *testing.T) {
	service := NewService()
	_, err := service.SaveProduct("travel", productdomain.Product{
		Name:          "Travel",
		Status:        productdomain.StatusActive,
		Visibility:    productdomain.VisibilityPublic,
		MasterAgentID: "travel-master",
	})
	if err == nil {
		t.Fatal("expected builtin overwrite to fail")
	}
}
