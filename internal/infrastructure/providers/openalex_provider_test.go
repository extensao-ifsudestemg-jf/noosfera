package providers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	"noosfera/internal/domain"
)

type openAlexMockResponse struct {
	Results []openAlexMockWork `json:"results"`
}

type openAlexMockWork struct {
	ID                    string               `json:"id"`
	DisplayName           string               `json:"display_name"`
	PublicationYear       int                  `json:"publication_year"`
	DOI                   string               `json:"doi"`
	PrimaryLocation       openAlexMockLocation `json:"primary_location"`
	Authorships           []openAlexMockAuthor `json:"authorships"`
	CitedByCount          int                  `json:"cited_by_count"`
	AbstractInvertedIndex map[string][]int     `json:"abstract_inverted_index"`
	OpenAccess            *openAlexOpenAccess  `json:"open_access"`
	Type                  string               `json:"type"`
	Language              string               `json:"language"`
}

type openAlexMockLocation struct {
	Source struct {
		DisplayName string `json:"display_name"`
	} `json:"source"`
	LandingPageURL string `json:"landing_page_url"`
}

type openAlexMockAuthor struct {
	Author struct {
		DisplayName string `json:"display_name"`
	} `json:"author"`
}

func TestOpenAlexProvider_FetchArticles(t *testing.T) {
	tests := []struct {
		name           string
		apiKey         string
		query          string
		limit          int
		mockStatusCode int
		mockResponse   interface{}
		mockDelay      time.Duration
		setupContext   func() (context.Context, context.CancelFunc)
		wantReqQuery   map[string]string
		wantArticles   []domain.Article
		wantErr        bool
	}{
		{
			name:           "Cenário 1 (Mapeamento Válido [CT-I01])",
			apiKey:         "",
			query:          "golang",
			limit:          10,
			mockStatusCode: http.StatusOK,
			mockResponse: openAlexMockResponse{
				Results: []openAlexMockWork{
					{
						ID:              "https://openalex.org/W123",
						DisplayName:     "Test Article",
						PublicationYear: 2021,
						DOI:             "https://doi.org/10.123/abc",
						PrimaryLocation: openAlexMockLocation{
							Source: struct {
								DisplayName string `json:"display_name"`
							}{DisplayName: "Test Journal"},
							LandingPageURL: "https://test.com/article",
						},
						Authorships: []openAlexMockAuthor{
							{Author: struct {
								DisplayName string `json:"display_name"`
							}{DisplayName: "John Doe"}},
						},
						CitedByCount: 42,
						AbstractInvertedIndex: map[string][]int{
							"This":  {0},
							"is":    {1},
							"a":     {2},
							"test.": {3},
						},
					},
				},
			},
			setupContext: func() (context.Context, context.CancelFunc) {
				return context.WithTimeout(context.Background(), 5*time.Second)
			},
			wantReqQuery: map[string]string{
				"search":   "golang",
				"per_page": "10",
			},
			wantArticles: []domain.Article{
				{
					ID:             "openalex:https://openalex.org/W123",
					Title:          "Test Article",
					Authors:        []string{"John Doe"},
					Year:           2021,
					DOI:            "https://doi.org/10.123/abc",
					URL:            "https://test.com/article",
					Journal:        "Test Journal",
					Citations:      42,
					Abstract:       "This is a test.",
					SourceProvider: "OpenAlex",
				},
			},
			wantErr: false,
		},
		{
			name:           "Cenário 2 (Chave de API)",
			apiKey:         "SUA_CHAVE",
			query:          "machine learning",
			limit:          5,
			mockStatusCode: http.StatusOK,
			mockResponse:   openAlexMockResponse{Results: []openAlexMockWork{}},
			setupContext: func() (context.Context, context.CancelFunc) {
				return context.WithTimeout(context.Background(), 5*time.Second)
			},
			wantReqQuery: map[string]string{
				"search":   "machine learning",
				"per_page": "5",
				"api_key":  "SUA_CHAVE",
			},
			wantArticles: []domain.Article{},
			wantErr:      false,
		},
		{
			name:           "Cenário 3 (Resposta Vazia [CT-I02])",
			apiKey:         "",
			query:          "xyz987123_nonexistent",
			limit:          10,
			mockStatusCode: http.StatusOK,
			mockResponse:   openAlexMockResponse{Results: []openAlexMockWork{}},
			setupContext: func() (context.Context, context.CancelFunc) {
				return context.WithTimeout(context.Background(), 5*time.Second)
			},
			wantArticles: []domain.Article{},
			wantErr:      false,
		},
		{
			name:           "Cenário 4 (Erro de Servidor HTTP)",
			apiKey:         "",
			query:          "golang",
			limit:          10,
			mockStatusCode: http.StatusInternalServerError,
			mockResponse:   "Internal Server Error",
			setupContext: func() (context.Context, context.CancelFunc) {
				return context.WithTimeout(context.Background(), 5*time.Second)
			},
			wantArticles: nil,
			wantErr:      true,
		},
		{
			name:           "Cenário 5 (Timeout de Contexto)",
			apiKey:         "",
			query:          "timeout_test",
			limit:          10,
			mockStatusCode: http.StatusOK,
			mockResponse:   openAlexMockResponse{Results: []openAlexMockWork{}},
			mockDelay:      100 * time.Millisecond,
			setupContext: func() (context.Context, context.CancelFunc) {
				return context.WithTimeout(context.Background(), 10*time.Millisecond)
			},
			wantArticles: nil,
			wantErr:      true,
		},
		{
			name:           "Cenário 6 (Filtros Avançados URL e Mapeamento)",
			apiKey:         "",
			query:          "machine learning",
			limit:          5,
			mockStatusCode: http.StatusOK,
			mockResponse: openAlexMockResponse{
				Results: []openAlexMockWork{
					{
						ID:              "https://openalex.org/W999",
						DisplayName:     "ML Paper",
						PublicationYear: 2022,
						DOI:             "https://doi.org/10.123/ml",
						PrimaryLocation: openAlexMockLocation{
							Source: struct {
								DisplayName string `json:"display_name"`
							}{DisplayName: "ML Journal"},
							LandingPageURL: "https://example.com/ml",
						},
						Authorships: []openAlexMockAuthor{
							{Author: struct {
								DisplayName string `json:"display_name"`
							}{DisplayName: "Alice Smith"}},
						},
						CitedByCount: 10,
						OpenAccess:   &openAlexOpenAccess{IsOpenAccess: true},
						Type:         "article",
						Language:     "pt",
					},
				},
			},
			mockDelay: 0,
			setupContext: func() (context.Context, context.CancelFunc) {
				trueVal := true
				filter := domain.SearchFilter{
					IsOpenAccess: &trueVal,
					DocType:      "article",
					Language:     "pt",
					MinYear:      2020,
					MaxYear:      2024,
				}
				ctx := context.WithValue(context.Background(), "search_filter", filter)
				return ctx, func() {}
			},
			wantReqQuery: map[string]string{
				"search":   "machine learning",
				"per_page": "5",
				"filter":   "is_oa:true,type:article,language:pt,from_publication_date:2020-01-01,to_publication_date:2024-12-31",
			},
			wantArticles: []domain.Article{
				{
					ID:             "openalex:https://openalex.org/W999",
					Title:          "ML Paper",
					Authors:        []string{"Alice Smith"},
					Year:           2022,
					DOI:            "https://doi.org/10.123/ml",
					URL:            "https://example.com/ml",
					Journal:        "ML Journal",
					Citations:      10,
					Abstract:       "",
					IsOpenAccess:   func() *bool { b := true; return &b }(),
					DocType:        "article",
					Language:       "pt",
					SourceProvider: "OpenAlex",
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tt.mockDelay > 0 {
					time.Sleep(tt.mockDelay)
				}

				if len(tt.wantReqQuery) > 0 {
					query := r.URL.Query()
					for k, v := range tt.wantReqQuery {
						if query.Get(k) != v {
							t.Errorf("parâmetro query esperado %s=%s, obtido %s", k, v, query.Get(k))
						}
					}
				}

				w.WriteHeader(tt.mockStatusCode)
				if tt.mockResponse != nil {
					switch v := tt.mockResponse.(type) {
					case string:
						_, _ = w.Write([]byte(v))
					default:
						_ = json.NewEncoder(w).Encode(v)
					}
				}
			})

			mockServer := httptest.NewServer(handler)
			defer mockServer.Close()

			provider := NewOpenAlexProvider(mockServer.Client(), tt.apiKey)
			provider.SetBaseURL(mockServer.URL)

			ctx, cancel := tt.setupContext()
			defer cancel()

			articles, err := provider.FetchArticles(ctx, tt.query, tt.limit)

			if (err != nil) != tt.wantErr {
				t.Fatalf("FetchArticles() erro = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr {
				if articles == nil {
					t.Fatalf("FetchArticles() retornou nil, esperava-se pelo menos []domain.Article{} vazio")
				}

				if len(articles) != len(tt.wantArticles) {
					t.Fatalf("FetchArticles() retornou %d itens, esperado %d", len(articles), len(tt.wantArticles))
				}

				for i := range articles {
					if !reflect.DeepEqual(articles[i], tt.wantArticles[i]) {
						t.Errorf("FetchArticles()[%d] =\n%+v\nEsperado:\n%+v", i, articles[i], tt.wantArticles[i])
					}
				}
			}
		})
	}
}

func TestOpenAlexProvider_GetSourceID(t *testing.T) {
	provider := NewOpenAlexProvider(nil, "")
	if sourceID := provider.GetSourceID(); sourceID != "OpenAlex" {
		t.Errorf("GetSourceID() = %v, esperado %v", sourceID, "OpenAlex")
	}
}
