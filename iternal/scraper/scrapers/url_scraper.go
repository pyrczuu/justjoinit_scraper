package scrapers

import (
	"context"
	"log"
	"math/rand"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/emulation"
	"github.com/chromedp/chromedp"
)

const (
	browserDataDir = `~/.config/google-chrome/Default`
	source         = "https://nofluffjobs.com/pl/artificial-intelligence?criteria=category%3Dsys-administrator,business-analyst,architecture,backend,data,ux,devops,erp,embedded,frontend,fullstack,game-dev,mobile,project-manager,security,support,testing,other"
	// tylko do testow
	//source                = "https://nofluffjobs.com/pl/Golang"
	minTimeMs             = 3000
	maxTimeMs             = 4000
	prefix                = "https://nofluffjobs.com/pl/job/"
	offerSelector         = "a.posting-list-item"
	cookiesButtonSelector = "button#save"                                // zamknięcie cookies
	loginButtonSelector   = "button[.//inline-icon[@maticon=\"close\"]]" // zamknięcie prośby o zalogowanie
	loadMoreSelector      = "button[nfjloadmore]"
)

func getUrlsFromContent(html string) ([]string, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		log.Printf("goquery parse error: %v", err)
		return nil, err
	}

	var urls []string

	doc.Find(offerSelector).Each(func(_ int, s *goquery.Selection) {
		href, exists := s.Attr("href")
		if exists {
			urls = append(urls, prefix+href)
		}
	})

	return urls, nil
}

func ScrollAndRead(parentCtx context.Context) ([]string, error) {
	var urls []string

	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.ExecPath("/usr/bin/google-chrome"),
		chromedp.UserDataDir(browserDataDir),
		chromedp.Flag("disable-blink-features", "AutomationControlled"),
		chromedp.Flag("headless", false),
		chromedp.Flag("disable-gpu", false),
		chromedp.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/119.0.0.0 Safari/537.36"),
		chromedp.Flag("disable-web-security", true),
	)

	allocCtx, cancelAlloc := chromedp.NewExecAllocator(parentCtx, opts...)
	defer cancelAlloc()

	chromeDpCtx, cancelCtx := chromedp.NewContext(allocCtx)
	defer cancelCtx()

	log.Println("Uruchamianie przeglądarki...")

	var html string

	err := chromedp.Run(chromeDpCtx,

		chromedp.ActionFunc(func(ctx context.Context) error {
			return emulation.SetDeviceMetricsOverride(1280, 900, 1.0, false).Do(ctx)
		}),
		chromedp.Navigate(source),
		chromedp.Evaluate(`delete navigator.__proto__.webdriver`, nil),
		chromedp.WaitVisible(`body`, chromedp.ByQuery),
		//klika wymagane cookies jeśli jest komunikat, blokuje program jeśli ich nie ma :/
		//chromedp.Click(
		//	cookiesButtonSelector,
		//	chromedp.NodeVisible,
		//),
		//chromedp.Click(
		//	loginButtonSelector,
		//	chromedp.NodeVisible,
		//),
		chromedp.ActionFunc(func(ctx context.Context) error {
			log.Println("Strona załadowana. Rozpoczynanie pętli wewnętrznej...")
			var nodes []*cdp.Node

			for i := 1; ; i++ {
				log.Printf("Iteracja: %v", i)
				randomDelay := rand.Intn(maxTimeMs-minTimeMs) + minTimeMs
				err := chromedp.Sleep(time.Duration(randomDelay) * time.Millisecond).Do(ctx)
				if err != nil {
					return err
				}

				err = chromedp.Nodes(loadMoreSelector, &nodes, chromedp.AtLeast(0)).Do(ctx)
				if err != nil {
					return err
				}
				log.Println(nodes)
				if len(nodes) == 0 {
					break
				}

				err = chromedp.Click(loadMoreSelector).Do(ctx)
				if err != nil {
					return err
				}

				randomDelay = rand.Intn((maxTimeMs+10*i)-(minTimeMs+10*i)) + (minTimeMs + 10*i) // czym więcej kontentu (kolejne iteracje) tym dłużej czekamy (wolniejsza strona)
				err = chromedp.Sleep(time.Duration(randomDelay) * time.Millisecond).Do(ctx)
				if err != nil {
					return err
				}
			}
			return nil
		}),
		chromedp.OuterHTML("html", &html),
	)
	urls, err = getUrlsFromContent(html)
	log.Printf("Znaleziono %v linków", len(urls))
	if err != nil {
		log.Println("Błąd wyciąganie url z kontentu")
		return nil, err
	}

	return urls, nil
}
