package scrapers

import (
	"bufio"
	"context"
	"log"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/chromedp/cdproto/emulation"
	"github.com/chromedp/chromedp"
	"github.com/pfczx/jobscraper/iternal/scraper"
)

var proxyList = []string{
	"213.73.25.231:8080",
}

// selectors
const (
	titleSelector       = "div[class*=\"MuiStack-root\"] > h1"
	companySelector     = "h2:has(svg[data-testid=\"ApartmentRoundedIcon\"])"
	locationSelector    = "div.MuiBox-root.mui-1jfrpka"
	workTypeSelector    = "MuiStack-root.mui-aa3a55"
	descriptionSelector = "h3 + div[class*=\"MuiBox-root\"]"
	techSelector        = "h4[aria-label]"
)

// wait times are random (min,max) in seconds
type JustJoinItScraper struct {
	minTimeS int
	maxTimeS int
	urls     []string
}

func NewJustJoinItScraper(urls []string) *JustJoinItScraper {
	return &JustJoinItScraper{
		minTimeS: 5,
		maxTimeS: 10,
		urls:     urls,
	}
}

func (*JustJoinItScraper) Source() string {
	return "https://justjoin.it/"
}

func waitForCaptcha() {
	log.Println("Cloudflare detected, solve and press enter")
	reader := bufio.NewReader(os.Stdin)
	reader.ReadBytes('\n')
}

// extracting data from string html with goquer selectors
func (p *JustJoinItScraper) extractDataFromHTML(html string, url string) (scraper.JobOffer, error, bool) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		log.Printf("goquery parse error: %v", err)
		return scraper.JobOffer{}, err, false
	}

	if strings.Contains(html, "Verifying you are human") {
		waitForCaptcha()
		return scraper.JobOffer{}, nil, true
	}

	var job scraper.JobOffer
	job.URL = url
	job.Source = p.Source()
	job.Title = strings.TrimSpace(doc.Find(titleSelector).Text())

	company := strings.TrimSpace(doc.Find(companySelector).Text())
	unwantedDetails := []string{
		"O firmie",
		"About company",
		"About the company",
	}

	for _, u := range unwantedDetails {
		company = strings.TrimSuffix(company, u)
	}

	job.Company = strings.TrimSpace(company)

	rawLocation := strings.TrimSpace(doc.Find(locationSelector).First().Text())

	location := rawLocation
	parts := strings.Split(rawLocation, "+")
	if len(parts) > 1 {
		location = strings.TrimSpace(parts[1])
	}

	workType := strings.TrimSpace(doc.Find(workTypeSelector).Text())
	if len(workType) > 0 {
		location += ", " + workType
	}

	job.Location = location

	var htmlBuilder strings.Builder

	//description
	descText := strings.TrimSpace(doc.Find(descriptionSelector).Text())
	if descText != "" {
		htmlBuilder.WriteString("<p>" + descText + "</p>\n")
	}
	job.Description = htmlBuilder.String()

	// skills / tech stack
	var skills []string
	doc.Find(techSelector).Each(func(_ int, s *goquery.Selection) {
		name := s.Text()

		skills = append(skills, name)
		job.Skills = skills
	})

	doc.Find("div[class*='MuiStack-root']").Has("div[class*='MuiTypography-h4']").Each(func(i int, s *goquery.Selection) {

		rawAmount := strings.TrimSpace(s.Find("div[class*='MuiTypography-h4']").Text())

		lowerDesc := strings.ToLower(s.Find("span[class*='MuiTypography-subtitle4']").Text())

		fullInfo := rawAmount + ", " + lowerDesc

		switch {
		case strings.Contains(lowerDesc, "permanent") || strings.Contains(lowerDesc, "employment"):
			job.SalaryEmployment = fullInfo

		case strings.Contains(lowerDesc, "mandate") || strings.Contains(lowerDesc, "specific-task"):
			job.SalaryContract = fullInfo

		case strings.Contains(lowerDesc, "b2b"):
			job.SalaryB2B = fullInfo

		case strings.Contains(lowerDesc, "any"):
			job.SalaryEmployment = fullInfo
			job.SalaryContract = fullInfo
			job.SalaryB2B = fullInfo

		}

	})

	return job, nil, false
}

// html chromedp
func (p *JustJoinItScraper) getHTMLContent(chromeDpCtx context.Context, url string) (string, error) {
	var html string

	//chromdp run config
	err := chromedp.Run(
		chromeDpCtx,
		chromedp.ActionFunc(func(ctx context.Context) error {
			return emulation.SetDeviceMetricsOverride(1280, 900, 1.0, false).Do(ctx)
		}),
		chromedp.Navigate(url),
		chromedp.Evaluate(`delete navigator.__proto__.webdriver`, nil),
		chromedp.Evaluate(`Object.defineProperty(navigator, "webdriver", { get: () => false })`, nil),
		chromedp.Sleep(time.Duration(rand.Intn(800)+300)*time.Millisecond),
		chromedp.WaitVisible("body", chromedp.ByQuery),
		chromedp.OuterHTML("html", &html),
	)
	return html, err
}

// main func for scraping
func (p *JustJoinItScraper) Scrape(ctx context.Context, q chan<- scraper.JobOffer) error {

	//chromdp config
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.ExecPath("/usr/bin/google-chrome"),
		chromedp.UserDataDir(browserDataDir),
		chromedp.Flag("disable-blink-features", "AutomationControlled"),
		chromedp.Flag("headless", false),
		chromedp.Flag("disable-gpu", false),
		chromedp.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) "+
			"AppleWebKit/537.36 (KHTML, like Gecko) Chrome/119.0.0.0 Safari/537.36"),
		//chromedp.Flag("proxy-server", proxyList[rand.Intn(len(proxyList))]),
		chromedp.Flag("disable-web-security", true),
		chromedp.Flag("disable-site-isolation-trials", true),
	)
	allocCtx, cancelAlloc := chromedp.NewExecAllocator(ctx, opts...)
	defer cancelAlloc()

	chromeDpCtx, cancelCtx := chromedp.NewContext(allocCtx)
	defer cancelCtx()

	for i := 0; i < len(p.urls); i++ {
		url := p.urls[i]
		html, err := p.getHTMLContent(chromeDpCtx, url)
		if err != nil {
			log.Printf("Chromedp error: %v", err)
			continue
		}

		job, err, captchaAppeared := p.extractDataFromHTML(html, url)
		if captchaAppeared == true {
			time.Sleep(5 * time.Second)
			i--
			continue
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case q <- job:
		}

		log.Printf("Scraped %d: %s", i+1, url)
		randomDelay := rand.Intn(p.maxTimeS-p.minTimeS) + p.minTimeS
		log.Printf("Sleeping for: %ds", randomDelay)
		time.Sleep(time.Duration(randomDelay) * time.Second)
	}

	return nil
}
