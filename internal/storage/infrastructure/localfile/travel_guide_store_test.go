package localfile

import (
	"context"
	"testing"

	productdomain "alter0/internal/product/domain"
)

func TestTravelGuideStoreRoundTrip(t *testing.T) {
	store := NewTravelGuideStore(t.TempDir(), FormatJSON)
	items := []productdomain.TravelGuide{{
		ID:        "shanghai-guide",
		ProductID: productdomain.TravelProductID,
		City:      "Shanghai",
		Days:      3,
		Content:   "Shanghai 3-day travel guide",
		Revision:  1,
	}}
	if err := store.Save(context.Background(), items); err != nil {
		t.Fatalf("save failed: %v", err)
	}
	loaded, err := store.Load(context.Background())
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}
	if len(loaded) != 1 || loaded[0].ID != "shanghai-guide" {
		t.Fatalf("unexpected guides: %+v", loaded)
	}
}
