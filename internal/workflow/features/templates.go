// Package features provides workflow templates
package features

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/linkflow-ai/linkflow-ai/internal/workflow/domain/model"
)

// Template represents a workflow template
type Template struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Category    string                 `json:"category"`
	Tags        []string               `json:"tags"`
	Icon        string                 `json:"icon"`
	Nodes       []model.Node           `json:"nodes"`
	Connections []model.Connection     `json:"connections"`
	Settings    model.Settings         `json:"settings"`
	Variables   []TemplateVariable     `json:"variables"`
	Integrations []string              `json:"integrations"` // Required integrations
	Author      string                 `json:"author"`
	Version     string                 `json:"version"`
	IsPublic    bool                   `json:"isPublic"`
	UsageCount  int64                  `json:"usageCount"`
	Rating      float64                `json:"rating"`
	CreatedAt   time.Time              `json:"createdAt"`
	UpdatedAt   time.Time              `json:"updatedAt"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// TemplateVariable represents a variable in a template that needs to be configured
type TemplateVariable struct {
	Name         string      `json:"name"`
	Description  string      `json:"description"`
	Type         string      `json:"type"` // string, number, boolean, credential, integration
	Required     bool        `json:"required"`
	DefaultValue interface{} `json:"defaultValue"`
	Options      []string    `json:"options,omitempty"` // For select type
	Placeholder  string      `json:"placeholder"`
}

// TemplateCategory represents a template category
type TemplateCategory struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Icon        string `json:"icon"`
	Order       int    `json:"order"`
}

// TemplateRepository defines template persistence
type TemplateRepository interface {
	Create(ctx context.Context, template *Template) error
	FindByID(ctx context.Context, id string) (*Template, error)
	Update(ctx context.Context, template *Template) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, filter *TemplateFilter, limit, offset int) ([]*Template, error)
	Count(ctx context.Context, filter *TemplateFilter) (int64, error)
	IncrementUsage(ctx context.Context, id string) error
}

// TemplateFilter holds filter options for listing templates
type TemplateFilter struct {
	Category     string
	Tags         []string
	Integrations []string
	IsPublic     *bool
	Author       string
	Search       string
}

// TemplateService manages workflow templates
type TemplateService struct {
	repo       TemplateRepository
	categories map[string]*TemplateCategory
	mu         sync.RWMutex
}

// NewTemplateService creates a new template service
func NewTemplateService(repo TemplateRepository) *TemplateService {
	svc := &TemplateService{
		repo:       repo,
		categories: make(map[string]*TemplateCategory),
	}
	svc.initDefaultCategories()
	return svc
}

func (s *TemplateService) initDefaultCategories() {
	defaultCategories := []*TemplateCategory{
		{ID: "marketing", Name: "Marketing", Description: "Marketing automation workflows", Icon: "megaphone", Order: 1},
		{ID: "sales", Name: "Sales", Description: "Sales and CRM workflows", Icon: "chart-line", Order: 2},
		{ID: "devops", Name: "DevOps", Description: "CI/CD and infrastructure workflows", Icon: "server", Order: 3},
		{ID: "productivity", Name: "Productivity", Description: "Personal productivity workflows", Icon: "clock", Order: 4},
		{ID: "data", Name: "Data & Analytics", Description: "Data processing workflows", Icon: "database", Order: 5},
		{ID: "communication", Name: "Communication", Description: "Messaging and notification workflows", Icon: "message", Order: 6},
		{ID: "ecommerce", Name: "E-commerce", Description: "Online store workflows", Icon: "shopping-cart", Order: 7},
		{ID: "hr", Name: "HR & Recruiting", Description: "Human resources workflows", Icon: "users", Order: 8},
		{ID: "finance", Name: "Finance", Description: "Financial and accounting workflows", Icon: "dollar-sign", Order: 9},
		{ID: "other", Name: "Other", Description: "Miscellaneous workflows", Icon: "folder", Order: 100},
	}

	for _, cat := range defaultCategories {
		s.categories[cat.ID] = cat
	}
}

// CreateTemplate creates a new template
func (s *TemplateService) CreateTemplate(ctx context.Context, template *Template) error {
	if template.ID == "" {
		template.ID = uuid.New().String()
	}
	template.CreatedAt = time.Now()
	template.UpdatedAt = time.Now()
	template.Version = "1.0.0"
	template.UsageCount = 0

	if template.Category == "" {
		template.Category = "other"
	}

	return s.repo.Create(ctx, template)
}

// CreateFromWorkflow creates a template from an existing workflow
func (s *TemplateService) CreateFromWorkflow(ctx context.Context, workflow *model.Workflow, name, description, category string, isPublic bool) (*Template, error) {
	template := &Template{
		ID:          uuid.New().String(),
		Name:        name,
		Description: description,
		Category:    category,
		Tags:        []string{},
		Nodes:       workflow.Nodes(),
		Connections: workflow.Connections(),
		Settings:    workflow.Settings(),
		Variables:   s.extractVariables(workflow),
		Integrations: s.extractIntegrations(workflow),
		Author:      workflow.UserID(),
		IsPublic:    isPublic,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Version:     "1.0.0",
		Metadata:    make(map[string]interface{}),
	}

	if err := s.repo.Create(ctx, template); err != nil {
		return nil, err
	}

	return template, nil
}

// extractVariables extracts variables from workflow nodes
func (s *TemplateService) extractVariables(workflow *model.Workflow) []TemplateVariable {
	var variables []TemplateVariable
	seen := make(map[string]bool)

	for _, node := range workflow.Nodes() {
		for key, value := range node.Config {
			// Look for placeholder patterns like {{variable_name}}
			if strVal, ok := value.(string); ok {
				// Simple pattern matching for variables
				if len(strVal) > 4 && strVal[:2] == "{{" && strVal[len(strVal)-2:] == "}}" {
					varName := strVal[2 : len(strVal)-2]
					if !seen[varName] {
						seen[varName] = true
						variables = append(variables, TemplateVariable{
							Name:        varName,
							Description: fmt.Sprintf("Variable for %s in %s", key, node.Name),
							Type:        "string",
							Required:    true,
						})
					}
				}
			}
		}
	}

	return variables
}

// extractIntegrations extracts required integrations from workflow
func (s *TemplateService) extractIntegrations(workflow *model.Workflow) []string {
	integrations := make(map[string]bool)

	for _, node := range workflow.Nodes() {
		// Map node types to integration names
		switch {
		case contains(node.Name, "Slack"):
			integrations["slack"] = true
		case contains(node.Name, "GitHub"):
			integrations["github"] = true
		case contains(node.Name, "Google"):
			integrations["google"] = true
		case contains(node.Name, "Email"), contains(node.Name, "SMTP"):
			integrations["email"] = true
		case contains(node.Name, "Notion"):
			integrations["notion"] = true
		case contains(node.Name, "Airtable"):
			integrations["airtable"] = true
		}
	}

	result := make([]string, 0, len(integrations))
	for integration := range integrations {
		result = append(result, integration)
	}
	return result
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// GetTemplate retrieves a template by ID
func (s *TemplateService) GetTemplate(ctx context.Context, id string) (*Template, error) {
	return s.repo.FindByID(ctx, id)
}

// ListTemplates lists templates with filtering
func (s *TemplateService) ListTemplates(ctx context.Context, filter *TemplateFilter, limit, offset int) ([]*Template, error) {
	return s.repo.List(ctx, filter, limit, offset)
}

// ListCategories returns all template categories
func (s *TemplateService) ListCategories() []*TemplateCategory {
	s.mu.RLock()
	defer s.mu.RUnlock()

	categories := make([]*TemplateCategory, 0, len(s.categories))
	for _, cat := range s.categories {
		categories = append(categories, cat)
	}
	return categories
}

// InstantiateTemplate creates a workflow from a template
func (s *TemplateService) InstantiateTemplate(ctx context.Context, templateID string, userID string, values map[string]interface{}) (*model.Workflow, error) {
	template, err := s.repo.FindByID(ctx, templateID)
	if err != nil {
		return nil, fmt.Errorf("template not found: %w", err)
	}

	// Validate required variables
	for _, v := range template.Variables {
		if v.Required {
			if _, ok := values[v.Name]; !ok {
				if v.DefaultValue == nil {
					return nil, fmt.Errorf("required variable %s not provided", v.Name)
				}
				values[v.Name] = v.DefaultValue
			}
		}
	}

	// Create workflow from template
	workflow, err := model.NewWorkflow(userID, template.Name, template.Description)
	if err != nil {
		return nil, err
	}

	// Apply nodes with variable substitution
	for _, node := range template.Nodes {
		processedNode := s.substituteVariables(node, values)
		if err := workflow.AddNode(processedNode); err != nil {
			return nil, fmt.Errorf("failed to add node: %w", err)
		}
	}

	// Apply connections
	for _, conn := range template.Connections {
		if err := workflow.AddConnection(conn); err != nil {
			return nil, fmt.Errorf("failed to add connection: %w", err)
		}
	}

	// Apply settings
	if err := workflow.UpdateSettings(template.Settings); err != nil {
		return nil, fmt.Errorf("failed to update settings: %w", err)
	}

	// Increment usage count
	s.repo.IncrementUsage(ctx, templateID)

	return workflow, nil
}

func (s *TemplateService) substituteVariables(node model.Node, values map[string]interface{}) model.Node {
	// Deep copy the node
	configJSON, _ := json.Marshal(node.Config)
	configStr := string(configJSON)

	// Replace variables
	for name, value := range values {
		placeholder := "{{" + name + "}}"
		var replacement string
		switch v := value.(type) {
		case string:
			replacement = v
		default:
			b, _ := json.Marshal(v)
			replacement = string(b)
		}
		configStr = replaceAll(configStr, placeholder, replacement)
	}

	var newConfig map[string]interface{}
	json.Unmarshal([]byte(configStr), &newConfig)

	return model.Node{
		ID:          uuid.New().String(), // Generate new ID
		Type:        node.Type,
		Name:        node.Name,
		Description: node.Description,
		Config:      newConfig,
		Position:    node.Position,
	}
}

func replaceAll(s, old, new string) string {
	result := s
	for {
		idx := -1
		for i := 0; i <= len(result)-len(old); i++ {
			if result[i:i+len(old)] == old {
				idx = i
				break
			}
		}
		if idx == -1 {
			break
		}
		result = result[:idx] + new + result[idx+len(old):]
	}
	return result
}

// UpdateTemplate updates a template
func (s *TemplateService) UpdateTemplate(ctx context.Context, template *Template) error {
	template.UpdatedAt = time.Now()
	return s.repo.Update(ctx, template)
}

// DeleteTemplate deletes a template
func (s *TemplateService) DeleteTemplate(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}

// InMemoryTemplateRepository implements TemplateRepository in memory
type InMemoryTemplateRepository struct {
	templates map[string]*Template
	mu        sync.RWMutex
}

// NewInMemoryTemplateRepository creates a new in-memory template repository
func NewInMemoryTemplateRepository() *InMemoryTemplateRepository {
	return &InMemoryTemplateRepository{
		templates: make(map[string]*Template),
	}
}

func (r *InMemoryTemplateRepository) Create(ctx context.Context, template *Template) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.templates[template.ID] = template
	return nil
}

func (r *InMemoryTemplateRepository) FindByID(ctx context.Context, id string) (*Template, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	template, ok := r.templates[id]
	if !ok {
		return nil, fmt.Errorf("template not found")
	}
	return template, nil
}

func (r *InMemoryTemplateRepository) Update(ctx context.Context, template *Template) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.templates[template.ID] = template
	return nil
}

func (r *InMemoryTemplateRepository) Delete(ctx context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.templates, id)
	return nil
}

func (r *InMemoryTemplateRepository) List(ctx context.Context, filter *TemplateFilter, limit, offset int) ([]*Template, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []*Template
	for _, t := range r.templates {
		if r.matchesFilter(t, filter) {
			result = append(result, t)
		}
	}

	// Apply pagination
	if offset >= len(result) {
		return []*Template{}, nil
	}
	end := offset + limit
	if end > len(result) {
		end = len(result)
	}
	return result[offset:end], nil
}

func (r *InMemoryTemplateRepository) matchesFilter(t *Template, filter *TemplateFilter) bool {
	if filter == nil {
		return true
	}
	if filter.Category != "" && t.Category != filter.Category {
		return false
	}
	if filter.IsPublic != nil && t.IsPublic != *filter.IsPublic {
		return false
	}
	if filter.Author != "" && t.Author != filter.Author {
		return false
	}
	return true
}

func (r *InMemoryTemplateRepository) Count(ctx context.Context, filter *TemplateFilter) (int64, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var count int64
	for _, t := range r.templates {
		if r.matchesFilter(t, filter) {
			count++
		}
	}
	return count, nil
}

func (r *InMemoryTemplateRepository) IncrementUsage(ctx context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if t, ok := r.templates[id]; ok {
		t.UsageCount++
	}
	return nil
}
