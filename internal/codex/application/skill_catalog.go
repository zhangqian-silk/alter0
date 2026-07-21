package application

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type NativeSkillLocation string

const (
	NativeSkillLocationAlter0     NativeSkillLocation = "alter0"
	NativeSkillLocationUserAgents NativeSkillLocation = "user_agents"
	NativeSkillLocationCodexHome  NativeSkillLocation = "codex_home"
	NativeSkillLocationRepo       NativeSkillLocation = "repo"
	NativeSkillLocationAdmin      NativeSkillLocation = "admin"
	NativeSkillLocationSystem     NativeSkillLocation = "system"
	NativeSkillLocationOther      NativeSkillLocation = "other"
)

type NativeSkillDependency struct {
	Type        string `json:"type"`
	Value       string `json:"value"`
	Command     string `json:"command,omitempty"`
	Description string `json:"description,omitempty"`
	Transport   string `json:"transport,omitempty"`
	URL         string `json:"url,omitempty"`
}

type NativeSkillCatalogItem struct {
	Name             string                  `json:"name"`
	Description      string                  `json:"description"`
	Enabled          bool                    `json:"enabled"`
	Scope            string                  `json:"scope"`
	Location         NativeSkillLocation     `json:"location"`
	DisplayName      string                  `json:"display_name,omitempty"`
	ShortDescription string                  `json:"short_description,omitempty"`
	Dependencies     []NativeSkillDependency `json:"dependencies"`
	Duplicate        bool                    `json:"duplicate"`
	DuplicateGroup   string                  `json:"duplicate_group,omitempty"`
	ManagedSkillID   string                  `json:"-"`
	path             string
}

type NativeSkillCatalogError struct {
	Code     string              `json:"code"`
	Message  string              `json:"message"`
	Location NativeSkillLocation `json:"location"`
}

type NativeSkillCatalog struct {
	Items  []NativeSkillCatalogItem  `json:"items"`
	Errors []NativeSkillCatalogError `json:"errors"`
}

type appServerSkillsListResponse struct {
	Data []appServerSkillsListEntry `json:"data"`
}

type appServerSkillsListEntry struct {
	CWD    string                    `json:"cwd"`
	Skills []appServerSkillMetadata  `json:"skills"`
	Errors []appServerSkillErrorInfo `json:"errors"`
}

type appServerSkillMetadata struct {
	Name             string                      `json:"name"`
	Description      string                      `json:"description"`
	Enabled          bool                        `json:"enabled"`
	Path             string                      `json:"path"`
	Scope            string                      `json:"scope"`
	ShortDescription *string                     `json:"shortDescription"`
	Interface        *appServerSkillInterface    `json:"interface"`
	Dependencies     *appServerSkillDependencies `json:"dependencies"`
}

type appServerSkillInterface struct {
	DisplayName      *string `json:"displayName"`
	ShortDescription *string `json:"shortDescription"`
}

type appServerSkillDependencies struct {
	Tools []appServerSkillToolDependency `json:"tools"`
}

type appServerSkillToolDependency struct {
	Type        string  `json:"type"`
	Value       string  `json:"value"`
	Command     *string `json:"command"`
	Description *string `json:"description"`
	Transport   *string `json:"transport"`
	URL         *string `json:"url"`
}

type appServerSkillErrorInfo struct {
	Message string `json:"message"`
	Path    string `json:"path"`
}

func (s *Service) ListSkills(ctx context.Context, cwd string) (*NativeSkillCatalog, error) {
	activeHome, err := s.resolveActiveHome()
	if err != nil {
		return nil, err
	}
	cwd = filepath.Clean(strings.TrimSpace(cwd))
	if cwd == "." || cwd == "" {
		cwd, err = filepath.Abs(".")
		if err != nil {
			return nil, err
		}
	}
	userHome, _ := os.UserHomeDir()
	callContext := ctx
	if callContext == nil {
		callContext = context.Background()
	}
	if _, hasDeadline := callContext.Deadline(); !hasDeadline {
		var cancel context.CancelFunc
		callContext, cancel = context.WithTimeout(callContext, 10*time.Second)
		defer cancel()
	}
	var payload appServerSkillsListResponse
	if err := s.appServerCall(callContext, activeHome, "skills/list", map[string]any{
		"cwds":        []string{cwd},
		"forceReload": true,
	}, &payload); err != nil {
		return nil, err
	}

	catalog := &NativeSkillCatalog{
		Items:  []NativeSkillCatalogItem{},
		Errors: []NativeSkillCatalogError{},
	}
	seenPaths := map[string]struct{}{}
	for _, entry := range payload.Data {
		entryCWD := strings.TrimSpace(entry.CWD)
		if entryCWD == "" {
			entryCWD = cwd
		}
		for _, skill := range entry.Skills {
			path := filepath.Clean(strings.TrimSpace(skill.Path))
			if path == "." || path == "" {
				continue
			}
			if _, exists := seenPaths[path]; exists {
				continue
			}
			seenPaths[path] = struct{}{}
			location, managedSkillID := classifyNativeSkillLocation(path, skill.Scope, activeHome, userHome, entryCWD)
			catalog.Items = append(catalog.Items, NativeSkillCatalogItem{
				Name:             strings.TrimSpace(skill.Name),
				Description:      strings.TrimSpace(skill.Description),
				Enabled:          skill.Enabled,
				Scope:            strings.TrimSpace(skill.Scope),
				Location:         location,
				DisplayName:      skillDisplayName(skill),
				ShortDescription: skillShortDescription(skill),
				Dependencies:     skillDependencies(skill.Dependencies),
				ManagedSkillID:   managedSkillID,
				path:             path,
			})
		}
		for _, item := range entry.Errors {
			location, _ := classifyNativeSkillLocation(item.Path, "", activeHome, userHome, entryCWD)
			catalog.Errors = append(catalog.Errors, NativeSkillCatalogError{
				Code:     "parse_error",
				Message:  "Codex could not load a Skill from this location.",
				Location: location,
			})
		}
	}
	markDuplicateNativeSkills(catalog.Items)
	sort.SliceStable(catalog.Items, func(i, j int) bool {
		left := strings.ToLower(catalog.Items[i].Name) + "\x00" + string(catalog.Items[i].Location)
		right := strings.ToLower(catalog.Items[j].Name) + "\x00" + string(catalog.Items[j].Location)
		return left < right
	})
	return catalog, nil
}

func classifyNativeSkillLocation(path string, scope string, activeHome string, userHome string, cwd string) (NativeSkillLocation, string) {
	skillDir := nativeSkillDirectory(path)
	if marker, ok := readNativeSkillMarker(skillDir); ok {
		return NativeSkillLocationAlter0, normalizeNativeSkillID(marker.SkillID)
	}
	normalizedScope := strings.ToLower(strings.TrimSpace(scope))
	if normalizedScope == "repo" {
		return NativeSkillLocationRepo, ""
	}
	if normalizedScope == "admin" {
		return NativeSkillLocationAdmin, ""
	}
	if normalizedScope == "system" {
		return NativeSkillLocationSystem, ""
	}
	if pathWithinNativeSkillRoot(path, filepath.Join(userHome, ".agents", "skills")) {
		return NativeSkillLocationUserAgents, ""
	}
	if pathWithinNativeSkillRoot(path, filepath.Join(activeHome, "skills")) {
		return NativeSkillLocationCodexHome, ""
	}
	if pathWithinNativeSkillRoot(path, filepath.Join(cwd, ".agents", "skills")) {
		return NativeSkillLocationRepo, ""
	}
	return NativeSkillLocationOther, ""
}

func nativeSkillDirectory(path string) string {
	path = filepath.Clean(strings.TrimSpace(path))
	if strings.EqualFold(filepath.Base(path), "SKILL.md") {
		return filepath.Dir(path)
	}
	return path
}

func pathWithinNativeSkillRoot(path string, root string) bool {
	path = filepath.Clean(strings.TrimSpace(path))
	root = filepath.Clean(strings.TrimSpace(root))
	if path == "." || root == "." || path == "" || root == "" {
		return false
	}
	relative, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}
	return relative != ".." && relative != "." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))
}

func skillDisplayName(skill appServerSkillMetadata) string {
	if skill.Interface == nil || skill.Interface.DisplayName == nil {
		return ""
	}
	return strings.TrimSpace(*skill.Interface.DisplayName)
}

func skillShortDescription(skill appServerSkillMetadata) string {
	if skill.Interface != nil && skill.Interface.ShortDescription != nil {
		if value := strings.TrimSpace(*skill.Interface.ShortDescription); value != "" {
			return value
		}
	}
	if skill.ShortDescription == nil {
		return ""
	}
	return strings.TrimSpace(*skill.ShortDescription)
}

func skillDependencies(dependencies *appServerSkillDependencies) []NativeSkillDependency {
	result := []NativeSkillDependency{}
	if dependencies == nil {
		return result
	}
	for _, dependency := range dependencies.Tools {
		result = append(result, NativeSkillDependency{
			Type:        strings.TrimSpace(dependency.Type),
			Value:       strings.TrimSpace(dependency.Value),
			Command:     normalizeTextPointer(dependency.Command),
			Description: normalizeTextPointer(dependency.Description),
			Transport:   normalizeTextPointer(dependency.Transport),
			URL:         normalizeTextPointer(dependency.URL),
		})
	}
	return result
}

func markDuplicateNativeSkills(items []NativeSkillCatalogItem) {
	pathsByName := map[string]map[string]struct{}{}
	for _, item := range items {
		name := strings.ToLower(strings.TrimSpace(item.Name))
		if name == "" {
			continue
		}
		if pathsByName[name] == nil {
			pathsByName[name] = map[string]struct{}{}
		}
		pathsByName[name][item.path] = struct{}{}
	}
	for index := range items {
		name := strings.ToLower(strings.TrimSpace(items[index].Name))
		if len(pathsByName[name]) < 2 {
			continue
		}
		items[index].Duplicate = true
		items[index].DuplicateGroup = name
	}
}
