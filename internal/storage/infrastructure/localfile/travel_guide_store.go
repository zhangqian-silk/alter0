package localfile

import (
	"context"
	"path/filepath"
	"sync"

	productapp "alter0/internal/product/application"
	productdomain "alter0/internal/product/domain"
)

type travelGuideState struct {
	Items []productdomain.TravelGuide `json:"items"`
}

type TravelGuideStore struct {
	path   string
	format Format
	mu     sync.Mutex
}

func NewTravelGuideStore(baseDir string, format Format) *TravelGuideStore {
	return &TravelGuideStore{
		path:   filepath.Join(baseDir, "travel_guides."+extension(format)),
		format: format,
	}
}

var _ productapp.TravelGuideStore = (*TravelGuideStore)(nil)

func (s *TravelGuideStore) Load(_ context.Context) ([]productdomain.TravelGuide, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	raw, ok, err := readIfExists(s.path)
	if err != nil {
		return nil, err
	}
	if !ok {
		return []productdomain.TravelGuide{}, nil
	}

	state := travelGuideState{}
	if err := unmarshalPayload(s.format, raw, &state); err != nil {
		return nil, err
	}
	if len(state.Items) == 0 {
		return []productdomain.TravelGuide{}, nil
	}
	items := make([]productdomain.TravelGuide, 0, len(state.Items))
	items = append(items, state.Items...)
	return items, nil
}

func (s *TravelGuideStore) Save(_ context.Context, guides []productdomain.TravelGuide) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	items := make([]productdomain.TravelGuide, 0, len(guides))
	items = append(items, guides...)
	raw, err := marshalPayload(s.format, "alter0 travel guides", travelGuideState{Items: items})
	if err != nil {
		return err
	}
	return writeFile(s.path, raw)
}
