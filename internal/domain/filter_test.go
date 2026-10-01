package domain

import (
	"reflect"
	"testing"
)

func TestSearchFilter_Apply(t *testing.T) {

	mockArticles := []Article{
		{
			Year:      2019,
			Citations: 5,
			Journal:   "IEEE Transactions on Software Engineering",
		},
		{
			Year:      2021,
			Citations: 50,
			Journal:   "Nature Machine Intelligence",
		},
		{
			Year:      2024,
			Citations: 150,
			Journal:   "ACM Computing Surveys",
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
}
