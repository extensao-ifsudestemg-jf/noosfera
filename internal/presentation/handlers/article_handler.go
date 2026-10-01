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

	var cleanProviders []string
	if len(req.Providers) == 0 {
		if provQuery := r.URL.Query().Get("providers"); provQuery != "" {
			for _, p := range strings.Split(provQuery, ",") {
				if trimmed := strings.TrimSpace(p); trimmed != "" {
					cleanProviders = append(cleanProviders, trimmed)
				}
			}
		}
	} else {
		for _, p := range req.Providers {
			for _, sub := range strings.Split(p, ",") {
				if trimmed := strings.TrimSpace(sub); trimmed != "" {
					cleanProviders = append(cleanProviders, trimmed)
				}
			}
		}
	}

	filter := domain.SearchFilter{
		MinYear:      req.MinYear,
		MinCitations: req.MinCitations,
		Providers:    cleanProviders,
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

	switch strings.ToLower(req.Format) {
	case "csv":
		w.Header().Set("Content-Type", "text/csv")
		w.Header().Set("Content-Disposition", "attachment; filename=noosfera_artigos.csv")
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

	case "bib", "bibtex":
		w.Header().Set("Content-Type", "application/x-bibtex; charset=utf-8")
		w.Header().Set("Content-Disposition", "attachment; filename=noosfera_artigos.bib")
		w.WriteHeader(http.StatusOK)

		seenKeys := make(map[string]int)
		for idx, article := range req.Articles {
			key := generateBibTeXKey(article, seenKeys)
			w.Write([]byte(formatBibTeXEntry(key, article)))
			if idx < len(req.Articles)-1 {
				w.Write([]byte("\n\n"))
			} else {
				w.Write([]byte("\n"))
			}
		}

	case "ris":
		w.Header().Set("Content-Type", "application/x-research-info-systems; charset=utf-8")
		w.Header().Set("Content-Disposition", "attachment; filename=noosfera_artigos.ris")
		w.WriteHeader(http.StatusOK)

		for idx, article := range req.Articles {
			w.Write([]byte(formatRISEntry(article)))
			if idx < len(req.Articles)-1 {
				w.Write([]byte("\r\n\r\n"))
			} else {
				w.Write([]byte("\r\n"))
			}
		}

	default:
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ProblemDetails{
			Code:   "UNSUPPORTED_FORMAT",
			Status: http.StatusBadRequest,
			Detail: "Format must be csv, bib, bibtex or ris",
		})
	}
}

func extractLastName(author string) string {
	author = strings.TrimSpace(author)
	if strings.Contains(author, ",") {
		return strings.TrimSpace(strings.Split(author, ",")[0])
	}
	fields := strings.Fields(author)
	if len(fields) > 0 {
		return fields[len(fields)-1]
	}
	return author
}

func sanitizeKeyChar(r rune) rune {
	switch r {
	case 'á', 'à', 'ã', 'â', 'ä':
		return 'a'
	case 'é', 'è', 'ê', 'ë':
		return 'e'
	case 'í', 'ì', 'î', 'ï':
		return 'i'
	case 'ó', 'ò', 'õ', 'ô', 'ö':
		return 'o'
	case 'ú', 'ù', 'û', 'ü':
		return 'u'
	case 'ç':
		return 'c'
	case 'ñ':
		return 'n'
	}
	if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
		return r
	}
	return -1
}

func cleanKeyPart(s string) string {
	var sb strings.Builder
	for _, r := range strings.ToLower(s) {
		if c := sanitizeKeyChar(r); c != -1 {
			sb.WriteRune(c)
		}
	}
	return sb.String()
}

func generateBibTeXKey(article domain.Article, seenKeys map[string]int) string {
	base := ""
	if len(article.Authors) > 0 {
		base = cleanKeyPart(extractLastName(article.Authors[0]))
	}
	if base == "" {
		base = "article"
	}
	yearStr := "nodate"
	if article.Year > 0 {
		yearStr = fmt.Sprintf("%d", article.Year)
	}
	key := base + yearStr
	count, exists := seenKeys[key]
	if !exists {
		seenKeys[key] = 1
		return key
	}
	seenKeys[key] = count + 1
	suffix := string(rune('a' + count))
	return key + suffix
}

func formatBibTeXEntry(key string, article domain.Article) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("@article{%s,\n", key))
	if article.Title != "" {
		sb.WriteString(fmt.Sprintf("  title = {%s},\n", article.Title))
	}
	if len(article.Authors) > 0 {
		sb.WriteString(fmt.Sprintf("  author = {%s},\n", strings.Join(article.Authors, " and ")))
	}
	if article.Journal != "" {
		sb.WriteString(fmt.Sprintf("  journal = {%s},\n", article.Journal))
	}
	if article.Year > 0 {
		sb.WriteString(fmt.Sprintf("  year = {%d},\n", article.Year))
	}
	if article.DOI != "" {
		sb.WriteString(fmt.Sprintf("  doi = {%s},\n", article.DOI))
	}
	if article.URL != "" {
		sb.WriteString(fmt.Sprintf("  url = {%s},\n", article.URL))
	}
	if article.Abstract != "" {
		sb.WriteString(fmt.Sprintf("  abstract = {%s},\n", article.Abstract))
	}
	content := sb.String()
	content = strings.TrimSuffix(content, ",\n") + "\n"
	content += "}"
	return content
}

func formatRISEntry(article domain.Article) string {
	var lines []string
	lines = append(lines, "TY  - JOUR")
	if article.Title != "" {
		lines = append(lines, "TI  - "+article.Title)
	}
	for _, auth := range article.Authors {
		trimmed := strings.TrimSpace(auth)
		if trimmed != "" {
			lines = append(lines, "AU  - "+trimmed)
		}
	}
	if article.Journal != "" {
		lines = append(lines, "JO  - "+article.Journal)
	}
	if article.Year > 0 {
		lines = append(lines, fmt.Sprintf("PY  - %d", article.Year))
	}
	if article.DOI != "" {
		lines = append(lines, "DO  - "+article.DOI)
	}
	if article.URL != "" {
		lines = append(lines, "UR  - "+article.URL)
	}
	if article.Abstract != "" {
		lines = append(lines, "AB  - "+article.Abstract)
	}
	lines = append(lines, "ER  - ")
	return strings.Join(lines, "\r\n")
}
