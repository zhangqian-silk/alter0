package application

import productdomain "alter0/internal/product/domain"

func builtinProducts() []productdomain.Product {
	return []productdomain.Product{
		{
			ID:            "travel",
			Name:          "Travel",
			Slug:          "travel",
			Summary:       "Travel product for city guides, itinerary planning, metro advice, food recommendations, and map-oriented trip artifacts.",
			Status:        productdomain.StatusActive,
			Visibility:    productdomain.VisibilityPublic,
			OwnerType:     productdomain.OwnerTypeBuiltin,
			MasterAgentID: "travel-master",
			EntryRoute:    "products",
			Tags:          []string{"travel", "city-guide", "itinerary"},
			Version:       productdomain.DefaultVersion,
			WorkerAgents: []productdomain.WorkerAgent{
				{
					AgentID:        "travel-city-guide",
					Role:           "city-guide",
					Responsibility: "Aggregate city overview and scenic spot layering for trip plans.",
					Capabilities:   []string{"poi-curation", "city-summary"},
					Enabled:        true,
				},
				{
					AgentID:        "travel-route-planner",
					Role:           "route-planner",
					Responsibility: "Design day-by-day routes and visit ordering.",
					Capabilities:   []string{"itinerary", "route-ordering"},
					Enabled:        true,
				},
				{
					AgentID:        "travel-metro-guide",
					Role:           "metro-guide",
					Responsibility: "Recommend metro and public transit guidance for each itinerary.",
					Capabilities:   []string{"metro", "public-transit"},
					Enabled:        true,
				},
				{
					AgentID:        "travel-food-recommender",
					Role:           "food-recommender",
					Responsibility: "Recommend food choices and dining distribution around the route.",
					Capabilities:   []string{"food", "dining"},
					Enabled:        true,
				},
				{
					AgentID:        "travel-map-annotator",
					Role:           "map-annotator",
					Responsibility: "Prepare point, line, and layer data for future map highlights.",
					Capabilities:   []string{"map-layers", "route-annotation"},
					Enabled:        true,
				},
			},
			ArtifactTypes:    []string{"city_guide", "itinerary", "map_layers"},
			KnowledgeSources: []string{"city_profile", "poi_catalog", "metro_network", "food_catalog"},
		},
	}
}
