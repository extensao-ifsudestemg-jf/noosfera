package domain

import (
	"reflect"
	"testing"
)

func TestSearchFilter_Apply(t *testing.T) {
	trueVal := true
	falseVal := false

	mockArticles := []Article{
		{
			Year:         2019,
			Citations:    5,
			Journal:      "IEEE Transactions on Software Engineering",
			IsOpenAccess: &falseVal,
			DocType:      "journal-article",
			Language:     "en",
		},
		{
			Year:         2021,
			Citations:    50,
			Journal:      "Nature Machine Intelligence",
			IsOpenAccess: &trueVal,
			DocType:      "review",
			Language:     "en",
		},
		{
			Year:         2024,
			Citations:    150,
			Journal:      "ACM Computing Surveys",
			IsOpenAccess: &trueVal,
			DocType:      "book-chapter",
			Language:     "pt",
		},
	}

	tests := []struct {
		name     string
		filter   SearchFilter
		expected []Article
	}{
		{
			name:     "Filtro neutro (SearchFilter vazia) - Retornando 100% dos artigos",
			filter:   SearchFilter{},
			expected: mockArticles,
		},
		{
			name: "[CT-U01] Filtro por intervalo de anos (MinYear=2020, MaxYear=2023)",
			filter: SearchFilter{
				MinYear: 2020,
				MaxYear: 2023,
			},
			expected: []Article{
				mockArticles[1],
			},
		},
		{
			name: "[CT-U02] Filtro por número mínimo de citações (MinCitations=10)",
			filter: SearchFilter{
				MinCitations: 10,
			},
			expected: []Article{
				mockArticles[1],
				mockArticles[2],
			},
		},
		{
			name: "Filtro textual por palavra-chave de periódico (JournalKeyword case-insensitive)",
			filter: SearchFilter{
				JournalKeyword: "nature",
			},
			expected: []Article{
				mockArticles[1],
			},
		},
		{
			name: "Filtro textual por palavra-chave de periódico (JournalKeyword uppercase)",
			filter: SearchFilter{
				JournalKeyword: "COMPUTING",
			},
			expected: []Article{
				mockArticles[2],
			},
		},
		{
			name: "Filtros combinados (MinYear, MinCitations e JournalKeyword)",
			filter: SearchFilter{
				MinYear:        2020,
				MinCitations:   100,
				JournalKeyword: "acm",
			},
			expected: []Article{
				mockArticles[2],
			},
		},
		{
			name: "Filtro sem resultados correspondentes",
			filter: SearchFilter{
				MinYear: 2025,
			},
			expected: []Article{},
		},
		{
			name: "Filtro por Acesso Aberto (IsOpenAccess = true)",
			filter: SearchFilter{
				IsOpenAccess: &trueVal,
			},
			expected: []Article{
				mockArticles[1],
				mockArticles[2],
			},
		},
		{
			name: "Filtro por Acesso Aberto (IsOpenAccess = false)",
			filter: SearchFilter{
				IsOpenAccess: &falseVal,
			},
			expected: []Article{
				mockArticles[0],
			},
		},
		{
			name: "Filtro por DocType específico (review)",
			filter: SearchFilter{
				DocType: "review",
			},
			expected: []Article{
				mockArticles[1],
			},
		},
		{
			name: "Filtro por DocType normalizado (article)",
			filter: SearchFilter{
				DocType: "article",
			},
			expected: []Article{
				mockArticles[0],
			},
		},
		{
			name: "Filtro por Idioma (Language = pt)",
			filter: SearchFilter{
				Language: "pt",
			},
			expected: []Article{
				mockArticles[2],
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.filter.Apply(mockArticles)

			if len(result) == 0 && len(tt.expected) == 0 {
				return
			}

			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("SearchFilter.Apply() = %v, esperado = %v", result, tt.expected)
			}
		})
	}

	t.Run("Filtro por Idioma descarta estritamente artigos sem idioma ou diferentes", func(t *testing.T) {
		articles := []Article{
			{ID: "1", Language: "pt"},
			{ID: "2", Language: "en"},
			{ID: "3", Language: ""},
			{ID: "4", Language: "   "},
		}

		filter := SearchFilter{Language: "pt"}
		got := filter.Apply(articles)

		if len(got) != 1 || got[0].ID != "1" {
			t.Errorf("esperado apenas o artigo com Language='pt', obtido %v", got)
		}
	})
}
