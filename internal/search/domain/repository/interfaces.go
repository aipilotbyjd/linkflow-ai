package repository

import (
	"context"

	"github.com/linkflow-ai/linkflow-ai/internal/search/domain/model"
)

type SearchRepository interface {
	Index(ctx context.Context, doc *model.SearchDocument) error
	BulkIndex(ctx context.Context, docs []*model.SearchDocument) error
	Update(ctx context.Context, doc *model.SearchDocument) error
	Delete(ctx context.Context, index model.SearchIndex, id string) error
	Search(ctx context.Context, query model.SearchQuery) (*model.SearchResult, error)
	Suggest(ctx context.Context, prefix string, index model.SearchIndex) ([]string, error)
	GetStats(ctx context.Context) (map[string]interface{}, error)
}
