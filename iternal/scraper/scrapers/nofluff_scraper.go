package scrapers

import (
	"context"
	"log"
	"strings"
	"time"

	"github.com/gocolly/colly"
	"github.com/pfczx/jobscraper/iternal/scraper"
)

const (
	titleSelector         = "h1.posting-details-description"
	companySelector       = "a#postingCompanyUrl"
	locationSelector      = "span[data-cy='location_pin']"
	descriptionSelector   = `section#posting-description`                            //concat in code
	skillsSelector        = `div#posting-requirements, section#posting-nice-to-have` //concat in code
	salarySectionSelector = `div.salary`
	salaryAmountSelector  = `div[data-test="text-earningAmount"]`
	contractTypeSelector  = `span[data-test="text-contractTypeName"]`
)

type NoFluffScraper struct {
	timeoutBetweenScraps time.Duration
	collector            *colly.Collector
	urls                 []string
}

// controls
func NewNoFLuffScraper(urls []string) *NoFluffScraper {
	c := colly.NewCollector(
		colly.AllowedDomains("www.nofluffjobs.com", "nofluffjobs.com"),
		//colly.Async(true),
	)

	// #nosec G104 - false positive i guess
	c.Limit(&colly.LimitRule{
		DomainGlob:  "*nofluffjobs.com*",
		Parallelism: 2,
		RandomDelay: 2 * time.Second,
	})

	return &NoFluffScraper{
		timeoutBetweenScraps: 10 * time.Second,
		collector:            c,
		urls:                 urls,
	}
}

func (*NoFluffScraper) Source() string {
	return "nofluffjobs.com"
}

func (p *NoFluffScraper) Scrape(ctx context.Context, q chan<- scraper.JobOffer) error {
	p.collector.UserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64)"

	p.collector.OnHTML("html", func(e *colly.HTMLElement) {
		select {
		case <-ctx.Done():
			return
		default:
		}

		var job scraper.JobOffer
		job.URL = e.Request.URL.String()
		job.Source = p.Source()
		job.Title = e.ChildText(titleSelector)
		job.Company = e.ChildText(companySelector)
		job.Location = e.ChildText(locationSelector)

		// description
		e.ForEach(descriptionSelector+" li", func(_ int, el *colly.HTMLElement) {
			if text := el.Text; text != "" {
				job.Description += text + "\n"
			}
		})

		// skills
		var skills []string
		e.ForEach(skillsSelector, func(_ int, el *colly.HTMLElement) {
			if text := el.Text; text != "" {
				skills = append(skills, text)
			}
		})
		job.Skills = skills

		// salary

		e.ForEach(salarySectionSelector, func(_ int, el *colly.HTMLElement) {
			sectionText := strings.TrimSpace(el.Text)
			parts := strings.Split(sectionText, "|")
			if len(parts) != 2 {
				return
			}
			amount := strings.TrimSpace(parts[0])
			ctype := strings.TrimSpace(parts[1])

			if ctype == "umowa o pracę" {
				job.SalaryEmployment = amount
			}
			if ctype == "umowa zlecenie" {
				job.SalaryContract = amount
			}
			if ctype == "kontrakt B2B" {
				job.SalaryB2B = amount
			}
		})

		select {
		case <-ctx.Done():
			return
		case q <- job:
		}
	})

	// Pętla po URL-ach
	for _, url := range p.urls {
		time.Sleep(p.timeoutBetweenScraps)
		log.Println("Waiting timeoutBetweenScraps")
		if err := p.collector.Visit(url); err != nil {
			log.Printf("Visit error: %v", err)
			return err
		}
	}

	p.collector.Wait()
	return nil
}
