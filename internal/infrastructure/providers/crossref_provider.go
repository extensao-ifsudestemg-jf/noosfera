package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"noosfera/internal/domain"
)

var _ domain.ArticleProvider = (*CrossRefProvider)(nil)

type CrossRefProvider struct {
	client    *http.Client
	baseURL   string
	userEmail string
}

func NewCrossRefProvider(client *http.Client, userEmail ...string) *CrossRefProvider {
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	email := ""
	if len(userEmail) > 0 {
		email = strings.TrimSpace(userEmail[0])
	}
	return &CrossRefProvider{
		client:    client,
		baseURL:   "https://api.crossref.org/works",
		userEmail: email,
	}
}

func (p *CrossRefProvider) SetBaseURL(baseURL string) {
	p.baseURL = baseURL
}

func (p *CrossRefProvider) SetUserEmail(email string) {
	p.userEmail = strings.TrimSpace(email)
}

func (p *CrossRefProvider) GetSourceID() string {
	return "CrossRef"
}

type crossRefResponse struct {
	Status  string          `json:"status"`
	Message crossRefMessage `json:"message"`
}

type crossRefMessage struct {
	TotalResults int            `json:"total-results"`
	Items        []crossRefItem `json:"items"`
}

type crossRefItem struct {
	DOI                 string           `json:"DOI"`
	Title               []string         `json:"title"`
	Author              []crossRefAuthor `json:"author"`
	ContainerTitle      []string         `json:"container-title"`
	PublishedPrint      *crossRefDate    `json:"published-print"`
	PublishedOnline     *crossRefDate    `json:"published-online"`
	Created             *crossRefDate    `json:"created"`
	Issued              *crossRefDate    `json:"issued"`
	IsReferencedByCount int              `json:"is-referenced-by-count"`
	URL                 string           `json:"URL"`
	Abstract            string           `json:"abstract"`
}

type crossRefAuthor struct {
	Given  string `json:"given"`
	Family string `json:"family"`
	Name   string `json:"name"`
}

type crossRefDate struct {
	DateParts [][]int `json:"date-parts"`
}

func (d *crossRefDate) getYear() int {
	if d != nil && len(d.DateParts) > 0 && len(d.DateParts[0]) > 0 {
		return d.DateParts[0][0]
	}
	return 0
}

func (p *CrossRefProvider) FetchArticles(ctx context.Context, query string, limit int) ([]domain.Article, error) {
	sep := "?"
	if strings.Contains(p.baseURL, "?") {
		sep = "&"
	}

	reqURL := fmt.Sprintf("%s%squery=%s&rows=%d", p.baseURL, sep, url.QueryEscape(query), limit)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("falha ao criar requisição para CrossRef: %w", err)
	}

	email := p.userEmail
	if ctxEmail, ok := ctx.Value("user_email").(string); ok && strings.TrimSpace(ctxEmail) != "" {
		email = strings.TrimSpace(ctxEmail)
	}

	if email != "" {
		req.Header.Set("User-Agent", fmt.Sprintf("Noosfera/1.0 (mailto:%s)", email))
	} else {
		req.Header.Set("User-Agent", "Noosfera/1.0")
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("falha ao executar requisição HTTP para CrossRef: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("erro HTTP retornado pelo servidor CrossRef: status %d", resp.StatusCode)
	}

	var crResp crossRefResponse
	if err := json.NewDecoder(resp.Body).Decode(&crResp); err != nil {
		return nil, fmt.Errorf("falha ao desserializar resposta JSON do CrossRef: %w", err)
	}

	items := crResp.Message.Items
	articles := make([]domain.Article, 0, len(items))

	for idx, item := range items {
		id := item.DOI
		if id == "" {
			if item.URL != "" {
				id = item.URL
			} else {
				id = fmt.Sprintf("item-%d", idx)
			}
		}
		if !strings.HasPrefix(id, "crossref:") {
			id = "crossref:" + id
		}

		title := ""
		if len(item.Title) > 0 {
			title = strings.TrimSpace(item.Title[0])
		}

		authors := make([]string, 0, len(item.Author))
		for _, auth := range item.Author {
			if auth.Name != "" {
				authors = append(authors, auth.Name)
			} else {
				fullName := strings.TrimSpace(auth.Given + " " + auth.Family)
				if fullName != "" {
					authors = append(authors, fullName)
				}
			}
		}

		year := item.PublishedPrint.getYear()
		if year == 0 {
			year = item.PublishedOnline.getYear()
		}
		if year == 0 {
			year = item.Issued.getYear()
		}
		if year == 0 {
			year = item.Created.getYear()
		}

		journal := ""
		if len(item.ContainerTitle) > 0 {
			journal = strings.TrimSpace(item.ContainerTitle[0])
		}

		doi := item.DOI
		articleURL := item.URL
		if articleURL == "" && doi != "" {
			articleURL = "https://doi.org/" + doi
		}

		article := domain.Article{
			ID:             id,
			Title:          title,
			Authors:        authors,
			Year:           year,
			DOI:            doi,
			URL:            articleURL,
			Journal:        journal,
			Citations:      item.IsReferencedByCount,
			Abstract:       cleanAbstract(item.Abstract),
			SourceProvider: "CrossRef",
		}

		articles = append(articles, article)
	}

	return articles, nil
}

var xmlTagRegex = regexp.MustCompile(`<[^>]*>`)

func cleanAbstract(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	unescaped := html.UnescapeString(raw)
	stripped := xmlTagRegex.ReplaceAllString(unescaped, " ")
	return strings.Join(strings.Fields(stripped), " ")
}
