package domain

type Skill struct {
	ID       string            `json:"id"`
	Name     string            `json:"name"`
	Type     CapabilityType    `json:"type"`
	Enabled  bool              `json:"enabled"`
	Scope    CapabilityScope   `json:"scope"`
	Version  string            `json:"version"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

func (s Skill) AsCapability() Capability {
	capabilityType := s.Type
	if capabilityType == "" {
		capabilityType = CapabilityTypeSkill
	}
	return Capability{
		ID:       s.ID,
		Name:     s.Name,
		Type:     capabilityType,
		Enabled:  s.Enabled,
		Scope:    s.Scope,
		Version:  s.Version,
		Metadata: s.Metadata,
	}
}

func (s Skill) Validate() error {
	return s.AsCapability().Validate()
}

func SkillFromCapability(capability Capability) Skill {
	return Skill{
		ID:       capability.ID,
		Name:     capability.Name,
		Type:     capability.Type,
		Enabled:  capability.Enabled,
		Scope:    capability.Scope,
		Version:  capability.Version,
		Metadata: capability.Metadata,
	}
}
