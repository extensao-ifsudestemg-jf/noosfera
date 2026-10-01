package domain

import "context"

type ArticleProvider interface {
	FetchArticles(ctx context.Context, query string, limit int) ([]Article, error)

	GetSourceID() string
}
