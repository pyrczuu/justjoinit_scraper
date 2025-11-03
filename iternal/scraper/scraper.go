package scraper

import (
	"context"
	"log"
	"sync"
)

type JobOffer struct {
	ID               string   `json:"id"`
	Title            string   `json:"title"`
	Company          string   `json:"company"`
	Location         string   `json:"location"`
	SalaryEmployment string   `json:"salary_employment"`
	SalaryContract   string   `json:"salary_contract"`
	SalaryB2B        string   `json:"salary_b2b"`
	Description      string   `json:"description"`
	URL              string   `json:"url"`
	Source           string   `json:"source"`
	PublishedAt      *string  `json:"published_at,omitempty"` //potencial problems
	Skills           []string `json:"skills,omitempty"`
}

type Scraper interface {
	Source() string
	Scrape(ctx context.Context, q chan<- JobOffer) error
}

func RunScrapers(ctx context.Context, scrapers []Scraper) chan JobOffer {
	out := make(chan JobOffer)
	var wg sync.WaitGroup

	for _, s := range scrapers {
		wg.Add(1)
		go func(scr Scraper) {
			defer wg.Done()
			log.Printf("Starting scraper: %s", scr.Source())
			if err := scr.Scrape(ctx, out); err != nil {
				log.Printf("Error in scraper %s: %v", scr.Source(), err)
			}
			log.Printf("Finished scraper: %s", scr.Source())
		}(s)
	}

	go func() {
		wg.Wait()
		close(out)
	}()

	return out
}
