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
	defer func(db *sql.DB) {
		err := db.Close()
		if err != nil {
			log.Fatal(err)
		}
	}(db)

	//read only for js backend
	//_, err = db.Exec("PRAGMA journal_mode=WAL;")

	ctx := context.Background()

	urls, err := scrapers.ScrollAndRead(ctx)
	if err != nil {
		log.Fatal(err)
	}
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
