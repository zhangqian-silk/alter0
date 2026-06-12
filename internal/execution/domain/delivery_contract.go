package domain

type SessionProfileField struct {
	Key         string `json:"key"`
	Label       string `json:"label"`
	Description string `json:"description,omitempty"`
	ReadOnly    bool   `json:"readonly,omitempty"`
}

type Deliverable struct {
	ID                  string `json:"id"`
	Label               string `json:"label"`
	Description         string `json:"description,omitempty"`
	Format              string `json:"format,omitempty"`
	Required            bool   `json:"required,omitempty"`
	SessionAttributeKey string `json:"session_attribute_key,omitempty"`
}

const (
	CompletionCheckTypeSessionFileExists         = "session_file_exists"
	CompletionCheckTypeWorkspaceServicePublished = "workspace_service_published"
	CompletionCheckTypeSessionAttributeNonEmpty  = "session_attribute_nonempty"
)

type CompletionCheck struct {
	ID                    string `json:"id"`
	Label                 string `json:"label"`
	Description           string `json:"description,omitempty"`
	Type                  string `json:"type"`
	Required              bool   `json:"required,omitempty"`
	SessionPath           string `json:"session_path,omitempty"`
	ServiceID             string `json:"service_id,omitempty"`
	RequireServiceURL     bool   `json:"require_service_url,omitempty"`
	RequirePublicReadOnly bool   `json:"require_public_read_only,omitempty"`
	SessionAttributeKey   string `json:"session_attribute_key,omitempty"`
	FailureMessage        string `json:"failure_message,omitempty"`
	RepairInstruction     string `json:"repair_instruction,omitempty"`
}
