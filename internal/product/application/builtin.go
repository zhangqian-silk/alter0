package application

import productdomain "alter0/internal/product/domain"

func builtinProducts() []productdomain.Product {
	return []productdomain.Product{
		{
			ID:               "travel",
			Name:             "Travel",
			Slug:             "travel",
			Summary:          "Travel product with a single travel-master agent for city guides, itinerary planning, transit advice, food recommendations, and map-oriented trip artifacts.",
			Status:           productdomain.StatusActive,
			Visibility:       productdomain.VisibilityPublic,
			OwnerType:        productdomain.OwnerTypeBuiltin,
			MasterAgentID:    "travel-master",
			EntryRoute:       "products",
			Tags:             []string{"travel", "city-guide", "itinerary"},
			Version:          productdomain.DefaultVersion,
			ArtifactTypes:    []string{"city_guide", "itinerary", "map_layers"},
			KnowledgeSources: []string{"city_profile", "poi_catalog", "metro_network", "food_catalog"},
		},
	}
}
