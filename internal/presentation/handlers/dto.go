package handlers

import (
	"noosfera/internal/domain"
)

type SearchRequestDTO struct {
	Query        string `json:"query"`
	Limit        int    `json:"limit"`
	MinYear      int    `json:"min_year"`
	MinCitations int    `json:"min_citations"`
	UserEmail    string `json:"user_email,omitempty"`
}

type SearchResponseDTO struct {
	Articles        []domain.Article `json:"articles"`
	TotalFound      int              `json:"total_found"`
	ExecutionTimeMs int64            `json:"execution_time_ms"`
	SourceStats     map[string]int   `json:"source_stats"`
}

type ExportRequestDTO struct {
	Articles []domain.Article `json:"articles"`
	Format   string           `json:"format"`
}

type ProblemDetails struct {
	Code   string `json:"code"`
	Status int    `json:"status"`
	Detail string `json:"detail"`
}

type ErrorResponseDTO = ProblemDetails
