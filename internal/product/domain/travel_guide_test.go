package domain

import (
	"testing"
	"time"
)

func TestTravelGuideNormalizedAppliesTravelDefaultsAndLimits(t *testing.T) {
	guide := TravelGuide{
		ID:                     " guide-1 ",
		ProductID:              " ",
		City:                   " Shanghai ",
		Days:                   20,
		TravelStyle:            " city walk ",
		Budget:                 " mid ",
		Companions:             []string{" friend ", "Friend", ""},
		MustVisit:              []string{" Bund ", "Yu Garden"},
		Avoid:                  []string{" crowd ", "Crowd"},
		AdditionalRequirements: []string{" metro "},
		KeepConditions:         []string{" food "},
		ReplaceConditions:      []string{" hotel "},
		Notes:                  []string{" note ", "note"},
		Content:                "  content  ",
		Revision:               -1,
	}

	normalized := guide.Normalized()

	if normalized.ProductID != TravelProductID {
		t.Fatalf("ProductID = %q, want %q", normalized.ProductID, TravelProductID)
	}
	if normalized.City != "Shanghai" {
		t.Fatalf("City = %q, want Shanghai", normalized.City)
	}
	if normalized.Days != 14 {
		t.Fatalf("Days = %d, want 14", normalized.Days)
	}
	if normalized.Revision != 1 {
		t.Fatalf("Revision = %d, want 1", normalized.Revision)
	}
	if normalized.Content != "content" {
		t.Fatalf("Content = %q, want content", normalized.Content)
	}
	assertStringSlice(t, normalized.Companions, []string{"friend"})
	assertStringSlice(t, normalized.Avoid, []string{"crowd"})
	assertStringSlice(t, normalized.Notes, []string{"note"})
}

func TestTravelGuideValidateRequiresTravelProductCityAndContent(t *testing.T) {
	valid := TravelGuide{
		ID:        "guide-1",
		ProductID: TravelProductID,
		City:      "Shanghai",
		Days:      3,
		Content:   "content",
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	if err := valid.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}

	cases := []struct {
		name    string
		mutate  func(TravelGuide) TravelGuide
		wantErr string
	}{
		{
			name:    "wrong product",
			mutate:  func(g TravelGuide) TravelGuide { g.ProductID = "other"; return g },
			wantErr: "travel guide product_id must be travel",
		},
		{
			name:    "blank city",
			mutate:  func(g TravelGuide) TravelGuide { g.City = " "; return g },
			wantErr: "travel guide city is required",
		},
		{
			name:    "blank content",
			mutate:  func(g TravelGuide) TravelGuide { g.Content = " "; return g },
			wantErr: "travel guide content is required",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.mutate(valid).Validate()
			if err == nil {
				t.Fatalf("Validate() error = nil, want containing %q", tc.wantErr)
			}
			assertErrorContains(t, err, tc.wantErr)
		})
	}
}

func TestTravelGuideCreateInputNormalizeAndValidate(t *testing.T) {
	input := TravelGuideCreateInput{
		City:                   " Beijing ",
		Days:                   -1,
		TravelStyle:            " museum ",
		Budget:                 " high ",
		Companions:             []string{" family ", "Family"},
		MustVisit:              []string{" Forbidden City "},
		Avoid:                  []string{" crowd "},
		AdditionalRequirements: []string{" metro "},
	}

	normalized := input.Normalized()

	if normalized.City != "Beijing" {
		t.Fatalf("City = %q, want Beijing", normalized.City)
	}
	if normalized.Days != 3 {
		t.Fatalf("Days = %d, want 3", normalized.Days)
	}
	assertStringSlice(t, normalized.Companions, []string{"family"})
	if err := normalized.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if err := (TravelGuideCreateInput{City: " "}).Validate(); err == nil {
		t.Fatalf("Validate() error = nil, want city error")
	}
}

func TestTravelGuideReviseInputNormalizesOptionalDays(t *testing.T) {
	days := 30
	input := TravelGuideReviseInput{
		Days:              &days,
		TravelStyle:       " slow ",
		KeepConditions:    []string{" metro ", "metro"},
		ReplaceConditions: []string{" hotel "},
	}

	normalized := input.Normalized()

	if normalized.Days == nil || *normalized.Days != 14 {
		t.Fatalf("Days = %v, want 14", normalized.Days)
	}
	if normalized.TravelStyle != "slow" {
		t.Fatalf("TravelStyle = %q, want slow", normalized.TravelStyle)
	}
	assertStringSlice(t, normalized.KeepConditions, []string{"metro"})
	assertStringSlice(t, normalized.ReplaceConditions, []string{"hotel"})
	if days != 30 {
		t.Fatalf("input days mutated = %d, want 30", days)
	}
}
