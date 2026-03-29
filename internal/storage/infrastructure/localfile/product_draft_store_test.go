package localfile

import (
	"context"
	"testing"

	productdomain "alter0/internal/product/domain"
)

func TestProductDraftStoreRoundTrip(t *testing.T) {
	store := NewProductDraftStore(t.TempDir(), FormatJSON)
	items := []productdomain.ProductDraft{
		{
			DraftID:      "travel-premium-draft",
			Mode:         productdomain.GenerationModeBootstrap,
			ReviewStatus: productdomain.ReviewStatusDraft,
			GeneratedBy:  "product-builder",
			Product: productdomain.Product{
				ID:            "travel-premium",
				Name:          "Travel Premium",
				Slug:          "travel-premium",
				Status:        productdomain.StatusDraft,
				Visibility:    productdomain.VisibilityPrivate,
				OwnerType:     productdomain.OwnerTypeManaged,
				MasterAgentID: "travel-premium-master",
				Version:       productdomain.DefaultVersion,
			},
			MasterAgent: productdomain.ProductAgentDraft{
				AgentID: "travel-premium-master",
				Name:    "Travel Premium Master",
			},
		},
	}
	if err := store.Save(context.Background(), items); err != nil {
		t.Fatalf("save drafts failed: %v", err)
	}
	loaded, err := store.Load(context.Background())
	if err != nil {
		t.Fatalf("load drafts failed: %v", err)
	}
	if len(loaded) != 1 || loaded[0].DraftID != "travel-premium-draft" {
		t.Fatalf("unexpected loaded drafts: %+v", loaded)
	}
}
