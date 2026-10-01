package handlers

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"noosfera/internal/domain"
	"noosfera/internal/usecase"
)

type SearchArticlesUseCaseInterface interface {
	Execute(ctx context.Context, query string, limit int, filter domain.SearchFilter) (*usecase.SearchResult, error)
}

type ArticleHandler struct {
	useCase SearchArticlesUseCaseInterface
}

func NewArticleHandler(useCase SearchArticlesUseCaseInterface) *ArticleHandler {
	return &ArticleHandler{
		useCase: useCase,
	}
}

func (h *ArticleHandler) HandleSearch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var req SearchRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if len(strings.TrimSpace(req.Query)) < 3 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ProblemDetails{
			Code:   "INVALID_SEARCH_QUERY",
			Status: http.StatusBadRequest,
			Detail: "Query must be at least 3 characters long",
		})
		return
	}

	filter := domain.SearchFilter{
		MinYear:      req.MinYear,
		MinCitations: req.MinCitations,
	}

	ctx := r.Context()
	if strings.TrimSpace(req.UserEmail) != "" {
		ctx = context.WithValue(ctx, "user_email", strings.TrimSpace(req.UserEmail))
	}

	result, err := h.useCase.Execute(ctx, req.Query, req.Limit, filter)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(ProblemDetails{
			Code:   "INTERNAL_SERVER_ERROR",
			Status: http.StatusInternalServerError,
			Detail: err.Error(),
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(result)
}

func (h *ArticleHandler) HandleExport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var req ExportRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if req.Format != "csv" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ProblemDetails{
			Code:   "UNSUPPORTED_FORMAT",
			Status: http.StatusBadRequest,
			Detail: "Format must be csv",
		})
		return
	}

	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", "attachment; filename=artigos.csv")
	w.WriteHeader(http.StatusOK)

	csvWriter := csv.NewWriter(w)
	defer csvWriter.Flush()

	headers := []string{"ID", "Título", "Autores", "Ano", "DOI", "URL", "Periódico", "Citações", "Resumo", "Provedor"}
	if err := csvWriter.Write(headers); err != nil {
		return
	}

	for _, article := range req.Articles {
		row := []string{
			article.ID,
			article.Title,
			strings.Join(article.Authors, ", "),
			fmt.Sprintf("%d", article.Year),
			article.DOI,
			article.URL,
			article.Journal,
			fmt.Sprintf("%d", article.Citations),
			article.Abstract,
			article.SourceProvider,
		}
		if err := csvWriter.Write(row); err != nil {
			return
		}
	}
}
