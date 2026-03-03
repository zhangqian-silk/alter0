package domain

const (
	SkillContextProtocolVersion = "alter0.skill-context/v1"
	SkillContextMetadataKey     = "alter0.skill_context"
)

type SkillContext struct {
	Protocol string      `json:"protocol"`
	Skills   []SkillSpec `json:"skills"`
}

type SkillSpec struct {
	ID                string            `json:"id"`
	Name              string            `json:"name"`
	Description       string            `json:"description"`
	Priority          int               `json:"priority"`
	ParameterTemplate map[string]string `json:"parameter_template,omitempty"`
	Constraints       []string          `json:"constraints,omitempty"`
}
