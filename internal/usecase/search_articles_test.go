package usecase_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"noosfera/internal/domain"
	"noosfera/internal/usecase"
)

type MockProvider struct {
	SourceID string
	Articles []domain.Article
	Err      error
	Delay    time.Duration
}

func (m *MockProvider) FetchArticles(ctx context.Context, query string, limit int) ([]domain.Article, error) {
	if m.Delay > 0 {
		timer := time.NewTimer(m.Delay)
		defer timer.Stop()
		select {
		case <-timer.C:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	} else {

		if err := ctx.Err(); err != nil {
			return nil, err
		}
	}

	if m.Err != nil {
		return nil, m.Err
	}

	return m.Articles, nil
}

func (m *MockProvider) GetSourceID() string {
	return m.SourceID
}

func TestSearchArticlesUseCase_RaceCondition(t *testing.T) {

	providers := []domain.ArticleProvider{
		&MockProvider{SourceID: "P1", Articles: []domain.Article{{ID: "1"}}, Delay: 10 * time.Millisecond},
		&MockProvider{SourceID: "P2", Articles: []domain.Article{{ID: "2"}}, Delay: 5 * time.Millisecond},
		&MockProvider{SourceID: "P3", Articles: []domain.Article{{ID: "3"}}, Delay: 15 * time.Millisecond},
		&MockProvider{SourceID: "P4", Articles: []domain.Article{{ID: "4"}}, Delay: 8 * time.Millisecond},
	}

	uc := usecase.NewSearchArticlesUseCase(providers)
	ctx := context.Background()
	filter := domain.SearchFilter{}

	res, err := uc.Execute(ctx, "race test", 10, filter)

	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}

	if len(res.Articles) != 4 {
		t.Errorf("esperado 4 artigos, obtido %d", len(res.Articles))
	}

	if res.TotalFound != 4 {
		t.Errorf("esperado TotalFound == 4, obtido %d", res.TotalFound)
	}

	if len(res.SourceStats) != 4 {
		t.Errorf("esperado 4 entradas em SourceStats, obtido %d", len(res.SourceStats))
	}
}

func TestSearchArticlesUseCase_TimeoutResilience(t *testing.T) {

	providerA := &MockProvider{
		SourceID: "FastProvider",
		Articles: []domain.Article{{ID: "A1"}},
		Delay:    50 * time.Millisecond,
	}

	providerB := &MockProvider{
		SourceID: "SlowProvider",
		Articles: []domain.Article{{ID: "B1"}},
		Delay:    500 * time.Millisecond,
	}

	uc := usecase.NewSearchArticlesUseCase([]domain.ArticleProvider{providerA, providerB})

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	filter := domain.SearchFilter{}

	start := time.Now()
	res, err := uc.Execute(ctx, "timeout test", 10, filter)
	duration := time.Since(start)

	if err != nil {
		t.Fatalf("erro inesperado, orquestrador deveria retornar resultados parciais sem falhar. Erro: %v", err)
	}

	if len(res.Articles) != 1 || res.Articles[0].ID != "A1" {
		t.Errorf("esperado 1 artigo parcial do Provedor A, obtido: %v", res.Articles)
	}

	if res.TotalFound != 1 {
		t.Errorf("esperado TotalFound == 1, obtido %d", res.TotalFound)
	}

	if duration >= 400*time.Millisecond {
		t.Errorf("a execução demorou muito (%v), esperado próximo de 100ms", duration)
	}
}

func TestSearchArticlesUseCase_FilterApplication(t *testing.T) {
	providerA := &MockProvider{
		SourceID: "ProvA",
		Articles: []domain.Article{
			{ID: "A1", Year: 2020, Citations: 5, Journal: "Nature"},
			{ID: "A2", Year: 2022, Citations: 20, Journal: "Science"},
		},
	}

	providerB := &MockProvider{
		SourceID: "ProvB",
		Articles: []domain.Article{
			{ID: "B1", Year: 2023, Citations: 50, Journal: "Nature Physics"},
			{ID: "B2", Year: 2018, Citations: 100, Journal: "Cell"},
		},
	}

	uc := usecase.NewSearchArticlesUseCase([]domain.ArticleProvider{providerA, providerB})

	filter := domain.SearchFilter{
		MinYear: 2021,
	}

	ctx := context.Background()
	res, err := uc.Execute(ctx, "filter test", 10, filter)

	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}

	if len(res.Articles) != 2 {
		t.Fatalf("esperado 2 artigos após o filtro, obtido %d", len(res.Articles))
	}

	if res.TotalFound != 2 {
		t.Fatalf("esperado TotalFound == 2, obtido %d", res.TotalFound)
	}

	for _, a := range res.Articles {
		if a.Year < 2021 {
			t.Errorf("artigo %s possui ano %d, que não deveria passar no filtro (MinYear=2021)", a.ID, a.Year)
		}
	}
}

func TestSearchArticlesUseCase_ProviderErrorResilience(t *testing.T) {
	providerSuccess := &MockProvider{
		SourceID: "SuccessProvider",
		Articles: []domain.Article{{ID: "S1"}},
	}
	providerFail := &MockProvider{
		SourceID: "FailingProvider",
		Err:      errors.New("api rate limit exceeded"),
	}

	uc := usecase.NewSearchArticlesUseCase([]domain.ArticleProvider{providerSuccess, providerFail})
	res, err := uc.Execute(context.Background(), "error resilience test", 10, domain.SearchFilter{})

	if err != nil {
		t.Fatalf("erro inesperado ao tratar falha parcial: %v", err)
	}

	if len(res.Articles) != 1 || res.Articles[0].ID != "S1" {
		t.Errorf("esperado 1 artigo retornado do provedor de sucesso, obtido: %v", res.Articles)
	}

	if res.SourceStats["SuccessProvider"] != 1 {
		t.Errorf("esperado SourceStats['SuccessProvider'] == 1, obtido %d", res.SourceStats["SuccessProvider"])
	}
}

func TestSearchArticlesUseCase_DeduplicationAndInterleaving(t *testing.T) {
	t.Run("Desduplicação por DOI e Intercalação Round-Robin", func(t *testing.T) {
		mockOpenAlex := &MockProvider{
			SourceID: "OpenAlex",
			Articles: []domain.Article{
				{
					ID:             "OA-1",
					Title:          "Quantum Computing Advances",
					DOI:            "https://doi.org/10.1000/182",
					Citations:      15,
					SourceProvider: "OpenAlex",
				},
				{
					ID:             "OA-2",
					Title:          "Machine Learning in Bioinformatics",
					DOI:            "10.2000/ml.bio",
					Citations:      30,
					SourceProvider: "OpenAlex",
				},
				{
					ID:             "OA-3",
					Title:          "Distributed Systems at Scale",
					DOI:            "10.3000/dist.sys",
					Citations:      5,
					SourceProvider: "OpenAlex",
				},
			},
		}

		mockCrossRef := &MockProvider{
			SourceID: "CrossRef",
			Articles: []domain.Article{
				{
					ID:             "CR-1",
					Title:          "Quantum Computing Advances!",
					DOI:            "10.1000/182",
					Citations:      45,
					SourceProvider: "CrossRef",
				},
				{
					ID:             "CR-2",
					Title:          "Graph Neural Networks",
					DOI:            "10.4000/gnn",
					Citations:      60,
					SourceProvider: "CrossRef",
				},
				{
					ID:             "CR-3",
					Title:          "Cybersecurity in Cloud Environments",
					DOI:            "10.5000/cloud.sec",
					Citations:      12,
					SourceProvider: "CrossRef",
				},
			},
		}

		uc := usecase.NewSearchArticlesUseCase([]domain.ArticleProvider{mockOpenAlex, mockCrossRef})
		ctx := context.Background()

		res, err := uc.Execute(ctx, "computing", 10, domain.SearchFilter{})
		if err != nil {
			t.Fatalf("erro inesperado na busca: %v", err)
		}

		if len(res.Articles) != 5 {
			t.Fatalf("esperado 5 artigos após desduplicação, obtido %d", len(res.Articles))
		}
		if res.TotalFound != 5 {
			t.Errorf("esperado TotalFound == 5, obtido %d", res.TotalFound)
		}

		firstArticle := res.Articles[0]
		if firstArticle.Citations != 45 {
			t.Errorf("esperado maior número de citações (45), obtido %d", firstArticle.Citations)
		}
		if !strings.Contains(firstArticle.SourceProvider, "OpenAlex") || !strings.Contains(firstArticle.SourceProvider, "CrossRef") {
			t.Errorf("esperado proveniência combinada contendo OpenAlex e CrossRef, obtido '%s'", firstArticle.SourceProvider)
		}

		expectedOrder := []struct {
			expectedID     string
			expectedSource string
		}{
			{"OA-1", "OpenAlex, CrossRef"},
			{"CR-2", "CrossRef"},
			{"OA-2", "OpenAlex"},
			{"CR-3", "CrossRef"},
			{"OA-3", "OpenAlex"},
		}

		for i, exp := range expectedOrder {
			if res.Articles[i].ID != exp.expectedID {
				t.Errorf("artigo na posição %d: esperado ID '%s', obtido '%s'", i, exp.expectedID, res.Articles[i].ID)
			}
			if res.Articles[i].SourceProvider != exp.expectedSource {
				t.Errorf("artigo na posição %d: esperado SourceProvider '%s', obtido '%s'", i, exp.expectedSource, res.Articles[i].SourceProvider)
			}
		}

		if res.SourceStats["OpenAlex"] != 3 {
			t.Errorf("esperado SourceStats['OpenAlex'] == 3, obtido %d", res.SourceStats["OpenAlex"])
		}
		if res.SourceStats["CrossRef"] != 3 {
			t.Errorf("esperado SourceStats['CrossRef'] == 3, obtido %d", res.SourceStats["CrossRef"])
		}
	})

	t.Run("Desduplicação Secundária por Título quando DOI ausente", func(t *testing.T) {
		mock1 := &MockProvider{
			SourceID: "P1",
			Articles: []domain.Article{
				{
					ID:             "P1-1",
					Title:          "Concurrency Patterns in Go: A Deep Dive",
					Citations:      10,
					SourceProvider: "P1",
				},
			},
		}

		mock2 := &MockProvider{
			SourceID: "P2",
			Articles: []domain.Article{
				{
					ID:             "P2-1",
					Title:          "concurrency patterns in go a deep dive.",
					Citations:      50,
					SourceProvider: "P2",
				},
			},
		}

		uc := usecase.NewSearchArticlesUseCase([]domain.ArticleProvider{mock1, mock2})
		res, err := uc.Execute(context.Background(), "concurrency", 10, domain.SearchFilter{})
		if err != nil {
			t.Fatalf("erro inesperado: %v", err)
		}

		if len(res.Articles) != 1 {
			t.Fatalf("esperado 1 artigo unificado por título, obtido %d", len(res.Articles))
		}
		if res.Articles[0].Citations != 50 {
			t.Errorf("esperado 50 citações, obtido %d", res.Articles[0].Citations)
		}
		if res.Articles[0].SourceProvider != "P1, P2" {
			t.Errorf("esperado proveniência 'P1, P2', obtido '%s'", res.Articles[0].SourceProvider)
		}
	})

	t.Run("Esgotamento de Fonte Consome Restante Até Limite", func(t *testing.T) {
		mockShort := &MockProvider{
			SourceID: "Short",
			Articles: []domain.Article{
				{ID: "S1", Title: "Short 1"},
			},
		}

		mockLong := &MockProvider{
			SourceID: "Long",
			Articles: []domain.Article{
				{ID: "L1", Title: "Long 1"},
				{ID: "L2", Title: "Long 2"},
				{ID: "L3", Title: "Long 3"},
				{ID: "L4", Title: "Long 4"},
			},
		}

		uc := usecase.NewSearchArticlesUseCase([]domain.ArticleProvider{mockShort, mockLong})
		res, err := uc.Execute(context.Background(), "test", 4, domain.SearchFilter{})
		if err != nil {
			t.Fatalf("erro inesperado: %v", err)
		}

		if len(res.Articles) != 4 {
			t.Fatalf("esperado 4 artigos respeitando o limite, obtido %d", len(res.Articles))
		}

		expectedIDs := []string{"S1", "L1", "L2", "L3"}
		for i, expID := range expectedIDs {
			if res.Articles[i].ID != expID {
				t.Errorf("posição %d: esperado ID '%s', obtido '%s'", i, expID, res.Articles[i].ID)
			}
		}
	})
}
