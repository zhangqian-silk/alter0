package localfile

import (
	"context"
	"testing"

	productdomain "alter0/internal/product/domain"
)

func TestProductStoreJSONRoundTrip(t *testing.T) {
	store := NewProductStore(t.TempDir(), FormatJSON)
	products := []productdomain.Product{
		{
			ID:            "travel-lab",
			Name:          "Travel Lab",
			Slug:          "travel-lab",
			Summary:       "Managed travel product",
			Status:        productdomain.StatusDraft,
			Visibility:    productdomain.VisibilityPrivate,
			OwnerType:     productdomain.OwnerTypeManaged,
			MasterAgentID: "travel-master",
			EntryRoute:    "products",
			Tags:          []string{"travel", "lab"},
			Version:       productdomain.DefaultVersion,
			WorkerAgents: []productdomain.WorkerAgent{
				{AgentID: "travel-route-planner", Role: "route-planner", Enabled: true},
			},
			ArtifactTypes:    []string{"city_guide"},
			KnowledgeSources: []string{"poi_catalog"},
		},
	}
	if err := store.Save(context.Background(), products); err != nil {
		t.Fatalf("save failed: %v", err)
	}
	loaded, err := store.Load(context.Background())
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}
	if len(loaded) != 1 || loaded[0].ID != "travel-lab" {
		t.Fatalf("unexpected products: %+v", loaded)
	}
	if loaded[0].MasterAgentID != "travel-master" {
		t.Fatalf("unexpected master agent: %+v", loaded[0])
	}
}
