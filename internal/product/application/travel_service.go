package application

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	productdomain "alter0/internal/product/domain"
)

type TravelGuideStore interface {
	Load(ctx context.Context) ([]productdomain.TravelGuide, error)
	Save(ctx context.Context, guides []productdomain.TravelGuide) error
}

type TravelGuideService struct {
	mu     sync.RWMutex
	store  TravelGuideStore
	guides map[string]productdomain.TravelGuide
	now    func() time.Time
}

func NewTravelGuideService() *TravelGuideService {
	return newTravelGuideService(nil)
}

func NewTravelGuideServiceWithStore(ctx context.Context, store TravelGuideStore) (*TravelGuideService, error) {
	service := newTravelGuideService(store)
	if store == nil {
		return service, nil
	}
	items, err := store.Load(ctx)
	if err != nil {
		return nil, fmt.Errorf("load travel guides: %w", err)
	}
	for _, item := range items {
		normalized := item.Normalized()
		if normalized.ID == "" {
			continue
		}
		if err := normalized.Validate(); err != nil {
			return nil, fmt.Errorf("invalid travel guide in store: %w", err)
		}
		service.guides[normalized.ID] = cloneTravelGuide(normalized)
	}
	return service, nil
}

func newTravelGuideService(store TravelGuideStore) *TravelGuideService {
	return &TravelGuideService{
		store:  store,
		guides: map[string]productdomain.TravelGuide{},
		now: func() time.Time {
			return time.Now().UTC()
		},
	}
}

func (s *TravelGuideService) GetGuide(id string) (productdomain.TravelGuide, bool) {
	key := normalizeTravelGuideID(id)
	s.mu.RLock()
	defer s.mu.RUnlock()
	item, ok := s.guides[key]
	if !ok {
		return productdomain.TravelGuide{}, false
	}
	return cloneTravelGuide(item), true
}

func (s *TravelGuideService) CreateGuide(input productdomain.TravelGuideCreateInput) (productdomain.TravelGuide, error) {
	normalized := input.Normalized()
	if err := normalized.Validate(); err != nil {
		return productdomain.TravelGuide{}, err
	}
	now := s.now().UTC()

	s.mu.Lock()
	defer s.mu.Unlock()
	guide := s.buildGuideLocked("", productdomain.TravelGuide{}, normalized, productdomain.TravelGuideReviseInput{}, now)
	s.guides[guide.ID] = cloneTravelGuide(guide)
	if err := s.storeLocked(); err != nil {
		delete(s.guides, guide.ID)
		return productdomain.TravelGuide{}, err
	}
	return cloneTravelGuide(guide), nil
}

func (s *TravelGuideService) ReviseGuide(id string, input productdomain.TravelGuideReviseInput) (productdomain.TravelGuide, error) {
	key := normalizeTravelGuideID(id)
	if key == "" {
		return productdomain.TravelGuide{}, errors.New("guide id is required")
	}
	revise := input.Normalized()
	now := s.now().UTC()

	s.mu.Lock()
	defer s.mu.Unlock()
	previous, ok := s.guides[key]
	if !ok {
		return productdomain.TravelGuide{}, errors.New("travel guide not found")
	}
	updated := s.buildGuideLocked(key, previous, productdomain.TravelGuideCreateInput{}, revise, now)
	s.guides[key] = cloneTravelGuide(updated)
	if err := s.storeLocked(); err != nil {
		s.guides[key] = previous
		return productdomain.TravelGuide{}, err
	}
	return cloneTravelGuide(updated), nil
}

func (s *TravelGuideService) buildGuideLocked(id string, previous productdomain.TravelGuide, create productdomain.TravelGuideCreateInput, revise productdomain.TravelGuideReviseInput, now time.Time) productdomain.TravelGuide {
	isRevision := strings.TrimSpace(previous.ID) != ""
	baseCity := create.City
	if isRevision {
		baseCity = previous.City
	}
	guideID := strings.TrimSpace(id)
	if guideID == "" {
		guideID = s.nextGuideIDLocked(baseCity)
	}

	days := create.Days
	if isRevision {
		days = previous.Days
	}
	if revise.Days != nil {
		days = *revise.Days
	}
	if days <= 0 {
		days = 3
	}

	travelStyle := strings.TrimSpace(create.TravelStyle)
	if isRevision {
		travelStyle = previous.TravelStyle
	}
	if strings.TrimSpace(revise.TravelStyle) != "" {
		travelStyle = strings.TrimSpace(revise.TravelStyle)
	}
	if travelStyle == "" {
		travelStyle = "balanced"
	}

	budget := strings.TrimSpace(create.Budget)
	if isRevision {
		budget = previous.Budget
	}
	if strings.TrimSpace(revise.Budget) != "" {
		budget = strings.TrimSpace(revise.Budget)
	}
	if budget == "" {
		budget = "mid-range"
	}

	companions := create.Companions
	if isRevision {
		companions = append([]string(nil), previous.Companions...)
	}
	if len(revise.Companions) > 0 {
		companions = unionTravelLists(companions, revise.Companions)
	}

	mustVisit := create.MustVisit
	if isRevision {
		mustVisit = append([]string(nil), previous.MustVisit...)
	}
	if len(revise.MustVisit) > 0 {
		mustVisit = unionTravelLists(mustVisit, revise.MustVisit)
	}
	mustVisit = ensureTravelMustVisit(baseCity, mustVisit, days)

	avoid := create.Avoid
	if isRevision {
		avoid = append([]string(nil), previous.Avoid...)
	}
	if len(revise.Avoid) > 0 {
		avoid = unionTravelLists(avoid, revise.Avoid)
	}

	additionalRequirements := create.AdditionalRequirements
	if isRevision {
		additionalRequirements = append([]string(nil), previous.AdditionalRequirements...)
	}
	if len(revise.AdditionalRequirements) > 0 {
		additionalRequirements = unionTravelLists(additionalRequirements, revise.AdditionalRequirements)
	}

	keepConditions := revise.KeepConditions
	if isRevision {
		keepConditions = unionTravelLists(previous.KeepConditions, revise.KeepConditions)
	}
	replaceConditions := revise.ReplaceConditions
	if isRevision {
		replaceConditions = unionTravelLists(previous.ReplaceConditions, revise.ReplaceConditions)
	}

	pois := buildTravelPOIs(baseCity, mustVisit)
	metroLines := buildTravelMetroLines(baseCity, days, mustVisit)
	foods := buildTravelFoods(baseCity, budget, days)
	routes := buildTravelDailyRoutes(days, pois, metroLines, foods)
	notes := buildTravelNotes(baseCity, travelStyle, budget, companions, avoid, additionalRequirements, keepConditions, replaceConditions)
	mapLayers := buildTravelMapLayers(baseCity, pois, routes, metroLines, foods)

	guide := productdomain.TravelGuide{
		ID:                     guideID,
		ProductID:              productdomain.TravelProductID,
		City:                   strings.TrimSpace(baseCity),
		Days:                   days,
		TravelStyle:            travelStyle,
		Budget:                 budget,
		Companions:             companions,
		MustVisit:              mustVisit,
		Avoid:                  avoid,
		AdditionalRequirements: additionalRequirements,
		KeepConditions:         keepConditions,
		ReplaceConditions:      replaceConditions,
		POIs:                   pois,
		MetroLines:             metroLines,
		DailyRoutes:            routes,
		Foods:                  foods,
		Notes:                  notes,
		MapLayers:              mapLayers,
		Revision:               1,
		CreatedAt:              now,
		UpdatedAt:              now,
	}
	if isRevision {
		guide.Revision = previous.Revision + 1
		guide.CreatedAt = previous.CreatedAt
	}
	guide.Content = renderTravelGuideContent(guide)
	return guide.Normalized()
}

func (s *TravelGuideService) nextGuideIDLocked(city string) string {
	base := travelGuideIDBase(city)
	if base == "" {
		base = "travel-guide"
	}
	candidate := base
	for index := 2; ; index++ {
		if _, ok := s.guides[candidate]; !ok {
			return candidate
		}
		candidate = fmt.Sprintf("%s-%d", base, index)
	}
}

func (s *TravelGuideService) storeLocked() error {
	if s == nil || s.store == nil {
		return nil
	}
	items := make([]productdomain.TravelGuide, 0, len(s.guides))
	for _, guide := range s.guides {
		items = append(items, cloneTravelGuide(guide))
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].UpdatedAt.Equal(items[j].UpdatedAt) {
			return items[i].ID < items[j].ID
		}
		return items[i].UpdatedAt.Before(items[j].UpdatedAt)
	})
	return s.store.Save(context.Background(), items)
}

func cloneTravelGuide(guide productdomain.TravelGuide) productdomain.TravelGuide {
	cloned := guide.Normalized()
	cloned.Companions = append([]string(nil), guide.Companions...)
	cloned.MustVisit = append([]string(nil), guide.MustVisit...)
	cloned.Avoid = append([]string(nil), guide.Avoid...)
	cloned.AdditionalRequirements = append([]string(nil), guide.AdditionalRequirements...)
	cloned.KeepConditions = append([]string(nil), guide.KeepConditions...)
	cloned.ReplaceConditions = append([]string(nil), guide.ReplaceConditions...)
	cloned.Notes = append([]string(nil), guide.Notes...)
	if len(guide.POIs) > 0 {
		cloned.POIs = append([]productdomain.TravelPOI(nil), guide.POIs...)
	}
	if len(guide.MetroLines) > 0 {
		cloned.MetroLines = append([]productdomain.TravelMetroLine(nil), guide.MetroLines...)
	}
	if len(guide.DailyRoutes) > 0 {
		cloned.DailyRoutes = append([]productdomain.TravelDailyRoute(nil), guide.DailyRoutes...)
	}
	if len(guide.Foods) > 0 {
		cloned.Foods = append([]productdomain.TravelFood(nil), guide.Foods...)
	}
	if len(guide.MapLayers) > 0 {
		cloned.MapLayers = append([]productdomain.TravelMapLayer(nil), guide.MapLayers...)
	}
	return cloned
}

func normalizeTravelGuideID(raw string) string {
	return strings.ToLower(strings.TrimSpace(raw))
}

func travelGuideIDBase(city string) string {
	trimmed := strings.ToLower(strings.TrimSpace(city))
	if trimmed == "" {
		return ""
	}
	var builder strings.Builder
	lastDash := false
	for _, r := range trimmed {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			builder.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash {
			builder.WriteByte('-')
			lastDash = true
		}
	}
	base := strings.Trim(builder.String(), "-")
	if base == "" {
		return "travel-guide"
	}
	return base + "-guide"
}

func unionTravelLists(left []string, right []string) []string {
	combined := append(append([]string(nil), left...), right...)
	return productdomain.TravelGuide{Notes: combined}.Normalized().Notes
}

func ensureTravelMustVisit(city string, items []string, days int) []string {
	resolved := productdomain.TravelGuide{MustVisit: items}.Normalized().MustVisit
	defaults := []string{
		city + " Central Landmark",
		city + " Old Town",
		city + " Riverside Walk",
		city + " Local Museum",
	}
	for _, item := range defaults {
		if len(resolved) >= max(days+1, 3) {
			break
		}
		resolved = unionTravelLists(resolved, []string{item})
	}
	return resolved
}

func buildTravelPOIs(city string, mustVisit []string) []productdomain.TravelPOI {
	items := make([]productdomain.TravelPOI, 0, len(mustVisit))
	for index, name := range mustVisit {
		items = append(items, productdomain.TravelPOI{
			ID:          fmt.Sprintf("poi-%d", index+1),
			Name:        name,
			District:    city + " core area",
			Category:    travelPOICategory(index),
			Highlights:  []string{"efficient stop sequencing", "photo-friendly", "fits city guide narrative"},
			VisitLength: travelVisitLength(index),
		})
	}
	return items
}

func buildTravelMetroLines(city string, days int, mustVisit []string) []productdomain.TravelMetroLine {
	segments := []string{city + " Station Hub", city + " Downtown", city + " Riverside"}
	if len(mustVisit) > 0 {
		segments = append(segments, mustVisit[0])
	}
	items := []productdomain.TravelMetroLine{
		{ID: "metro-line-1", Name: "Metro Line 1", Segments: segments, Reasoning: "Connects the highest-frequency sightseeing corridor."},
		{ID: "metro-line-2", Name: "Metro Loop", Segments: []string{city + " Old Town", city + " Museum District", city + " Food Street"}, Reasoning: "Useful for midday transfers and food-focused detours."},
	}
	if days >= 4 {
		items = append(items, productdomain.TravelMetroLine{ID: "metro-airport", Name: "Airport Express", Segments: []string{"Airport", city + " Station Hub"}, Reasoning: "Reduces transfer fatigue on arrival or departure days."})
	}
	return items
}

func buildTravelFoods(city string, budget string, days int) []productdomain.TravelFood {
	items := []productdomain.TravelFood{
		{ID: "food-1", Name: city + " Signature Breakfast", Area: city + " old town", Signature: []string{"local breakfast set", "coffee or soy milk"}, When: "breakfast"},
		{ID: "food-2", Name: city + " Market Lunch", Area: city + " downtown", Signature: []string{"regional staple", "quick lunch option"}, When: "lunch"},
		{ID: "food-3", Name: city + " Evening Specialty", Area: city + " riverside", Signature: []string{"local classic", "dessert stop"}, When: "dinner"},
	}
	if strings.Contains(strings.ToLower(budget), "high") || strings.Contains(strings.ToLower(budget), "premium") {
		items = append(items, productdomain.TravelFood{ID: "food-4", Name: city + " Chef Table", Area: city + " central district", Signature: []string{"seasonal tasting", "reservation suggested"}, When: "special dinner"})
	}
	if days <= 2 && len(items) > 3 {
		return items[:3]
	}
	return items
}

func buildTravelDailyRoutes(days int, pois []productdomain.TravelPOI, metroLines []productdomain.TravelMetroLine, foods []productdomain.TravelFood) []productdomain.TravelDailyRoute {
	if days <= 0 {
		days = 3
	}
	items := make([]productdomain.TravelDailyRoute, 0, days)
	for day := 1; day <= days; day++ {
		stops := make([]string, 0, 3)
		for index := day - 1; index < len(pois) && len(stops) < 3; index += days {
			stops = append(stops, pois[index].Name)
		}
		if len(stops) == 0 && len(pois) > 0 {
			stops = append(stops, pois[0].Name)
		}
		transit := []string{"Walk + Metro Line 1"}
		if len(metroLines) > 1 && day%2 == 0 {
			transit = []string{"Walk + " + metroLines[1].Name}
		}
		diningPlan := []string{}
		if len(foods) > 0 {
			diningPlan = append(diningPlan, foods[(day-1)%len(foods)].Name)
		}
		items = append(items, productdomain.TravelDailyRoute{
			Day:        day,
			Theme:      fmt.Sprintf("Day %d city loop", day),
			Stops:      stops,
			Transit:    transit,
			DiningPlan: diningPlan,
		})
	}
	return items
}

func buildTravelNotes(city, travelStyle, budget string, companions, avoid, additional, keep, replace []string) []string {
	items := []string{
		"Focus on " + travelStyle + " pacing for " + city + ".",
		"Budget baseline: " + budget + ".",
	}
	if len(companions) > 0 {
		items = append(items, "Companions: "+strings.Join(companions, ", ")+".")
	}
	if len(avoid) > 0 {
		items = append(items, "Avoid: "+strings.Join(avoid, ", ")+".")
	}
	if len(additional) > 0 {
		items = append(items, "Additional requirements: "+strings.Join(additional, ", ")+".")
	}
	if len(keep) > 0 {
		items = append(items, "Keep conditions: "+strings.Join(keep, ", ")+".")
	}
	if len(replace) > 0 {
		items = append(items, "Replace prior assumptions with: "+strings.Join(replace, ", ")+".")
	}
	return productdomain.TravelGuide{Notes: items}.Normalized().Notes
}

func buildTravelMapLayers(city string, pois []productdomain.TravelPOI, routes []productdomain.TravelDailyRoute, metroLines []productdomain.TravelMetroLine, foods []productdomain.TravelFood) []productdomain.TravelMapLayer {
	pointRefs := make([]string, 0, len(pois)+len(foods))
	for _, poi := range pois {
		pointRefs = append(pointRefs, poi.ID)
	}
	for _, food := range foods {
		pointRefs = append(pointRefs, food.ID)
	}
	routeRefs := make([]string, 0, len(routes))
	for _, route := range routes {
		routeRefs = append(routeRefs, fmt.Sprintf("day-%d", route.Day))
	}
	metroRefs := make([]string, 0, len(metroLines))
	for _, line := range metroLines {
		metroRefs = append(metroRefs, line.ID)
	}
	return []productdomain.TravelMapLayer{
		{ID: "layer-pois", Type: "point", Label: city + " POIs", References: pointRefs, Description: "Highlight key scenic spots and dining stops."},
		{ID: "layer-routes", Type: "line", Label: city + " Daily Routes", References: routeRefs, Description: "Show suggested day-by-day visit ordering."},
		{ID: "layer-metro", Type: "line", Label: city + " Metro", References: metroRefs, Description: "Show recommended metro corridors for inter-stop transfers."},
	}
}

func renderTravelGuideContent(guide productdomain.TravelGuide) string {
	var builder strings.Builder
	builder.WriteString(guide.City)
	builder.WriteString(" ")
	builder.WriteString(fmt.Sprintf("%d-day", guide.Days))
	builder.WriteString(" travel guide\n\n")
	builder.WriteString("Style: ")
	builder.WriteString(guide.TravelStyle)
	builder.WriteString("\nBudget: ")
	builder.WriteString(guide.Budget)
	if len(guide.Companions) > 0 {
		builder.WriteString("\nCompanions: ")
		builder.WriteString(strings.Join(guide.Companions, ", "))
	}
	builder.WriteString("\n\nKey POIs:\n")
	for _, poi := range guide.POIs {
		builder.WriteString("- ")
		builder.WriteString(poi.Name)
		builder.WriteString(" (")
		builder.WriteString(poi.VisitLength)
		builder.WriteString(")\n")
	}
	builder.WriteString("\nMetro:\n")
	for _, line := range guide.MetroLines {
		builder.WriteString("- ")
		builder.WriteString(line.Name)
		builder.WriteString(": ")
		builder.WriteString(line.Reasoning)
		builder.WriteString("\n")
	}
	builder.WriteString("\nDaily Routes:\n")
	for _, route := range guide.DailyRoutes {
		builder.WriteString(fmt.Sprintf("- Day %d: %s | Stops: %s | Transit: %s\n", route.Day, route.Theme, strings.Join(route.Stops, " -> "), strings.Join(route.Transit, ", ")))
	}
	builder.WriteString("\nFood:\n")
	for _, food := range guide.Foods {
		builder.WriteString("- ")
		builder.WriteString(food.Name)
		builder.WriteString(" (")
		builder.WriteString(food.When)
		builder.WriteString(")\n")
	}
	if len(guide.Notes) > 0 {
		builder.WriteString("\nNotes:\n")
		for _, note := range guide.Notes {
			builder.WriteString("- ")
			builder.WriteString(note)
			builder.WriteString("\n")
		}
	}
	builder.WriteString("\nMap Layers:\n")
	for _, layer := range guide.MapLayers {
		builder.WriteString("- ")
		builder.WriteString(layer.Label)
		builder.WriteString(": ")
		builder.WriteString(layer.Description)
		builder.WriteString("\n")
	}
	return strings.TrimSpace(builder.String())
}

func travelPOICategory(index int) string {
	categories := []string{"landmark", "walk", "museum", "food-district"}
	return categories[index%len(categories)]
}

func travelVisitLength(index int) string {
	lengths := []string{"1-1.5h", "2h", "2-3h", "45m-1h"}
	return lengths[index%len(lengths)]
}

func max(left int, right int) int {
	if left > right {
		return left
	}
	return right
}
