package domain

type Article struct {
	ID             string   `json:"id"`
	Title          string   `json:"title"`
	Authors        []string `json:"authors"`
	Year           int      `json:"publication_year"`
	DOI            string   `json:"doi"`
	URL            string   `json:"url"`
	Journal        string   `json:"journal"`
	Citations      int      `json:"citation_count"`
	Abstract       string   `json:"abstract"`
	SourceProvider string   `json:"source_provider"`
}
