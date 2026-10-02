package domain

import "strings"

type SearchFilter struct {
	MinYear        int
	MaxYear        int
	MinCitations   int
	JournalKeyword string
	Providers      []string
	IsOpenAccess   *bool
	DocType        string
	Language       string
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
		if f.IsOpenAccess != nil {
			if article.IsOpenAccess == nil || *article.IsOpenAccess != *f.IsOpenAccess {
				continue
			}
		}
		if f.DocType != "" {
			if !matchDocType(article.DocType, f.DocType) {
				continue
			}
		}
		if f.Language != "" {
			if !strings.EqualFold(strings.TrimSpace(article.Language), strings.TrimSpace(f.Language)) {
				continue
			}
		}
		result = append(result, article)
	}

	return result
}

func matchDocType(itemType, filterType string) bool {
	if filterType == "" {
		return true
	}
	item := strings.ToLower(strings.TrimSpace(itemType))
	filter := strings.ToLower(strings.TrimSpace(filterType))
	if item == "" {
		return true
	}
	if item == filter {
		return true
	}
	switch filter {
	case "article", "artigo":
		return item == "article" || item == "journal-article"
	case "review", "revisão":
		return item == "review" || strings.Contains(item, "review")
	case "book-chapter", "capítulo de livro", "chapter":
		return item == "book-chapter" || item == "book_chapter" || item == "chapter"
	case "conference", "conferência", "proceedings":
		return item == "proceedings-article" || item == "proceedings" || item == "conference"
	}
	return strings.Contains(item, filter)
}
