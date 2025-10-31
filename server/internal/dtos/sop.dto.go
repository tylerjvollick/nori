package dtos

// CreateSOPStepDTO represents the step data when creating an SOP
type CreateSOPStepDTO struct {
	StepNumber           int     `json:"stepNumber" binding:"required"`
	Title                string  `json:"title" binding:"required"`
	Instructions         *string `json:"instructions,omitempty"`
	EstimatedTimeMinutes *int    `json:"estimatedTimeMinutes,omitempty"`
	ImageURL             *string `json:"imageUrl,omitempty"`
	VideoURL             *string `json:"videoUrl,omitempty"`
	RequiresApproval     bool    `json:"requiresApproval"`
}

// CreateSOPDTO represents the request to create a new SOP template with its first version
type CreateSOPDTO struct {
	Name          string             `json:"name" binding:"required"`
	Description   *string            `json:"description,omitempty"`
	ChangeSummary *string            `json:"changeSummary,omitempty"`
	Steps         []CreateSOPStepDTO `json:"steps" binding:"required,min=1"`
}

// UpdateSOPDTO represents the request to update an SOP (creates a new version)
type UpdateSOPDTO struct {
	Name          *string            `json:"name,omitempty"`
	Description   *string            `json:"description,omitempty"`
	ChangeSummary string             `json:"changeSummary" binding:"required"`
	Steps         []CreateSOPStepDTO `json:"steps" binding:"required,min=1"`
}

// SOPStepResponseDTO represents a step in the response
type SOPStepResponseDTO struct {
	ID                   int     `json:"id"`
	StepNumber           int     `json:"stepNumber"`
	Title                string  `json:"title"`
	Instructions         *string `json:"instructions,omitempty"`
	EstimatedTimeMinutes *int    `json:"estimatedTimeMinutes,omitempty"`
	ImageURL             *string `json:"imageUrl,omitempty"`
	VideoURL             *string `json:"videoUrl,omitempty"`
	RequiresApproval     bool    `json:"requiresApproval"`
}

// SOPVersionResponseDTO represents a version in the response
type SOPVersionResponseDTO struct {
	ID            int                  `json:"id"`
	VersionNumber int                  `json:"versionNumber"`
	Description   *string              `json:"description,omitempty"`
	ChangeSummary *string              `json:"changeSummary,omitempty"`
	CreatedAt     string               `json:"createdAt"`
	IsActive      bool                 `json:"isActive"`
	Steps         []SOPStepResponseDTO `json:"steps,omitempty"`
}

// SOPTemplateResponseDTO represents an SOP template in the response
type SOPTemplateResponseDTO struct {
	ID             int                    `json:"id"`
	Name           string                 `json:"name"`
	CreatedAt      string                 `json:"createdAt"`
	UpdatedAt      string                 `json:"updatedAt"`
	CurrentVersion *SOPVersionResponseDTO `json:"currentVersion,omitempty"`
}

// SOPTemplateDetailResponseDTO represents a detailed SOP template with all versions
type SOPTemplateDetailResponseDTO struct {
	ID             int                     `json:"id"`
	Name           string                  `json:"name"`
	CreatedAt      string                  `json:"createdAt"`
	UpdatedAt      string                  `json:"updatedAt"`
	CurrentVersion *SOPVersionResponseDTO  `json:"currentVersion,omitempty"`
	Versions       []SOPVersionResponseDTO `json:"versions,omitempty"`
}
