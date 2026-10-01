package providers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"noosfera/internal/domain"
)

func TestCrossRefProvider_FetchArticles(t *testing.T) {
	tests := []struct {
		name            string
		email           string
		ctxEmail        string
		query           string
		limit           int
		mockStatusCode  int
		mockResponse    interface{}
		wantUserAgent   string
		wantQueryValues map[string]string
		wantArticles    []domain.Article
		wantErr         bool
	}{
		{
			name:           "Cenário 1 (Mapeamento Válido com Polite Pool User-Agent)",
			email:          "pesquisador@exemplo.com",
			query:          "quantum",
			limit:          10,
			mockStatusCode: http.StatusOK,
			mockResponse: crossRefResponse{
				Status: "ok",
				Message: crossRefMessage{
					TotalResults: 1,
					Items: []crossRefItem{
						{
							DOI:   "10.1103/PhysRevLett.125.010501",
							Title: []string{"Quantum Computational Advantage Using Photons"},
							Author: []crossRefAuthor{
								{Given: "Han-Sen", Family: "Zhong"},
								{Given: "Hui", Family: "Wang"},
							},
							ContainerTitle: []string{"Physical Review Letters"},
							PublishedPrint: &crossRefDate{
								DateParts: [][]int{{2020, 12}},
							},
							IsReferencedByCount: 820,
							URL:                 "http://dx.doi.org/10.1103/PhysRevLett.125.010501",
							Abstract:            "Quantum computers promise to solve certain problems...",
						},
					},
				},
			},
			wantUserAgent: "Noosfera/1.0 (mailto:pesquisador@exemplo.com)",
			wantQueryValues: map[string]string{
				"query": "quantum",
				"rows":  "10",
			},
			wantArticles: []domain.Article{
				{
					ID:             "crossref:10.1103/PhysRevLett.125.010501",
					Title:          "Quantum Computational Advantage Using Photons",
					Authors:        []string{"Han-Sen Zhong", "Hui Wang"},
					Year:           2020,
					DOI:            "10.1103/PhysRevLett.125.010501",
					URL:            "http://dx.doi.org/10.1103/PhysRevLett.125.010501",
					Journal:        "Physical Review Letters",
					Citations:      820,
					Abstract:       "Quantum computers promise to solve certain problems...",
					SourceProvider: "CrossRef",
				},
			},
			wantErr: false,
		},
		{
			name:           "Cenário 2 (Resposta com lista de itens vazia)",
			email:          "teste@dominio.com",
			query:          "inexistent_term_xyz",
			limit:          5,
			mockStatusCode: http.StatusOK,
			mockResponse: crossRefResponse{
				Status: "ok",
				Message: crossRefMessage{
					TotalResults: 0,
					Items:        []crossRefItem{},
				},
			},
			wantUserAgent: "Noosfera/1.0 (mailto:teste@dominio.com)",
			wantArticles:  []domain.Article{},
			wantErr:       false,
		},
		{
			name:           "Cenário 3 (Erro HTTP 500 do servidor)",
			email:          "fail@dominio.com",
			query:          "error",
			limit:          10,
			mockStatusCode: http.StatusInternalServerError,
			mockResponse:   "Internal Server Error",
			wantArticles:   nil,
			wantErr:        true,
		},
		{
			name:           "Cenário 4 (E-mail repassado via context.Context)",
			email:          "",
			ctxEmail:       "context_user@universidade.edu.br",
			query:          "bioinformatics",
			limit:          5,
			mockStatusCode: http.StatusOK,
			mockResponse: crossRefResponse{
				Status: "ok",
				Message: crossRefMessage{
					TotalResults: 0,
					Items:        []crossRefItem{},
				},
			},
			wantUserAgent: "Noosfera/1.0 (mailto:context_user@universidade.edu.br)",
			wantArticles:  []domain.Article{},
			wantErr:       false,
		},
		{
			name:           "Cenário 5 (Sem e-mail - User-Agent padrão)",
			email:          "",
			ctxEmail:       "",
			query:          "science",
			limit:          1,
			mockStatusCode: http.StatusOK,
			mockResponse: crossRefResponse{
				Status: "ok",
				Message: crossRefMessage{
					TotalResults: 0,
					Items:        []crossRefItem{},
				},
			},
			wantUserAgent: "Noosfera/1.0",
			wantArticles:  []domain.Article{},
			wantErr:       false,
		},
		{
			name:           "Cenário 6 (Abstract com tags XML/JATS sanitizado)",
			email:          "pesquisador@exemplo.com",
			query:          "photonics",
			limit:          1,
			mockStatusCode: http.StatusOK,
			mockResponse: crossRefResponse{
				Status: "ok",
				Message: crossRefMessage{
					TotalResults: 1,
					Items: []crossRefItem{
						{
							DOI:      "10.1000/182",
							Title:    []string{"Sample Title"},
							Abstract: "<jats:title>Abstract</jats:title><jats:p>This is a <jats:italic>sample</jats:italic> abstract &amp; overview.</jats:p>",
						},
					},
				},
			},
			wantUserAgent: "Noosfera/1.0 (mailto:pesquisador@exemplo.com)",
			wantArticles: []domain.Article{
				{
					ID:             "crossref:10.1000/182",
					Title:          "Sample Title",
					Authors:        []string{},
					Year:           0,
					DOI:            "10.1000/182",
					URL:            "https://doi.org/10.1000/182",
					Journal:        "",
					Citations:      0,
					Abstract:       "Abstract This is a sample abstract & overview.",
					SourceProvider: "CrossRef",
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var capturedUserAgent string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				capturedUserAgent = r.Header.Get("User-Agent")

				if len(tt.wantQueryValues) > 0 {
					q := r.URL.Query()
					for k, v := range tt.wantQueryValues {
						if q.Get(k) != v {
							t.Errorf("parâmetro query esperado %s=%s, obtido %s", k, v, q.Get(k))
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
			}))
			defer server.Close()

			provider := NewCrossRefProvider(server.Client(), tt.email)
			provider.SetBaseURL(server.URL)

			ctx := context.Background()
			if tt.ctxEmail != "" {
				ctx = context.WithValue(ctx, "user_email", tt.ctxEmail)
			}

			articles, err := provider.FetchArticles(ctx, tt.query, tt.limit)

			if (err != nil) != tt.wantErr {
				t.Fatalf("FetchArticles() erro = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.wantUserAgent != "" && capturedUserAgent != tt.wantUserAgent {
				t.Errorf("User-Agent esperado '%s', obtido '%s'", tt.wantUserAgent, capturedUserAgent)
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

func TestCrossRefProvider_GetSourceID(t *testing.T) {
	provider := NewCrossRefProvider(nil)
	if sourceID := provider.GetSourceID(); sourceID != "CrossRef" {
		t.Errorf("GetSourceID() = %v, esperado %v", sourceID, "CrossRef")
	}

	provider.SetUserEmail("novo_email@dominio.com")
}

func TestCleanAbstract(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{input: "", expected: ""},
		{input: "   ", expected: ""},
		{input: "Simple abstract without tags.", expected: "Simple abstract without tags."},
		{input: "<jats:p>Paragraph text.</jats:p>", expected: "Paragraph text."},
		{input: "<jats:title>Title</jats:title><jats:p>First &amp; second <jats:bold>bold</jats:bold> item.</jats:p>", expected: "Title First & second bold item."},
		{input: "  <jats:sec> <jats:p> Spaced </jats:p> </jats:sec> ", expected: "Spaced"},
	}

	for _, tc := range tests {
		got := cleanAbstract(tc.input)
		if got != tc.expected {
			t.Errorf("cleanAbstract(%q) = %q, want %q", tc.input, got, tc.expected)
		}
	}
}
