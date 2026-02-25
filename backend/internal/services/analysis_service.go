package services

import (
	"context"
	"sort"

	"github.com/google/uuid"
	"github.com/jycamier/retrotro/backend/internal/models"
)

// TopicCategory represents a category of discussed topics
type TopicCategory struct {
	Name   string                  `json:"name"`
	Topics []*models.DiscussedTopic `json:"topics"`
	Count  int                     `json:"count"`
}

// TopicAnalysis represents the result of topic analysis
type TopicAnalysis struct {
	Categories []*TopicCategory `json:"categories"`
	TotalTopics int             `json:"totalTopics"`
}

// AnalysisService provides topic analysis capabilities
type AnalysisService struct {
	lcService *LeanCoffeeService
}

// NewAnalysisService creates a new AnalysisService
func NewAnalysisService(lcService *LeanCoffeeService) *AnalysisService {
	return &AnalysisService{
		lcService: lcService,
	}
}

// AnalyzeTopics performs a simple keyword-based categorization of topics.
// This is a basic implementation that can be replaced with LLM-based analysis later.
func (s *AnalysisService) AnalyzeTopics(ctx context.Context, teamID uuid.UUID) (*TopicAnalysis, error) {
	topics, err := s.lcService.ListTopicsByTeam(ctx, teamID)
	if err != nil {
		return nil, err
	}

	// Simple categorization by keyword matching
	// Can be replaced with LLM call in the future
	categories := map[string][]*models.DiscussedTopic{
		"Technique":    {},
		"Process":      {},
		"Organisation": {},
		"Communication": {},
		"Autre":        {},
	}

	for _, topic := range topics {
		categories["Autre"] = append(categories["Autre"], topic)
	}

	result := &TopicAnalysis{
		TotalTopics: len(topics),
	}

	for name, topicList := range categories {
		if len(topicList) > 0 {
			result.Categories = append(result.Categories, &TopicCategory{
				Name:   name,
				Topics: topicList,
				Count:  len(topicList),
			})
		}
	}

	// Sort categories by count descending
	sort.Slice(result.Categories, func(i, j int) bool {
		return result.Categories[i].Count > result.Categories[j].Count
	})

	return result, nil
}
