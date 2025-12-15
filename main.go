package main

import (
	"context"
	"github.com/pfczx/jobscraper/iternal/scraper/scrapers"
)

func main() {

	ctx := context.Background()
	_, err := scrapers.ScrollAndRead(ctx)
	if err != nil {
		return
	}
}
