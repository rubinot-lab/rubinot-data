package scraper

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/go-resty/resty/v2"
	"go.opentelemetry.io/otel/attribute"
)

type FetchOptions struct {
	FlareSolverrURL string
	MaxTimeoutMs    int
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

type flareSolverrRequest struct {
	Cmd        string            `json:"cmd"`
	URL        string            `json:"url"`
	MaxTimeout int               `json:"maxTimeout,omitempty"`
	Session    string            `json:"session,omitempty"`
	Headers    map[string]string `json:"headers,omitempty"`
}

type flareSolverrResponse struct {
	Status   string `json:"status"`
	Message  string `json:"message"`
	Solution struct {
		Response string `json:"response"`
		Status   int    `json:"status"`
		URL      string `json:"url"`
	} `json:"solution"`
}

func FetchWorld(ctx context.Context, baseURL, world string, opts FetchOptions) (WorldResult, string, error) {
	ctx, span := tracer.Start(ctx, "scraper.FetchWorld")
	defer span.End()

	started := time.Now()
	formatted := strings.Title(strings.ToLower(strings.TrimSpace(world)))
	sourceURL := fmt.Sprintf("%s/?subtopic=worlds&world=%s", strings.TrimRight(baseURL, "/"), url.QueryEscape(formatted))

	if opts.FlareSolverrURL == "" {
		opts.FlareSolverrURL = "http://flaresolverr.network.svc.cluster.local:8191/v1"
	}
	if opts.MaxTimeoutMs <= 0 {
		opts.MaxTimeoutMs = 120000
	}

	span.SetAttributes(
		attribute.String("rubinot.endpoint", "world"),
		attribute.String("rubinot.world", formatted),
		attribute.String("rubinot.source_url", sourceURL),
	)

	htmlBody, err := fetchViaFlareSolverr(ctx, sourceURL, opts)
	scrapeDuration.WithLabelValues("world").Observe(time.Since(started).Seconds())
	if err != nil {
		scrapeRequests.WithLabelValues("world", "error").Inc()
		return WorldResult{}, sourceURL, err
	}
	scrapeRequests.WithLabelValues("world", "ok").Inc()

	parseStart := time.Now()
	result, source, parseErr := parseWorldHTML(formatted, sourceURL, htmlBody)
	parseDuration.WithLabelValues("world").Observe(time.Since(parseStart).Seconds())
	if parseErr != nil {
		return WorldResult{}, sourceURL, parseErr
	}

	return result, source, nil
}

func fetchViaFlareSolverr(ctx context.Context, sourceURL string, opts FetchOptions) (string, error) {
	ctx, span := tracer.Start(ctx, "scraper.fetchViaFlareSolverr")
	defer span.End()

	client := resty.New().SetTimeout(140 * time.Second)
	payload := flareSolverrRequest{
		Cmd:        "request.get",
		URL:        sourceURL,
		MaxTimeout: opts.MaxTimeoutMs,
		Headers: map[string]string{
			"User-Agent":      "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/145.0.0.0 Safari/537.36",
			"Accept-Language": "en-US,en;q=0.9,pt-BR;q=0.8",
		},
	}

	var out flareSolverrResponse
	res, err := client.R().
		SetContext(ctx).
		SetHeader("Content-Type", "application/json").
		SetBody(payload).
		SetResult(&out).
		Post(opts.FlareSolverrURL)
	if err != nil {
		return "", fmt.Errorf("flaresolverr request failed: %w", err)
	}
	if res.StatusCode() != 200 {
		return "", fmt.Errorf("flaresolverr returned non-200: %d", res.StatusCode())
	}
	if strings.ToLower(out.Status) != "ok" {
		return "", fmt.Errorf("flaresolverr error: %s", out.Message)
	}
	if out.Solution.Status != 200 {
		return "", fmt.Errorf("target returned non-200 via flaresolverr: %d", out.Solution.Status)
	}

	html := out.Solution.Response
	lower := strings.ToLower(html)
	if strings.Contains(lower, "just a moment") || strings.Contains(lower, "challenge-platform") || strings.Contains(lower, "cf-challenge") {
		return "", fmt.Errorf("cloudflare challenge page still present after flaresolverr")
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
