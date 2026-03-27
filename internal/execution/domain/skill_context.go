package domain

const (
	SkillContextProtocolVersion = "alter0.skill-context/v1"
	SkillContextMetadataKey     = "alter0.skill_context"

	SkillConflictTypeDuplicateName     = "duplicate_name"
	SkillConflictTypeDuplicateAbility  = "duplicate_ability"
	SkillConflictTypeParameterConflict = "parameter_conflict"
)

type SkillContext struct {
	Protocol           string            `json:"protocol"`
	Skills             []SkillSpec       `json:"skills"`
	ResolvedParameters map[string]string `json:"resolved_parameters,omitempty"`
	Conflicts          []SkillConflict   `json:"conflicts,omitempty"`
}

type SkillSpec struct {
	ID                string            `json:"id"`
	Name              string            `json:"name"`
	Description       string            `json:"description"`
	Guide             string            `json:"guide,omitempty"`
	Priority          int               `json:"priority"`
	ParameterTemplate map[string]string `json:"parameter_template,omitempty"`
	Constraints       []string          `json:"constraints,omitempty"`
	Abilities         []string          `json:"abilities,omitempty"`
}

type SkillConflict struct {
	Type           string `json:"type"`
	Key            string `json:"key"`
	WinnerSkillID  string `json:"winner_skill_id"`
	DroppedSkillID string `json:"dropped_skill_id"`
	Detail         string `json:"detail,omitempty"`
}
