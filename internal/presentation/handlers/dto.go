package handlers

import (
	"noosfera/internal/domain"
)

type SearchRequestDTO struct {
	Query        string   `json:"query"`
	Limit        int      `json:"limit"`
	MinYear      int      `json:"min_year"`
	MaxYear      int      `json:"max_year,omitempty"`
	YearFrom     int      `json:"year_from,omitempty"`
	YearTo       int      `json:"year_to,omitempty"`
	MinCitations int      `json:"min_citations"`
	Journal      string   `json:"journal,omitempty"`
	UserEmail    string   `json:"user_email,omitempty"`
	Providers    []string `json:"providers,omitempty"`
	IsOpenAccess *bool    `json:"is_open_access,omitempty"`
	DocType      string   `json:"doc_type,omitempty"`
	Language     string   `json:"language,omitempty"`
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
