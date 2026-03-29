package domain

import (
	"errors"
	"sort"
	"strings"
	"time"
)

const TravelProductID = "travel"

type TravelGuide struct {
	ID                     string             `json:"id"`
	ProductID              string             `json:"product_id"`
	City                   string             `json:"city"`
	Days                   int                `json:"days"`
	TravelStyle            string             `json:"travel_style,omitempty"`
	Budget                 string             `json:"budget,omitempty"`
	Companions             []string           `json:"companions,omitempty"`
	MustVisit              []string           `json:"must_visit,omitempty"`
	Avoid                  []string           `json:"avoid,omitempty"`
	AdditionalRequirements []string           `json:"additional_requirements,omitempty"`
	KeepConditions         []string           `json:"keep_conditions,omitempty"`
	ReplaceConditions      []string           `json:"replace_conditions,omitempty"`
	POIs                   []TravelPOI        `json:"pois,omitempty"`
	MetroLines             []TravelMetroLine  `json:"metro_lines,omitempty"`
	DailyRoutes            []TravelDailyRoute `json:"daily_routes,omitempty"`
	Foods                  []TravelFood       `json:"foods,omitempty"`
	Notes                  []string           `json:"notes,omitempty"`
	MapLayers              []TravelMapLayer   `json:"map_layers,omitempty"`
	Content                string             `json:"content"`
	Revision               int                `json:"revision"`
	CreatedAt              time.Time          `json:"created_at"`
	UpdatedAt              time.Time          `json:"updated_at"`
}

type TravelPOI struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	District    string   `json:"district,omitempty"`
	Category    string   `json:"category,omitempty"`
	Highlights  []string `json:"highlights,omitempty"`
	VisitLength string   `json:"visit_length,omitempty"`
}

type TravelMetroLine struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	Segments  []string `json:"segments,omitempty"`
	Reasoning string   `json:"reasoning,omitempty"`
}

type TravelDailyRoute struct {
	Day        int      `json:"day"`
	Theme      string   `json:"theme,omitempty"`
	Stops      []string `json:"stops,omitempty"`
	Transit    []string `json:"transit,omitempty"`
	DiningPlan []string `json:"dining_plan,omitempty"`
}

type TravelFood struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	Area      string   `json:"area,omitempty"`
	Signature []string `json:"signature,omitempty"`
	When      string   `json:"when,omitempty"`
}

type TravelMapLayer struct {
	ID          string   `json:"id"`
	Type        string   `json:"type"`
	Label       string   `json:"label"`
	References  []string `json:"references,omitempty"`
	Description string   `json:"description,omitempty"`
}

type TravelGuideCreateInput struct {
	City                   string   `json:"city"`
	Days                   int      `json:"days"`
	TravelStyle            string   `json:"travel_style,omitempty"`
	Budget                 string   `json:"budget,omitempty"`
	Companions             []string `json:"companions,omitempty"`
	MustVisit              []string `json:"must_visit,omitempty"`
	Avoid                  []string `json:"avoid,omitempty"`
	AdditionalRequirements []string `json:"additional_requirements,omitempty"`
}

type TravelGuideReviseInput struct {
	Days                   *int     `json:"days,omitempty"`
	TravelStyle            string   `json:"travel_style,omitempty"`
	Budget                 string   `json:"budget,omitempty"`
	Companions             []string `json:"companions,omitempty"`
	MustVisit              []string `json:"must_visit,omitempty"`
	Avoid                  []string `json:"avoid,omitempty"`
	AdditionalRequirements []string `json:"additional_requirements,omitempty"`
	KeepConditions         []string `json:"keep_conditions,omitempty"`
	ReplaceConditions      []string `json:"replace_conditions,omitempty"`
}

func (g TravelGuide) Normalized() TravelGuide {
	out := g
	out.ID = strings.TrimSpace(out.ID)
	out.ProductID = strings.TrimSpace(out.ProductID)
	if out.ProductID == "" {
		out.ProductID = TravelProductID
	}
	out.City = strings.TrimSpace(out.City)
	if out.Days <= 0 {
		out.Days = 3
	}
	if out.Days > 14 {
		out.Days = 14
	}
	out.TravelStyle = strings.TrimSpace(out.TravelStyle)
	out.Budget = strings.TrimSpace(out.Budget)
	out.Companions = normalizeTravelStringList(out.Companions)
	out.MustVisit = normalizeTravelStringList(out.MustVisit)
	out.Avoid = normalizeTravelStringList(out.Avoid)
	out.AdditionalRequirements = normalizeTravelStringList(out.AdditionalRequirements)
	out.KeepConditions = normalizeTravelStringList(out.KeepConditions)
	out.ReplaceConditions = normalizeTravelStringList(out.ReplaceConditions)
	out.Notes = normalizeTravelStringList(out.Notes)
	out.Content = strings.TrimSpace(out.Content)
	if out.Revision < 1 {
		out.Revision = 1
	}
	return out
}

func (g TravelGuide) Validate() error {
	normalized := g.Normalized()
	if normalized.ProductID != TravelProductID {
		return errors.New("travel guide product_id must be travel")
	}
	if normalized.City == "" {
		return errors.New("travel guide city is required")
	}
	if normalized.Days <= 0 {
		return errors.New("travel guide days must be greater than 0")
	}
	if strings.TrimSpace(normalized.Content) == "" {
		return errors.New("travel guide content is required")
	}
	return nil
}

func (in TravelGuideCreateInput) Normalized() TravelGuideCreateInput {
	out := in
	out.City = strings.TrimSpace(out.City)
	if out.Days <= 0 {
		out.Days = 3
	}
	if out.Days > 14 {
		out.Days = 14
	}
	out.TravelStyle = strings.TrimSpace(out.TravelStyle)
	out.Budget = strings.TrimSpace(out.Budget)
	out.Companions = normalizeTravelStringList(out.Companions)
	out.MustVisit = normalizeTravelStringList(out.MustVisit)
	out.Avoid = normalizeTravelStringList(out.Avoid)
	out.AdditionalRequirements = normalizeTravelStringList(out.AdditionalRequirements)
	return out
}

func (in TravelGuideCreateInput) Validate() error {
	normalized := in.Normalized()
	if normalized.City == "" {
		return errors.New("city is required")
	}
	if normalized.Days <= 0 {
		return errors.New("days must be greater than 0")
	}
	return nil
}

func (in TravelGuideReviseInput) Normalized() TravelGuideReviseInput {
	out := in
	if out.Days != nil {
		value := *out.Days
		if value <= 0 {
			value = 3
		}
		if value > 14 {
			value = 14
		}
		out.Days = &value
	}
	out.TravelStyle = strings.TrimSpace(out.TravelStyle)
	out.Budget = strings.TrimSpace(out.Budget)
	out.Companions = normalizeTravelStringList(out.Companions)
	out.MustVisit = normalizeTravelStringList(out.MustVisit)
	out.Avoid = normalizeTravelStringList(out.Avoid)
	out.AdditionalRequirements = normalizeTravelStringList(out.AdditionalRequirements)
	out.KeepConditions = normalizeTravelStringList(out.KeepConditions)
	out.ReplaceConditions = normalizeTravelStringList(out.ReplaceConditions)
	return out
}

func normalizeTravelStringList(items []string) []string {
	if len(items) == 0 {
		return nil
	}
	out := make([]string, 0, len(items))
	seen := map[string]struct{}{}
	for _, item := range items {
		trimmed := strings.TrimSpace(item)
		if trimmed == "" {
			continue
		}
		key := strings.ToLower(trimmed)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, trimmed)
	}
	if len(out) == 0 {
		return nil
	}
	sort.Strings(out)
	return out
}
