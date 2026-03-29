package application

import (
	"context"
	"testing"

	productdomain "alter0/internal/product/domain"
)

type memoryTravelGuideStore struct {
	items []productdomain.TravelGuide
}

func (m *memoryTravelGuideStore) Load(context.Context) ([]productdomain.TravelGuide, error) {
	return append([]productdomain.TravelGuide(nil), m.items...), nil
}

func (m *memoryTravelGuideStore) Save(_ context.Context, guides []productdomain.TravelGuide) error {
	m.items = append([]productdomain.TravelGuide(nil), guides...)
	return nil
}

func TestTravelGuideServiceCreateGuide(t *testing.T) {
	store := &memoryTravelGuideStore{}
	service, err := NewTravelGuideServiceWithStore(context.Background(), store)
	if err != nil {
		t.Fatalf("new travel guide service failed: %v", err)
	}

	guide, err := service.CreateGuide(productdomain.TravelGuideCreateInput{
		City:        "Shanghai",
		Days:        3,
		TravelStyle: "metro-first",
		Budget:      "mid-range",
		MustVisit:   []string{"The Bund", "Yu Garden"},
	})
	if err != nil {
		t.Fatalf("create guide failed: %v", err)
	}
	if guide.ID == "" {
		t.Fatal("expected guide id")
	}
	if guide.City != "Shanghai" || guide.ProductID != productdomain.TravelProductID {
		t.Fatalf("unexpected guide: %+v", guide)
	}
	if len(guide.DailyRoutes) != 3 {
		t.Fatalf("expected 3 daily routes, got %+v", guide.DailyRoutes)
	}
	if len(guide.MapLayers) == 0 {
		t.Fatalf("expected map layers, got %+v", guide.MapLayers)
	}
	if len(store.items) != 1 {
		t.Fatalf("expected persisted guide, got %+v", store.items)
	}
}

func TestTravelGuideServiceReviseGuide(t *testing.T) {
	service := NewTravelGuideService()
	guide, err := service.CreateGuide(productdomain.TravelGuideCreateInput{
		City:      "Chengdu",
		Days:      2,
		Budget:    "budget",
		MustVisit: []string{"Jinli"},
	})
	if err != nil {
		t.Fatalf("create guide failed: %v", err)
	}

	days := 4
	revised, err := service.ReviseGuide(guide.ID, productdomain.TravelGuideReviseInput{
		Days:                   &days,
		AdditionalRequirements: []string{"more local food", "slow mornings"},
		KeepConditions:         []string{"keep Jinli"},
	})
	if err != nil {
		t.Fatalf("revise guide failed: %v", err)
	}
	if revised.Revision != 2 {
		t.Fatalf("expected revision 2, got %+v", revised)
	}
	if revised.Days != 4 {
		t.Fatalf("expected revised days 4, got %+v", revised)
	}
	if len(revised.AdditionalRequirements) == 0 {
		t.Fatalf("expected additional requirements, got %+v", revised)
	}
	if revised.Content == guide.Content {
		t.Fatalf("expected content to change after revise")
	}
}

func TestTravelGuideServiceListGuidesSortsLatestFirst(t *testing.T) {
	service := NewTravelGuideService()
	_, err := service.CreateGuide(productdomain.TravelGuideCreateInput{City: "Wuhan", Days: 3})
	if err != nil {
		t.Fatalf("create wuhan guide failed: %v", err)
	}
	_, err = service.CreateGuide(productdomain.TravelGuideCreateInput{City: "Beijing", Days: 4})
	if err != nil {
		t.Fatalf("create beijing guide failed: %v", err)
	}

	items := service.ListGuides()
	if len(items) != 2 {
		t.Fatalf("expected 2 guides, got %+v", items)
	}
	if items[0].City != "Beijing" {
		t.Fatalf("expected latest guide first, got %+v", items)
	}
}

func TestTravelGuideServiceRejectsDuplicateCity(t *testing.T) {
	service := NewTravelGuideService()
	if _, err := service.CreateGuide(productdomain.TravelGuideCreateInput{City: "武汉", Days: 3}); err != nil {
		t.Fatalf("create guide failed: %v", err)
	}
	if _, err := service.CreateGuide(productdomain.TravelGuideCreateInput{City: "武汉", Days: 2}); err == nil {
		t.Fatal("expected duplicate city error")
	}
}

func TestTravelGuideServiceUsesUnicodeAwareGuideIDs(t *testing.T) {
	service := NewTravelGuideService()
	guide, err := service.CreateGuide(productdomain.TravelGuideCreateInput{City: "武汉", Days: 3})
	if err != nil {
		t.Fatalf("create guide failed: %v", err)
	}
	if guide.ID != "u6b66-u6c49-guide" {
		t.Fatalf("expected unicode-aware guide id, got %q", guide.ID)
	}
}
