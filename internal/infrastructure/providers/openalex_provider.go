package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"noosfera/internal/domain"
)

var _ domain.ArticleProvider = (*OpenAlexProvider)(nil)

type OpenAlexProvider struct {
	client  *http.Client
	baseURL string
	apiKey  string
}

func NewOpenAlexProvider(client *http.Client, apiKey string) *OpenAlexProvider {
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	return &OpenAlexProvider{
		client:  client,
		baseURL: "https://api.openalex.org/works",
		apiKey:  apiKey,
	}
}

func (p *OpenAlexProvider) SetBaseURL(baseURL string) {
	p.baseURL = baseURL
}

func (p *OpenAlexProvider) GetSourceID() string {
	return "OpenAlex"
}

type openAlexResponse struct {
	Results []openAlexWork `json:"results"`
}

type openAlexWork struct {
	ID                    string               `json:"id"`
	DisplayName           string               `json:"display_name"`
	PublicationYear       int                  `json:"publication_year"`
	DOI                   string               `json:"doi"`
	PrimaryLocation       openAlexLocation     `json:"primary_location"`
	Authorships           []openAlexAuthorship `json:"authorships"`
	CitedByCount          int                  `json:"cited_by_count"`
	AbstractInvertedIndex map[string][]int     `json:"abstract_inverted_index"`
	OpenAccess            *openAlexOpenAccess  `json:"open_access"`
	Type                  string               `json:"type"`
	Language              string               `json:"language"`
}

type openAlexOpenAccess struct {
	IsOpenAccess bool `json:"is_oa"`
}

type openAlexLocation struct {
	Source         openAlexSource `json:"source"`
	LandingPageURL string         `json:"landing_page_url"`
}

type openAlexSource struct {
	DisplayName string `json:"display_name"`
}

type openAlexAuthorship struct {
	Author openAlexAuthor `json:"author"`
}

type openAlexAuthor struct {
	DisplayName string `json:"display_name"`
}

func (p *OpenAlexProvider) FetchArticles(ctx context.Context, query string, limit int) ([]domain.Article, error) {
	sep := "?"
	if strings.Contains(p.baseURL, "?") {
		sep = "&"
	}

	reqURL := fmt.Sprintf("%s%ssearch=%s&per_page=%d", p.baseURL, sep, url.QueryEscape(query), limit)
	if p.apiKey != "" {
		reqURL += fmt.Sprintf("&api_key=%s", url.QueryEscape(p.apiKey))
	}

	var alexFilters []string
	if sf, ok := ctx.Value("search_filter").(domain.SearchFilter); ok {
		if sf.IsOpenAccess != nil {
			if *sf.IsOpenAccess {
				alexFilters = append(alexFilters, "is_oa:true")
			} else {
				alexFilters = append(alexFilters, "is_oa:false")
			}
		}
		if sf.DocType != "" {
			alexFilters = append(alexFilters, "type:"+strings.ToLower(sf.DocType))
		}
		if sf.Language != "" {
			alexFilters = append(alexFilters, "language:"+strings.ToLower(sf.Language))
		}
		if sf.MinYear > 0 {
			alexFilters = append(alexFilters, fmt.Sprintf("from_publication_date:%d-01-01", sf.MinYear))
		}
		if sf.MaxYear > 0 {
			alexFilters = append(alexFilters, fmt.Sprintf("to_publication_date:%d-12-31", sf.MaxYear))
		}
	}
	if len(alexFilters) > 0 {
		reqURL += "&filter=" + url.QueryEscape(strings.Join(alexFilters, ","))
	}

	log.Printf("[OpenAlex] Requisitando URL: %s", reqURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("falha ao criar requisição para OpenAlex: %w", err)
	}

	email := "contato@noosfera.local"
	if ctxEmail, ok := ctx.Value("user_email").(string); ok && strings.TrimSpace(ctxEmail) != "" {
		email = strings.TrimSpace(ctxEmail)
	}
	req.Header.Set("User-Agent", fmt.Sprintf("Noosfera/1.0 (mailto:%s)", email))

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("falha ao executar requisição HTTP para OpenAlex: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		log.Printf("[OpenAlex ERROR] Status: %d | Body: %s", resp.StatusCode, string(body))
		return nil, fmt.Errorf("erro HTTP retornado pelo servidor OpenAlex: status %d", resp.StatusCode)
	}

	var alexResp openAlexResponse
	if err := json.NewDecoder(resp.Body).Decode(&alexResp); err != nil {
		return nil, fmt.Errorf("falha ao desserializar resposta JSON do OpenAlex: %w", err)
	}

	articles := make([]domain.Article, 0, len(alexResp.Results))
	for _, work := range alexResp.Results {
		id := work.ID
		if !strings.HasPrefix(id, "openalex:") {
			id = "openalex:" + id
		}

		authors := make([]string, 0, len(work.Authorships))
		for _, auth := range work.Authorships {
			if auth.Author.DisplayName != "" {
				authors = append(authors, auth.Author.DisplayName)
			}
		}

		articleURL := work.PrimaryLocation.LandingPageURL
		if articleURL == "" {
			articleURL = work.DOI
		}

		abstract := buildAbstractFromInvertedIndex(work.AbstractInvertedIndex)

		var isOA *bool
		if work.OpenAccess != nil {
			val := work.OpenAccess.IsOpenAccess
			isOA = &val
		}
		article := domain.Article{
			ID:             id,
			Title:          work.DisplayName,
			Authors:        authors,
			Year:           work.PublicationYear,
			DOI:            work.DOI,
			URL:            articleURL,
			Journal:        work.PrimaryLocation.Source.DisplayName,
			Citations:      work.CitedByCount,
			Abstract:       abstract,
			IsOpenAccess:   isOA,
			DocType:        work.Type,
			Language:       work.Language,
			SourceProvider: "OpenAlex",
		}

		articles = append(articles, article)
	}

	return articles, nil
}

func buildAbstractFromInvertedIndex(invertedIndex map[string][]int) string {
	if len(invertedIndex) == 0 {
		return ""
	}

	maxPos := -1
	for _, positions := range invertedIndex {
		for _, pos := range positions {
			if pos > maxPos {
				maxPos = pos
			}
		}
	}

	if maxPos < 0 {
		return ""
	}

	words := make([]string, maxPos+1)
	for word, positions := range invertedIndex {
		for _, pos := range positions {
			if pos >= 0 && pos <= maxPos {
				words[pos] = word
			}
		}
	}

	return strings.Join(words, " ")
}
