package elasticsearch

import (
	"context"

	"github.com/linkflow-ai/linkflow-ai/internal/search/domain/model"
	"github.com/linkflow-ai/linkflow-ai/internal/search/domain/repository"
)

// MockSearchRepository is a mock implementation for now
type MockSearchRepository struct{}

func NewSearchRepository() repository.SearchRepository {
	return &MockSearchRepository{}
}

func (r *MockSearchRepository) Index(ctx context.Context, doc *model.SearchDocument) error {
	return nil
}

func (r *MockSearchRepository) BulkIndex(ctx context.Context, docs []*model.SearchDocument) error {
	return nil
}

func (r *MockSearchRepository) Update(ctx context.Context, doc *model.SearchDocument) error {
	return nil
}

func (r *MockSearchRepository) Delete(ctx context.Context, index model.SearchIndex, id string) error {
	return nil
}

func (r *MockSearchRepository) Search(ctx context.Context, query model.SearchQuery) (*model.SearchResult, error) {
	// Mock search results
	return &model.SearchResult{
		Documents: []*model.SearchDocument{},
		TotalHits: 0,
		MaxScore:  0,
		Took:      1,
	}, nil
}

func (r *MockSearchRepository) Suggest(ctx context.Context, prefix string, index model.SearchIndex) ([]string, error) {
	return []string{}, nil
}

func (r *MockSearchRepository) GetStats(ctx context.Context) (map[string]interface{}, error) {
	return map[string]interface{}{
		"total_documents": 0,
		"indexes": map[string]int{
			"workflows":     0,
			"users":         0,
			"executions":    0,
			"nodes":         0,
			"notifications": 0,
		},
	}, nil
}
