package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"noosfera/internal/domain"
	"noosfera/internal/presentation/handlers"
	"noosfera/internal/usecase"
)

type MockSearchUseCase struct {
	Result *usecase.SearchResult
	Err    error
}

func (m *MockSearchUseCase) Execute(ctx context.Context, query string, limit int, filter domain.SearchFilter) (*usecase.SearchResult, error) {
	if m.Err != nil {
		return nil, m.Err
	}
	return m.Result, nil
}

func TestArticleHandler_HandleSearch(t *testing.T) {

	mockArticle := domain.Article{
		ID:             "openalex:https://openalex.org/W123",
		Title:          "TDD in Go",
		Authors:        []string{"John Doe"},
		Year:           2026,
		DOI:            "https://doi.org/10.123/abc",
		URL:            "https://test.com/article",
		Journal:        "Testing Journal",
		Citations:      42,
		Abstract:       "This is a test article about TDD in Go.",
		SourceProvider: "OpenAlex",
	}

	mockResult := &usecase.SearchResult{
		Articles:        []domain.Article{mockArticle},
		TotalFound:      1,
		ExecutionTimeMs: 120,
		SourceStats:     map[string]int{"OpenAlex": 1},
	}

	tests := []struct {
		name           string
		method         string
		requestBody    interface{}
		useCaseResult  *usecase.SearchResult
		useCaseErr     error
		expectedStatus int
		verifyResponse func(t *testing.T, body []byte)
	}{
		{
			name:   "Cenário 1 (Busca Válida)",
			method: http.MethodPost,
			requestBody: handlers.SearchRequestDTO{
				Query:        "testing",
				Limit:        10,
				MinYear:      2020,
				MinCitations: 5,
			},
			useCaseResult:  mockResult,
			useCaseErr:     nil,
			expectedStatus: http.StatusOK,
			verifyResponse: func(t *testing.T, body []byte) {
				var res usecase.SearchResult
				if err := json.Unmarshal(body, &res); err != nil {
					t.Fatalf("falha ao desmarcializar resposta: %v", err)
				}
				if len(res.Articles) != 1 {
					t.Errorf("esperado 1 artigo, obtido %d", len(res.Articles))
				}
				if res.Articles[0].Title != "TDD in Go" {
					t.Errorf("título do artigo incorreto: %s", res.Articles[0].Title)
				}
				if res.TotalFound != 1 {
					t.Errorf("esperado total de 1, obtido %d", res.TotalFound)
				}
			},
		},
		{
			name:   "Cenário 2 (Validação de Query Curta)",
			method: http.MethodPost,
			requestBody: handlers.SearchRequestDTO{
				Query: "ai",
			},
			useCaseResult:  nil,
			useCaseErr:     nil,
			expectedStatus: http.StatusBadRequest,
			verifyResponse: func(t *testing.T, body []byte) {
				var problem handlers.ProblemDetails
				if err := json.Unmarshal(body, &problem); err != nil {
					t.Fatalf("falha ao desmarcializar erro RFC 7807: %v", err)
				}
				if problem.Code != "INVALID_SEARCH_QUERY" {
					t.Errorf("esperado código 'INVALID_SEARCH_QUERY', obtido '%s'", problem.Code)
				}
				if problem.Status != http.StatusBadRequest {
					t.Errorf("esperado status %d no JSON, obtido %d", http.StatusBadRequest, problem.Status)
				}
			},
		},
		{
			name:           "Cenário 3 (Método HTTP Inválido)",
			method:         http.MethodGet,
			requestBody:    nil,
			useCaseResult:  nil,
			useCaseErr:     nil,
			expectedStatus: http.StatusMethodNotAllowed,
			verifyResponse: func(t *testing.T, body []byte) {

			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockUC := &MockSearchUseCase{
				Result: tt.useCaseResult,
				Err:    tt.useCaseErr,
			}
			handler := handlers.NewArticleHandler(mockUC)

			var reqBody []byte
			if tt.requestBody != nil {
				var err error
				reqBody, err = json.Marshal(tt.requestBody)
				if err != nil {
					t.Fatalf("falha ao serializar corpo do teste: %v", err)
				}
			}

			req := httptest.NewRequest(tt.method, "/api/v1/search", bytes.NewBuffer(reqBody))
			rec := httptest.NewRecorder()

			handler.HandleSearch(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Errorf("esperado status %d, obtido %d", tt.expectedStatus, rec.Code)
			}

			if tt.verifyResponse != nil {
				tt.verifyResponse(t, rec.Body.Bytes())
			}
		})
	}
}

func TestArticleHandler_HandleExport(t *testing.T) {
	mockArticles := []domain.Article{
		{
			ID:             "1",
			Title:          "Article with, comma",
			Authors:        []string{"Author One", "Author Two"},
			Year:           2025,
			DOI:            "https://doi.org/10.123/xyz",
			URL:            "https://test.com/1",
			Journal:        "Awesome Journal",
			Citations:      100,
			Abstract:       "Abstract text",
			SourceProvider: "OpenAlex",
		},
	}

	tests := []struct {
		name           string
		method         string
		requestBody    interface{}
		expectedStatus int
		verifyResponse func(t *testing.T, rec *httptest.ResponseRecorder)
	}{
		{
			name:   "Cenário 1 (Exportação CSV)",
			method: http.MethodPost,
			requestBody: handlers.ExportRequestDTO{
				Articles: mockArticles,
				Format:   "csv",
			},
			expectedStatus: http.StatusOK,
			verifyResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				contentType := rec.Header().Get("Content-Type")
				if contentType != "text/csv" {
					t.Errorf("esperado Content-Type 'text/csv', obtido '%s'", contentType)
				}

				bodyStr := rec.Body.String()

				headers := []string{"ID", "Título", "Autores", "Ano", "DOI", "URL", "Periódico", "Citações", "Resumo", "Provedor"}
				for _, h := range headers {
					if !strings.Contains(bodyStr, h) {
						t.Errorf("esperado cabeçalho '%s' no corpo CSV", h)
					}
				}

				expectedContent := `"Article with, comma"`
				if !strings.Contains(bodyStr, expectedContent) {
					t.Errorf("esperado conteúdo escapado %s no corpo CSV, obtido:\n%s", expectedContent, bodyStr)
				}
			},
		},
		{
			name:   "Cenário 2 (Formato Não Suportado)",
			method: http.MethodPost,
			requestBody: handlers.ExportRequestDTO{
				Articles: mockArticles,
				Format:   "pdf",
			},
			expectedStatus: http.StatusBadRequest,
			verifyResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				var problem handlers.ProblemDetails
				if err := json.Unmarshal(rec.Body.Bytes(), &problem); err != nil {
					t.Fatalf("falha ao desmarcializar erro: %v", err)
				}
				if problem.Code != "UNSUPPORTED_FORMAT" {
					t.Errorf("esperado código 'UNSUPPORTED_FORMAT', obtido '%s'", problem.Code)
				}
			},
		},
		{
			name:   "Cenário 3 (Exportação BibTeX)",
			method: http.MethodPost,
			requestBody: handlers.ExportRequestDTO{
				Articles: mockArticles,
				Format:   "bib",
			},
			expectedStatus: http.StatusOK,
			verifyResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				contentType := rec.Header().Get("Content-Type")
				if !strings.Contains(contentType, "x-bibtex") {
					t.Errorf("esperado Content-Type contendo 'x-bibtex', obtido '%s'", contentType)
				}
				bodyStr := rec.Body.String()
				expectedParts := []string{
					"@article{one2025,",
					"title = {Article with, comma}",
					"author = {Author One and Author Two}",
					"journal = {Awesome Journal}",
					"year = {2025}",
					"doi = {https://doi.org/10.123/xyz}",
					"abstract = {Abstract text}",
					"}",
				}
				for _, part := range expectedParts {
					if !strings.Contains(bodyStr, part) {
						t.Errorf("esperado trecho '%s' no corpo BibTeX:\n%s", part, bodyStr)
					}
				}
			},
		},
		{
			name:   "Cenário 4 (Exportação RIS)",
			method: http.MethodPost,
			requestBody: handlers.ExportRequestDTO{
				Articles: mockArticles,
				Format:   "ris",
			},
			expectedStatus: http.StatusOK,
			verifyResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				contentType := rec.Header().Get("Content-Type")
				if !strings.Contains(contentType, "research-info-systems") {
					t.Errorf("esperado Content-Type contendo 'research-info-systems', obtido '%s'", contentType)
				}
				bodyStr := rec.Body.String()
				expectedParts := []string{
					"TY  - JOUR",
					"TI  - Article with, comma",
					"AU  - Author One",
					"AU  - Author Two",
					"JO  - Awesome Journal",
					"PY  - 2025",
					"DO  - https://doi.org/10.123/xyz",
					"AB  - Abstract text",
					"ER  - ",
				}
				for _, part := range expectedParts {
					if !strings.Contains(bodyStr, part) {
						t.Errorf("esperado trecho '%s' no corpo RIS:\n%s", part, bodyStr)
					}
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := handlers.NewArticleHandler(&MockSearchUseCase{})

			var reqBody []byte
			if tt.requestBody != nil {
				var err error
				reqBody, err = json.Marshal(tt.requestBody)
				if err != nil {
					t.Fatalf("falha ao serializar corpo do teste: %v", err)
				}
			}

			req := httptest.NewRequest(tt.method, "/api/v1/export", bytes.NewBuffer(reqBody))
			rec := httptest.NewRecorder()

			handler.HandleExport(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Errorf("esperado status %d, obtido %d", tt.expectedStatus, rec.Code)
			}

			if tt.verifyResponse != nil {
				tt.verifyResponse(t, rec)
			}
		})
	}
}

func TestMockSearchUseCase_ExecuteError(t *testing.T) {
	expectedErr := errors.New("usecase error")
	mockUC := &MockSearchUseCase{
		Err: expectedErr,
	}
	_, err := mockUC.Execute(context.Background(), "test", 10, domain.SearchFilter{})
	if err != expectedErr {
		t.Errorf("esperado erro '%v', obtido '%v'", expectedErr, err)
	}
}

func TestSearchRequestDTO_UserEmail(t *testing.T) {

	jsonData := `{"query":"machine learning","limit":20,"min_year":2021,"min_citations":5,"user_email":"author@domain.com"}`
	var req handlers.SearchRequestDTO
	if err := json.Unmarshal([]byte(jsonData), &req); err != nil {
		t.Fatalf("falha ao desserializar SearchRequestDTO: %v", err)
	}
	if req.UserEmail != "author@domain.com" {
		t.Errorf("esperado user_email 'author@domain.com', obtido '%s'", req.UserEmail)
	}

	reqEmpty := handlers.SearchRequestDTO{
		Query: "data science",
		Limit: 10,
	}
	bytesOut, err := json.Marshal(reqEmpty)
	if err != nil {
		t.Fatalf("falha ao serializar SearchRequestDTO: %v", err)
	}
	if strings.Contains(string(bytesOut), "user_email") {
		t.Errorf("esperado que user_email fosse omitido com omitempty, obtido: %s", string(bytesOut))
	}
}

func TestSearchRequestDTO_Providers(t *testing.T) {
	jsonData := `{"query":"quantum","limit":10,"providers":["openalex","crossref"]}`
	var req handlers.SearchRequestDTO
	if err := json.Unmarshal([]byte(jsonData), &req); err != nil {
		t.Fatalf("falha ao desserializar SearchRequestDTO: %v", err)
	}
	if len(req.Providers) != 2 || req.Providers[0] != "openalex" || req.Providers[1] != "crossref" {
		t.Errorf("esperado providers [openalex, crossref], obtido %v", req.Providers)
	}
}

type ProviderCapturingSearchUseCase struct {
	CapturedFilter domain.SearchFilter
}

func (p *ProviderCapturingSearchUseCase) Execute(ctx context.Context, query string, limit int, filter domain.SearchFilter) (*usecase.SearchResult, error) {
	p.CapturedFilter = filter
	return &usecase.SearchResult{Articles: []domain.Article{}}, nil
}

func TestArticleHandler_HandleSearch_ProvidersQueryParam(t *testing.T) {
	mockUC := &ProviderCapturingSearchUseCase{}
	handler := handlers.NewArticleHandler(mockUC)

	body := bytes.NewBufferString(`{"query":"deep learning","limit":10}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/search?providers=openalex,crossref", body)
	rec := httptest.NewRecorder()

	handler.HandleSearch(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("esperado status 200, obtido %d", rec.Code)
	}
	if len(mockUC.CapturedFilter.Providers) != 2 {
		t.Fatalf("esperado 2 providers capturados, obtido %d", len(mockUC.CapturedFilter.Providers))
	}
	if mockUC.CapturedFilter.Providers[0] != "openalex" || mockUC.CapturedFilter.Providers[1] != "crossref" {
		t.Errorf("providers capturados incorretos: %v", mockUC.CapturedFilter.Providers)
	}
}

