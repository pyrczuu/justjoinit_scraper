package main

import (
	"context"
	"database/sql"
	"log"
	"sync"

	_ "github.com/mattn/go-sqlite3"
	"github.com/pfczx/jobscraper/iternal"
	"github.com/pfczx/jobscraper/iternal/scraper"
	"github.com/pfczx/jobscraper/iternal/scraper/scrapers"
)

func main() {
	db, err := sql.Open("sqlite3", "./database/jobs.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	ctx := context.Background()

	pracujUrls := []string{ //tutaj ma byc funkcja pracujUrls := urlgenerator.GenerateUrlsForPracuj()
		"https://www.pracuj.pl/praca/junior-fullstack-developer-java-%2b-angular-poznan-krysiewicza-9,oferta,1004441708?s=e2cceb02&searchId=MTc2MjExODYyODE5MS4xMzU3&ref=top_boosterAI_L0_4_1_1",
		"https://www.pracuj.pl/praca/senior-software-engineer-android-automotive-developer-platforms-krakow,oferta,1004410438?sug=list_top_cr_bd_1_tname_261_tgroup_B_boosterAI_L2&s=1f7c2c91&searchId=MTc2MjAyMTgyMzcyMC4wOTM1",
	}
	pracujScraper := scrapers.NewPracujScraper(pracujUrls)

	scrapersList := []scraper.Scraper{pracujScraper}

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		iternal.StartCollector(ctx, db, scrapersList)
	}()

	wg.Wait()
}
