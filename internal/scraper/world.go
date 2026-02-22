package scraper

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/chromedp/cdproto/emulation"
	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
)

type FetchOptions struct {
	Mode            string
	BrowserPath     string
	BrowserFallback bool
}

type PlayerOnline struct {
	Name     string `json:"name"`
	Level    int    `json:"level"`
	Vocation string `json:"vocation"`
}

type WorldInfo struct {
	Status        string `json:"status,omitempty"`
	PlayersOnline int    `json:"players_online"`
	Location      string `json:"location,omitempty"`
	PVPType       string `json:"pvp_type,omitempty"`
	CreationDate  string `json:"creation_date,omitempty"`
}

type WorldResult struct {
	Name          string         `json:"name"`
	Info          WorldInfo      `json:"info"`
	PlayersOnline []PlayerOnline `json:"players_online_list"`
}

func FetchWorld(baseURL, world string, opts FetchOptions) (WorldResult, string, error) {
	formatted := strings.Title(strings.ToLower(strings.TrimSpace(world)))
	sourceURL := fmt.Sprintf("%s/?subtopic=worlds&world=%s", strings.TrimRight(baseURL, "/"), url.QueryEscape(formatted))

	htmlBody, err := fetchBrowser(sourceURL, opts.BrowserPath)
	if err != nil {
		return WorldResult{}, sourceURL, err
	}

	return parseWorldHTML(formatted, sourceURL, htmlBody)
}

func fetchBrowser(sourceURL, browserPath string) (string, error) {
	if browserPath == "" {
		browserPath = "/usr/bin/chromium-browser"
	}

	allocOpts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.ExecPath(browserPath),
		chromedp.Flag("headless", true),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-dev-shm-usage", true),
		chromedp.Flag("disable-crash-reporter", true),
		chromedp.Flag("disable-breakpad", true),
		chromedp.Flag("no-first-run", true),
		chromedp.Flag("no-default-browser-check", true),
		chromedp.Flag("disable-blink-features", "AutomationControlled"),
		chromedp.Flag("user-data-dir", "/tmp/chromium-data"),
		chromedp.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/145.0.0.0 Safari/537.36"),
	)

	allocCtx, cancel := chromedp.NewExecAllocator(context.Background(), allocOpts...)
	defer cancel()

	ctx, cancelCtx := chromedp.NewContext(allocCtx)
	defer cancelCtx()

	ctx, cancelTimeout := context.WithTimeout(ctx, 45*time.Second)
	defer cancelTimeout()

	var html string
	err := chromedp.Run(ctx,
		emulation.SetTimezoneOverride("UTC"),
		chromedp.ActionFunc(func(ctx context.Context) error {
			_, err := page.AddScriptToEvaluateOnNewDocument(`Object.defineProperty(navigator, 'webdriver', {get: () => undefined});` ).Do(ctx)
			return err
		}),
		chromedp.Navigate(sourceURL),
		chromedp.Sleep(8*time.Second),
		chromedp.WaitVisible("body", chromedp.ByQuery),
		chromedp.OuterHTML("html", &html, chromedp.ByQuery),
	)
	if err != nil {
		return "", err
	}

	if strings.Contains(strings.ToLower(html), "just a moment") || strings.Contains(strings.ToLower(html), "cf-challenge") {
		return "", fmt.Errorf("cloudflare challenge page still present after browser render")
	}

	return html, nil
}

func parseWorldHTML(formatted, sourceURL, htmlBody string) (WorldResult, string, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlBody))
	if err != nil {
		return WorldResult{}, sourceURL, err
	}

	result := WorldResult{Name: formatted}
	result.Info = parseWorldInfo(doc)
	result.PlayersOnline = parsePlayers(doc)

	if result.Info.PlayersOnline == 0 {
		result.Info.PlayersOnline = len(result.PlayersOnline)
	}

	return result, sourceURL, nil
}

func parseWorldInfo(doc *goquery.Document) WorldInfo {
	info := WorldInfo{}
	doc.Find("tr").Each(func(_ int, tr *goquery.Selection) {
		tds := tr.Find("td")
		if tds.Length() != 2 {
			return
		}
		label := strings.TrimSpace(strings.TrimSuffix(tds.Eq(0).Text(), ":"))
		value := strings.TrimSpace(tds.Eq(1).Text())
		switch strings.ToLower(label) {
		case "status":
			info.Status = value
		case "players online":
			info.PlayersOnline = parseInt(value)
		case "location":
			info.Location = value
		case "pvp type":
			info.PVPType = value
		case "creation date":
			info.CreationDate = value
		}
	})
	return info
}

func parsePlayers(doc *goquery.Document) []PlayerOnline {
	out := make([]PlayerOnline, 0)
	doc.Find("table").EachWithBreak(func(_ int, table *goquery.Selection) bool {
		headers := strings.ToLower(strings.Join(strings.Fields(table.Find("tr").First().Text()), " "))
		if !(strings.Contains(headers, "name") && strings.Contains(headers, "level") && strings.Contains(headers, "vocation")) {
			return true
		}

		table.Find("tr").Slice(1, goquery.ToEnd).Each(func(_ int, tr *goquery.Selection) {
			tds := tr.Find("td")
			if tds.Length() < 3 {
				return
			}
			name := strings.TrimSpace(tds.Eq(0).Text())
			lvl := parseInt(strings.TrimSpace(tds.Eq(1).Text()))
			voc := strings.TrimSpace(tds.Eq(2).Text())
			if name == "" || lvl <= 0 || voc == "" {
				return
			}
			out = append(out, PlayerOnline{Name: name, Level: lvl, Vocation: voc})
		})
		return false
	})
	return out
}

func parseInt(s string) int {
	s = strings.ReplaceAll(s, ",", "")
	i, _ := strconv.Atoi(s)
	return i
}
