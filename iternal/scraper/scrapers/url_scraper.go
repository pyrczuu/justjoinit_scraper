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
	//"github.com/chromedp/cdproto/emulation"
	"github.com/chromedp/chromedp"
)

//browser session data dir

const (
	browserDataDir = `~/.config/google-chrome/Default`
	source         = "https://justjoin.it"
	minTimeMs      = 3000
	maxTimeMs      = 4000
	prefix         = "https://justjoin.it/job-offer/"
	offerSelector  = "a.offer-card"
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

	err := chromedp.Run(chromeDpCtx,

		chromedp.ActionFunc(func(ctx context.Context) error {
			return emulation.SetDeviceMetricsOverride(1280, 900, 1.0, false).Do(ctx)
		}),
		chromedp.Navigate(source),
		chromedp.Evaluate(`delete navigator.__proto__.webdriver`, nil),
		chromedp.WaitVisible(`body`, chromedp.ByQuery),

		chromedp.ActionFunc(func(ctx context.Context) error {
			var prevHeight int64 = -99
			var currentHeight int64
			var html string

			log.Println("Strona załadowana. Rozpoczynanie pętli wewnętrznej...")

			sameHeightCount := 0
			for i := 1; ; i++ {
				err := chromedp.Evaluate(`document.body.scrollHeight`, &currentHeight).Do(ctx)
				if err != nil {
					return fmt.Errorf("błąd pobierania wysokości: %w", err)
				}

				if currentHeight == prevHeight {
					sameHeightCount++
				}

				if sameHeightCount > 10 {
					log.Printf("Koniec strony, wysokość (%d).", currentHeight)
					break
				}

				if err := chromedp.OuterHTML("html", &html).Do(ctx); err != nil {
					log.Printf("Błąd odczytu HTML: %v", err)
				} else {
					collected, err := getUrlsFromContent(html)
					if err == nil {
						urls = append(urls, collected...)
						log.Printf("Iteracja %d: Znaleziono %d linków (razem: %d)", i, len(collected), len(urls))
					}
				}

				prevHeight = currentHeight
				log.Printf("Scrollowanie do: %d", currentHeight)

				randomDelay := rand.Intn(maxTimeMs-minTimeMs) + minTimeMs
				err = chromedp.Sleep(time.Duration(randomDelay) * time.Millisecond).Do(ctx)
				if err != nil {
					return err
				}

				err = chromedp.Evaluate(`window.scrollBy(0, 1400);`, nil).Do(ctx)
				if err != nil {
					return err
				}
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
	)

	if err != nil {
		return nil, fmt.Errorf("błąd wykonania chromedp: %w", err)
	}

	return urls, nil
}
