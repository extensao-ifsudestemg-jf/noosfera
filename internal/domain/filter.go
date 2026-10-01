package domain

import "strings"

type SearchFilter struct {
	MinYear        int
	MaxYear        int
	MinCitations   int
	JournalKeyword string
	Providers      []string
}

func (f SearchFilter) Apply(articles []Article) []Article {
	result := make([]Article, 0)

	for _, article := range articles {
		if f.MinYear > 0 && article.Year < f.MinYear {
			continue
		}
		if f.MaxYear > 0 && article.Year > f.MaxYear {
			continue
		}
		if f.MinCitations > 0 && article.Citations < f.MinCitations {
			continue
		}
		if f.JournalKeyword != "" {
			if !strings.Contains(strings.ToLower(article.Journal), strings.ToLower(f.JournalKeyword)) {
				continue
			}
		}
		result = append(result, article)
	}

	return result
}
