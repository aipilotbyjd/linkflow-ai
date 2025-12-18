package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/linkflow-ai/linkflow-ai/internal/platform/logger"
	"github.com/linkflow-ai/linkflow-ai/internal/search/domain/model"
	"github.com/linkflow-ai/linkflow-ai/internal/search/domain/repository"
)

type SearchService struct {
	repository repository.SearchRepository
	logger     logger.Logger
}

func NewSearchService(
	repository repository.SearchRepository,
	logger logger.Logger,
) *SearchService {
	return &SearchService{
		repository: repository,
		logger:     logger,
	}
}

func (s *SearchService) Index(ctx context.Context, doc *model.SearchDocument) error {
	if err := s.repository.Index(ctx, doc); err != nil {
		return fmt.Errorf("failed to index document: %w", err)
	}

	s.logger.Debug("Document indexed",
		"id", doc.ID,
		"index", doc.Index,
		"title", doc.Title,
	)

	return nil
}

func (s *SearchService) BulkIndex(ctx context.Context, docs []*model.SearchDocument) error {
	if err := s.repository.BulkIndex(ctx, docs); err != nil {
		return fmt.Errorf("failed to bulk index documents: %w", err)
	}

	s.logger.Info("Documents bulk indexed", "count", len(docs))
	return nil
}

func (s *SearchService) Search(ctx context.Context, query model.SearchQuery) (*model.SearchResult, error) {
	start := time.Now()

	// Set defaults
	if query.Size == 0 {
		query.Size = 20
	}
	if query.Size > 100 {
		query.Size = 100
	}

	// Perform search
	result, err := s.repository.Search(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}

	// Add search time
	result.Took = time.Since(start).Milliseconds()

	// Highlight results if requested
	if query.Highlight && query.Query != "" {
		for _, doc := range result.Documents {
			doc.Title = s.highlight(doc.Title, query.Query)
			doc.Content = s.highlight(doc.Content, query.Query)
		}
	}

	s.logger.Debug("Search completed",
		"query", query.Query,
		"hits", result.TotalHits,
		"took_ms", result.Took,
	)

	return result, nil
}

func (s *SearchService) Delete(ctx context.Context, index model.SearchIndex, id string) error {
	if err := s.repository.Delete(ctx, index, id); err != nil {
		return fmt.Errorf("failed to delete document: %w", err)
	}

	s.logger.Debug("Document deleted", "index", index, "id", id)
	return nil
}

func (s *SearchService) Update(ctx context.Context, doc *model.SearchDocument) error {
	doc.UpdatedAt = time.Now()
	
	if err := s.repository.Update(ctx, doc); err != nil {
		return fmt.Errorf("failed to update document: %w", err)
	}

	s.logger.Debug("Document updated", "id", doc.ID, "index", doc.Index)
	return nil
}

func (s *SearchService) Suggest(ctx context.Context, prefix string, index model.SearchIndex) ([]string, error) {
	suggestions, err := s.repository.Suggest(ctx, prefix, index)
	if err != nil {
		return nil, fmt.Errorf("failed to get suggestions: %w", err)
	}

	return suggestions, nil
}

func (s *SearchService) highlight(text, query string) string {
	// Simple highlighting implementation
	queryLower := strings.ToLower(query)
	textLower := strings.ToLower(text)
	
	if idx := strings.Index(textLower, queryLower); idx >= 0 {
		before := text[:idx]
		match := text[idx : idx+len(query)]
		after := text[idx+len(query):]
		return fmt.Sprintf("%s<mark>%s</mark>%s", before, match, after)
	}
	
	return text
}

func (s *SearchService) GetStats(ctx context.Context) (map[string]interface{}, error) {
	stats, err := s.repository.GetStats(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get stats: %w", err)
	}

	return stats, nil
}
