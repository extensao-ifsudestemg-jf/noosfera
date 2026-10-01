package usecase

import (
	"context"
	"log"
	"strings"
	"sync"
	"time"
	"unicode"

	"noosfera/internal/domain"
)

type SearchResult struct {
	Articles        []domain.Article `json:"articles"`
	TotalFound      int              `json:"total_found"`
	ExecutionTimeMs int64            `json:"execution_time_ms"`
	SourceStats     map[string]int   `json:"source_stats"`
}

type SearchArticlesUseCase struct {
	providers []domain.ArticleProvider
}

func NewSearchArticlesUseCase(providers []domain.ArticleProvider) *SearchArticlesUseCase {
	return &SearchArticlesUseCase{
		providers: providers,
	}
}

type providerResult struct {
	providerIndex int
	sourceID      string
	articles      []domain.Article
	err           error
}

func normalizeDOI(rawDOI string) string {
	doi := strings.TrimSpace(strings.ToLower(rawDOI))
	if doi == "" {
		return ""
	}

	prefixes := []string{
		"https://doi.org/",
		"http://doi.org/",
		"https://dx.doi.org/",
		"http://dx.doi.org/",
		"doi.org/",
		"dx.doi.org/",
		"doi:",
	}
	for _, p := range prefixes {
		if strings.HasPrefix(doi, p) {
			doi = strings.TrimPrefix(doi, p)
			doi = strings.TrimSpace(doi)
		}
	}
	doi = strings.Trim(doi, "/")
	return strings.TrimSpace(doi)
}

func normalizeTitle(title string) string {
	title = strings.ToLower(title)
	var sb strings.Builder
	for _, r := range title {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			sb.WriteRune(r)
		} else if unicode.IsSpace(r) || unicode.IsPunct(r) || unicode.IsSymbol(r) {
			sb.WriteRune(' ')
		}
	}
	return strings.Join(strings.Fields(sb.String()), " ")
}

func combineSourceProviders(existing, incoming string) string {
	existing = strings.TrimSpace(existing)
	incoming = strings.TrimSpace(incoming)

	if existing == "" {
		return incoming
	}
	if incoming == "" {
		return existing
	}

	parts := strings.Split(existing, ",")
	seen := make(map[string]bool)
	var list []string
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" && !seen[trimmed] {
			seen[trimmed] = true
			list = append(list, trimmed)
		}
	}

	incomingParts := strings.Split(incoming, ",")
	for _, p := range incomingParts {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" && !seen[trimmed] {
			seen[trimmed] = true
			list = append(list, trimmed)
		}
	}

	return strings.Join(list, ", ")
}

func mergeArticle(target *domain.Article, incoming domain.Article) {
	if incoming.Citations > target.Citations {
		target.Citations = incoming.Citations
	}

	target.SourceProvider = combineSourceProviders(target.SourceProvider, incoming.SourceProvider)

	if target.DOI == "" && incoming.DOI != "" {
		target.DOI = incoming.DOI
	}
	if target.Title == "" && incoming.Title != "" {
		target.Title = incoming.Title
	}
	if len(target.Authors) == 0 && len(incoming.Authors) > 0 {
		target.Authors = incoming.Authors
	}
	if target.Year == 0 && incoming.Year != 0 {
		target.Year = incoming.Year
	}
	if target.URL == "" && incoming.URL != "" {
		target.URL = incoming.URL
	}
	if target.Journal == "" && incoming.Journal != "" {
		target.Journal = incoming.Journal
	}
	if target.Abstract == "" && incoming.Abstract != "" {
		target.Abstract = incoming.Abstract
	}
}

func findDuplicate(candidate domain.Article, existingList []domain.Article, doiMap, titleMap map[string]int) (bool, int) {
	normDOI := normalizeDOI(candidate.DOI)
	normTitle := normalizeTitle(candidate.Title)

	if normDOI != "" {
		if idx, found := doiMap[normDOI]; found {
			return true, idx
		}

		if normTitle != "" {
			if idx, found := titleMap[normTitle]; found {
				existingDOI := normalizeDOI(existingList[idx].DOI)
				if existingDOI == "" {
					return true, idx
				}
			}
		}
		return false, -1
	}

	if normTitle != "" {
		if idx, found := titleMap[normTitle]; found {
			return true, idx
		}
	}

	return false, -1
}

func (uc *SearchArticlesUseCase) Execute(ctx context.Context, query string, limit int, filter domain.SearchFilter) (*SearchResult, error) {
	startTime := time.Now()

	if limit <= 0 {
		limit = 20
	}

	numProviders := len(uc.providers)
	if numProviders == 0 {
		return &SearchResult{
			Articles:        make([]domain.Article, 0),
			TotalFound:      0,
			ExecutionTimeMs: time.Since(startTime).Milliseconds(),
			SourceStats:     make(map[string]int),
		}, nil
	}

	ch := make(chan providerResult, numProviders)
	var wg sync.WaitGroup

	for i, p := range uc.providers {
		wg.Add(1)
		go func(idx int, provider domain.ArticleProvider) {
			defer wg.Done()

			articles, err := provider.FetchArticles(ctx, query, limit)
			ch <- providerResult{
				providerIndex: idx,
				sourceID:      provider.GetSourceID(),
				articles:      articles,
				err:           err,
			}
		}(i, p)
	}

	go func() {
		wg.Wait()
		close(ch)
	}()

	providerBuckets := make([][]domain.Article, numProviders)
	for res := range ch {
		if res.err != nil {
			log.Printf("[UseCase] Provedor %s falhou: %v", res.sourceID, res.err)
			continue
		}

		for j := range res.articles {
			if res.articles[j].SourceProvider == "" {
				res.articles[j].SourceProvider = res.sourceID
			}
		}

		filtered := filter.Apply(res.articles)
		if filtered == nil {
			filtered = make([]domain.Article, 0)
		}
		providerBuckets[res.providerIndex] = filtered
	}

	finalArticles := make([]domain.Article, 0, limit)
	doiMap := make(map[string]int)
	titleMap := make(map[string]int)
	indices := make([]int, numProviders)
	countPerProvider := make([]int, numProviders)

	targetPerProvider := limit / numProviders
	if targetPerProvider < 1 {
		targetPerProvider = 1
	}

	for {
		anyAdded := false
		for i := 0; i < numProviders; i++ {
			if countPerProvider[i] >= targetPerProvider {
				continue
			}

			for indices[i] < len(providerBuckets[i]) {
				candidate := providerBuckets[i][indices[i]]
				indices[i]++

				isDup, dupIdx := findDuplicate(candidate, finalArticles, doiMap, titleMap)
				if isDup {
					mergeArticle(&finalArticles[dupIdx], candidate)
					if d := normalizeDOI(candidate.DOI); d != "" {
						doiMap[d] = dupIdx
					}
					if t := normalizeTitle(candidate.Title); t != "" {
						titleMap[t] = dupIdx
					}
					continue
				}

				idx := len(finalArticles)
				finalArticles = append(finalArticles, candidate)
				if d := normalizeDOI(candidate.DOI); d != "" {
					doiMap[d] = idx
				}
				if t := normalizeTitle(candidate.Title); t != "" {
					titleMap[t] = idx
				}

				countPerProvider[i]++
				anyAdded = true
				break
			}

			if len(finalArticles) >= limit {
				break
			}
		}

		if !anyAdded || len(finalArticles) >= limit {
			break
		}
	}

	for len(finalArticles) < limit {
		anyAdded := false
		for i := 0; i < numProviders; i++ {
			for indices[i] < len(providerBuckets[i]) {
				candidate := providerBuckets[i][indices[i]]
				indices[i]++

				isDup, dupIdx := findDuplicate(candidate, finalArticles, doiMap, titleMap)
				if isDup {
					mergeArticle(&finalArticles[dupIdx], candidate)
					if d := normalizeDOI(candidate.DOI); d != "" {
						doiMap[d] = dupIdx
					}
					if t := normalizeTitle(candidate.Title); t != "" {
						titleMap[t] = dupIdx
					}
					continue
				}

				idx := len(finalArticles)
				finalArticles = append(finalArticles, candidate)
				if d := normalizeDOI(candidate.DOI); d != "" {
					doiMap[d] = idx
				}
				if t := normalizeTitle(candidate.Title); t != "" {
					titleMap[t] = idx
				}

				countPerProvider[i]++
				anyAdded = true
				break
			}

			if len(finalArticles) >= limit {
				break
			}
		}

		if !anyAdded {
			break
		}
	}

	for i := 0; i < numProviders; i++ {
		for indices[i] < len(providerBuckets[i]) {
			candidate := providerBuckets[i][indices[i]]
			indices[i]++

			isDup, dupIdx := findDuplicate(candidate, finalArticles, doiMap, titleMap)
			if isDup {
				mergeArticle(&finalArticles[dupIdx], candidate)
				if d := normalizeDOI(candidate.DOI); d != "" {
					doiMap[d] = dupIdx
				}
				if t := normalizeTitle(candidate.Title); t != "" {
					titleMap[t] = dupIdx
				}
			}
		}
	}

	sourceStats := make(map[string]int)
	for _, p := range uc.providers {
		sourceStats[p.GetSourceID()] = 0
	}
	for _, article := range finalArticles {
		for _, src := range strings.Split(article.SourceProvider, ",") {
			trimmed := strings.TrimSpace(src)
			if trimmed != "" {
				sourceStats[trimmed]++
			}
		}
	}

	executionTimeMs := time.Since(startTime).Milliseconds()

	return &SearchResult{
		Articles:        finalArticles,
		TotalFound:      len(finalArticles),
		ExecutionTimeMs: executionTimeMs,
		SourceStats:     sourceStats,
	}, nil
}
