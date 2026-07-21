package web

import (
	"net/http"
	"sort"
	"strings"

	codexapp "alter0/internal/codex/application"
	controldomain "alter0/internal/control/domain"
)

const alter0SkillOwnerMetadataKey = "skill.owner"

type projectSkillCatalogItem struct {
	ID                string `json:"id"`
	Name              string `json:"name"`
	Description       string `json:"description"`
	ConfiguredEnabled bool   `json:"configured_enabled"`
	CodexVisible      bool   `json:"codex_visible"`
	SyncStatus        string `json:"sync_status"`
	Duplicate         bool   `json:"duplicate"`
	DuplicateGroup    string `json:"duplicate_group,omitempty"`
}

type skillCatalogResponse struct {
	ProjectSkills []projectSkillCatalogItem          `json:"project_skills"`
	CodexSkills   []codexapp.NativeSkillCatalogItem  `json:"codex_skills"`
	Errors        []codexapp.NativeSkillCatalogError `json:"errors"`
}

func (s *Server) skillCatalogHandler(w http.ResponseWriter, r *http.Request) {
	if s.control == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "control service unavailable"})
		return
	}
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	response := skillCatalogResponse{
		ProjectSkills: []projectSkillCatalogItem{},
		CodexSkills:   []codexapp.NativeSkillCatalogItem{},
		Errors:        []codexapp.NativeSkillCatalogError{},
	}
	catalogAvailable := false
	catalog := &codexapp.NativeSkillCatalog{
		Items:  []codexapp.NativeSkillCatalogItem{},
		Errors: []codexapp.NativeSkillCatalogError{},
	}
	if s.codexAccounts == nil {
		response.Errors = append(response.Errors, codexapp.NativeSkillCatalogError{
			Code:     "catalog_unavailable",
			Message:  "Codex Skill catalog is unavailable.",
			Location: codexapp.NativeSkillLocationOther,
		})
	} else if loaded, err := s.codexAccounts.ListSkills(r.Context(), s.workspaceRoot); err != nil {
		response.Errors = append(response.Errors, codexapp.NativeSkillCatalogError{
			Code:     "catalog_unavailable",
			Message:  "Codex Skill catalog is unavailable.",
			Location: codexapp.NativeSkillLocationOther,
		})
	} else if loaded != nil {
		catalog = loaded
		catalogAvailable = true
		response.Errors = append(response.Errors, loaded.Errors...)
	}

	projectCapabilities := alter0SkillCapabilities(s.control.ListCapabilitiesByType(controldomain.CapabilityTypeSkill))
	projectIDs := make(map[string]struct{}, len(projectCapabilities))
	managedItems := make(map[string]codexapp.NativeSkillCatalogItem, len(projectCapabilities))
	for _, capability := range projectCapabilities {
		projectIDs[capability.ID] = struct{}{}
	}
	for _, item := range catalog.Items {
		if _, managed := projectIDs[item.ManagedSkillID]; managed {
			managedItems[item.ManagedSkillID] = item
			continue
		}
		response.CodexSkills = append(response.CodexSkills, item)
	}
	for _, capability := range projectCapabilities {
		managedItem, found := managedItems[capability.ID]
		response.ProjectSkills = append(response.ProjectSkills, projectSkillCatalogItem{
			ID:                capability.ID,
			Name:              capability.Name,
			Description:       strings.TrimSpace(capability.Metadata["skill.description"]),
			ConfiguredEnabled: capability.Enabled,
			CodexVisible:      found && managedItem.Enabled,
			SyncStatus:        projectSkillSyncStatus(capability.Enabled, catalogAvailable, found, managedItem.Enabled),
			Duplicate:         found && managedItem.Duplicate,
			DuplicateGroup:    managedItem.DuplicateGroup,
		})
	}

	writeJSON(w, http.StatusOK, response)
}

func alter0SkillCapabilities(capabilities []controldomain.Capability) []controldomain.Capability {
	result := make([]controldomain.Capability, 0, len(capabilities))
	for _, capability := range capabilities {
		if !strings.EqualFold(strings.TrimSpace(capability.Metadata[alter0SkillOwnerMetadataKey]), "alter0") {
			continue
		}
		result = append(result, capability)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result
}

func projectSkillSyncStatus(configuredEnabled bool, catalogAvailable bool, found bool, codexEnabled bool) string {
	if !catalogAvailable {
		return "unknown"
	}
	if !configuredEnabled {
		if found {
			return "stale"
		}
		return "disabled"
	}
	if !found {
		return "missing"
	}
	if !codexEnabled {
		return "codex_disabled"
	}
	return "ready"
}
