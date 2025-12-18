package model

import "time"

type SearchIndex string

const (
	IndexWorkflows    SearchIndex = "workflows"
	IndexUsers        SearchIndex = "users"
	IndexExecutions   SearchIndex = "executions"
	IndexNodes        SearchIndex = "nodes"
	IndexNotifications SearchIndex = "notifications"
)

type SearchDocument struct {
	ID        string                 `json:"id"`
	Index     SearchIndex            `json:"index"`
	Type      string                 `json:"type"`
	Title     string                 `json:"title"`
	Content   string                 `json:"content"`
	Tags      []string               `json:"tags"`
	Metadata  map[string]interface{} `json:"metadata"`
	UserID    string                 `json:"user_id"`
	CreatedAt time.Time              `json:"created_at"`
	UpdatedAt time.Time              `json:"updated_at"`
	Score     float64                `json:"score,omitempty"`
}

func NewSearchDocument(id string, index SearchIndex, title, content string) *SearchDocument {
	now := time.Now()
	return &SearchDocument{
		ID:        id,
		Index:     index,
		Title:     title,
		Content:   content,
		Tags:      []string{},
		Metadata:  make(map[string]interface{}),
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func (d *SearchDocument) AddTag(tag string) {
	d.Tags = append(d.Tags, tag)
}

func (d *SearchDocument) SetMetadata(key string, value interface{}) {
	d.Metadata[key] = value
}

type SearchResult struct {
	Documents   []*SearchDocument `json:"documents"`
	TotalHits   int64             `json:"total_hits"`
	MaxScore    float64           `json:"max_score"`
	Took        int64             `json:"took_ms"`
	Facets      map[string][]Facet `json:"facets,omitempty"`
	Suggestions []string          `json:"suggestions,omitempty"`
}

type Facet struct {
	Value string `json:"value"`
	Count int64  `json:"count"`
}

type SearchQuery struct {
	Query       string              `json:"query"`
	Indexes     []SearchIndex       `json:"indexes"`
	Filters     map[string]string   `json:"filters"`
	Tags        []string            `json:"tags"`
	UserID      string              `json:"user_id"`
	From        int                 `json:"from"`
	Size        int                 `json:"size"`
	SortBy      string              `json:"sort_by"`
	SortOrder   string              `json:"sort_order"`
	Highlight   bool                `json:"highlight"`
	Facets      []string            `json:"facets"`
	Fuzzy       bool                `json:"fuzzy"`
}
