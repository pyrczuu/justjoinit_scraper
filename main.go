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

	//read only for js backend
	//_, err = db.Exec("PRAGMA journal_mode=WAL;")

	ctx := context.Background()

	//urls, err := scrapers.ScrollAndRead(ctx)
	//if err != nil {
	//	log.Fatal(err)
	//}
	urls := []string{
		"https://justjoin.it/job-offer/as-inbank-spolka-akcyjna-oddzial-w-polsce-senior-product-engineer-gdansk-java-ad38b4c9",
		"https://justjoin.it/job-offer/ai-clearing-senior-ai-engineer-cv--warszawa-ai",
		"https://justjoin.it/job-offer/szkola-w-chmurze-it-helpdesk-warszawa-support",
		"https://justjoin.it/job-offer/in4ge-sp-z-o-o--junior-java-developer-warszawa-java"}
	justJoinItScraper := scrapers.NewJustJoinItScraper(urls)

	scrapersList := []scraper.Scraper{justJoinItScraper}

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		iternal.StartCollector(ctx, db, scrapersList)
	}()

	wg.Wait()
	log.Println("-------------------")
}
