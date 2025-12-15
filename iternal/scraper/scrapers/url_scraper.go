package scrapers

import (
	"context"
	"fmt"

	"log"
	"math/rand"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/chromedp/cdproto/emulation"
	"github.com/chromedp/chromedp"
	//"github.com/chromedp/chromedp/kb"
)

const (
	browserDataDir        = `~/.config/google-chrome/Default`
	source                = "https://nofluffjobs.com/pl"
	minTimeMs             = 3000
	maxTimeMs             = 4000
	prefix                = "https://nofluffjobs.com/pl/job/"
	offerSelector         = "a.posting-list-item"
	minDelayButton        = 500
	maxDelayButton        = 1500
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
		//klika wymagane cookies jeśli jest komunikat
		//chromedp.Click(
		//	cookiesButtonSelector,
		//	chromedp.NodeVisible,
		//),
		//chromedp.Click(
		//	loginButtonSelector,
		//	chromedp.NodeVisible,
		//),
		chromedp.ActionFunc(func(ctx context.Context) error {
			var prevHeight int64 = -99
			var currentHeight int64

			log.Println("Strona załadowana. Rozpoczynanie pętli wewnętrznej...")

			for i := 1; ; i++ {
				err := chromedp.Evaluate(`document.body.scrollHeight`, &currentHeight).Do(ctx)
				if err != nil {
					return fmt.Errorf("błąd pobierania wysokości: %w", err)
				}

				if currentHeight == prevHeight {
					log.Printf("KONIEC: Wysokość stała (%d).", currentHeight)
					break
				}

				prevHeight = currentHeight
				log.Printf("Iteracja: %v", i)
				log.Printf("Scrollowanie do: %d", currentHeight)

				randomDelay := rand.Intn(maxTimeMs-minTimeMs) + minTimeMs
				err = chromedp.Sleep(time.Duration(randomDelay) * time.Millisecond).Do(ctx)
				if err != nil {
					return err
				}

				err = chromedp.Click(loadMoreSelector,
					chromedp.NodeVisible,
				).Do(ctx)
				if err != nil {
					return err
				}

				randomDelay = rand.Intn(maxTimeMs-minTimeMs) + minTimeMs
				err = chromedp.Sleep(time.Duration(randomDelay) * time.Millisecond).Do(ctx)
				if err != nil {
					return err
				}
			}
			return nil
		}),
		chromedp.OuterHTML("html", &html),
	)
	log.Println(html[:100])
	urls, err = getUrlsFromContent(html)
	log.Printf("Znaleziono %v linków", len(urls))
	if err != nil {
		log.Println("Błąd wyciąganie url z kontentu")
		return nil, err
	}

	return urls, nil
}
