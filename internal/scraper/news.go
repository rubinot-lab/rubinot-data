package scraper

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/giovannirco/rubinot-data/internal/domain"
	"github.com/giovannirco/rubinot-data/internal/validation"
	"go.opentelemetry.io/otel/attribute"
)

var (
	newsArchiveIDPattern  = regexp.MustCompile(`(?i)news/archive/(\d+)`)
	newsBracketCategoryRe = regexp.MustCompile(`^\[([^\]]+)\]`)
	newsNotFoundMessageRe = regexp.MustCompile(`(?i)this news doesn't exist or is hidden`)
	newsRequestNotFoundRe = regexp.MustCompile(`(?i)the requested url .* was not found on this server`)
)

func FetchNewsByID(
	ctx context.Context,
	baseURL string,
	newsID int,
	opts FetchOptions,
) (domain.NewsResult, []string, error) {
	ctx, span := tracer.Start(ctx, "scraper.FetchNewsByID")
	defer span.End()

	client := NewClient(opts)
	articleURL := fmt.Sprintf("%s/?news/archive/%d", strings.TrimRight(baseURL, "/"), newsID)
	tickerURL := fmt.Sprintf("%s/?news", strings.TrimRight(baseURL, "/"))

	span.SetAttributes(
		attribute.String("rubinot.endpoint", "news"),
		attribute.Int("rubinot.news_id", newsID),
		attribute.String("rubinot.source_url", articleURL),
	)

	started := time.Now()
	articleHTML, err := client.Fetch(ctx, articleURL)
	scrapeDuration.WithLabelValues("news").Observe(time.Since(started).Seconds())
	if err != nil {
		scrapeRequests.WithLabelValues("news", "error").Inc()
		return domain.NewsResult{}, []string{articleURL}, err
	}
	scrapeRequests.WithLabelValues("news", "ok").Inc()

	parseStarted := time.Now()
	article, notFound, parseErr := parseNewsArticleHTML(newsID, articleHTML)
	parseDuration.WithLabelValues("news").Observe(time.Since(parseStarted).Seconds())
	if parseErr != nil {
		return domain.NewsResult{}, []string{articleURL}, parseErr
	}
	if !notFound {
		return article, []string{articleURL}, nil
	}

	started = time.Now()
	tickerHTML, err := client.Fetch(ctx, tickerURL)
	scrapeDuration.WithLabelValues("news").Observe(time.Since(started).Seconds())
	if err != nil {
		scrapeRequests.WithLabelValues("news", "error").Inc()
		return domain.NewsResult{}, []string{articleURL, tickerURL}, err
	}
	scrapeRequests.WithLabelValues("news", "ok").Inc()

	parseStarted = time.Now()
	tickerEntry, tickerNotFound, parseTickerErr := parseNewsTickerEntryByIndex(newsID, tickerHTML)
	parseDuration.WithLabelValues("news").Observe(time.Since(parseStarted).Seconds())
	if parseTickerErr != nil {
		return domain.NewsResult{}, []string{articleURL, tickerURL}, parseTickerErr
	}
	if tickerNotFound {
		return domain.NewsResult{}, []string{articleURL, tickerURL}, validation.NewError(validation.ErrorEntityNotFound, "news not found", nil)
	}

	return tickerEntry, []string{tickerURL}, nil
}

func FetchNewsArchive(
	ctx context.Context,
	baseURL string,
	archiveDays int,
	opts FetchOptions,
) (domain.NewsListResult, string, error) {
	ctx, span := tracer.Start(ctx, "scraper.FetchNewsArchive")
	defer span.End()

	client := NewClient(opts)
	sourceURL := fmt.Sprintf("%s/?news/archive", strings.TrimRight(baseURL, "/"))
	span.SetAttributes(
		attribute.String("rubinot.endpoint", "news.archive"),
		attribute.Int("rubinot.archive_days", archiveDays),
		attribute.String("rubinot.source_url", sourceURL),
	)

	started := time.Now()
	htmlBody, err := client.Fetch(ctx, sourceURL)
	scrapeDuration.WithLabelValues("newslist").Observe(time.Since(started).Seconds())
	if err != nil {
		scrapeRequests.WithLabelValues("newslist", "error").Inc()
		return domain.NewsListResult{}, sourceURL, err
	}
	scrapeRequests.WithLabelValues("newslist", "ok").Inc()

	parseStarted := time.Now()
	result, parseErr := parseNewsArchiveListHTML(htmlBody, archiveDays)
	parseDuration.WithLabelValues("newslist").Observe(time.Since(parseStarted).Seconds())
	if parseErr != nil {
		return domain.NewsListResult{}, sourceURL, parseErr
	}

	return result, sourceURL, nil
}

func FetchNewsLatest(ctx context.Context, baseURL string, opts FetchOptions) (domain.NewsListResult, string, error) {
	ctx, span := tracer.Start(ctx, "scraper.FetchNewsLatest")
	defer span.End()

	client := NewClient(opts)
	sourceURL := fmt.Sprintf("%s/?news", strings.TrimRight(baseURL, "/"))
	span.SetAttributes(
		attribute.String("rubinot.endpoint", "news.latest"),
		attribute.String("rubinot.source_url", sourceURL),
	)

	started := time.Now()
	htmlBody, err := client.Fetch(ctx, sourceURL)
	scrapeDuration.WithLabelValues("newslist").Observe(time.Since(started).Seconds())
	if err != nil {
		scrapeRequests.WithLabelValues("newslist", "error").Inc()
		return domain.NewsListResult{}, sourceURL, err
	}
	scrapeRequests.WithLabelValues("newslist", "ok").Inc()

	parseStarted := time.Now()
	result, parseErr := parseNewsLatestListHTML(htmlBody)
	parseDuration.WithLabelValues("newslist").Observe(time.Since(parseStarted).Seconds())
	if parseErr != nil {
		return domain.NewsListResult{}, sourceURL, parseErr
	}

	return result, sourceURL, nil
}

func FetchNewsTicker(ctx context.Context, baseURL string, opts FetchOptions) (domain.NewsListResult, string, error) {
	ctx, span := tracer.Start(ctx, "scraper.FetchNewsTicker")
	defer span.End()

	client := NewClient(opts)
	sourceURL := fmt.Sprintf("%s/?news", strings.TrimRight(baseURL, "/"))
	span.SetAttributes(
		attribute.String("rubinot.endpoint", "news.newsticker"),
		attribute.String("rubinot.source_url", sourceURL),
	)

	started := time.Now()
	htmlBody, err := client.Fetch(ctx, sourceURL)
	scrapeDuration.WithLabelValues("newslist").Observe(time.Since(started).Seconds())
	if err != nil {
		scrapeRequests.WithLabelValues("newslist", "error").Inc()
		return domain.NewsListResult{}, sourceURL, err
	}
	scrapeRequests.WithLabelValues("newslist", "ok").Inc()

	parseStarted := time.Now()
	result, parseErr := parseNewsTickerListHTML(htmlBody)
	parseDuration.WithLabelValues("newslist").Observe(time.Since(parseStarted).Seconds())
	if parseErr != nil {
		return domain.NewsListResult{}, sourceURL, parseErr
	}

	return result, sourceURL, nil
}

func parseNewsArticleHTML(newsID int, htmlBody string) (domain.NewsResult, bool, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlBody))
	if err != nil {
		return domain.NewsResult{}, false, err
	}

	fullText := normalizeText(doc.Text())
	if newsNotFoundMessageRe.MatchString(fullText) || newsRequestNotFoundRe.MatchString(fullText) {
		return domain.NewsResult{}, true, nil
	}

	headline := doc.Find(".NewsHeadline").First()
	if headline.Length() == 0 {
		return domain.NewsResult{}, true, nil
	}

	date := parseNewsDateToUTC(headline.Find(".NewsHeadlineDate").First().Text())
	title := normalizeText(headline.Find(".NewsHeadlineText").First().Text())
	category := extractNewsCategory(title)

	contentContainer := headline.NextAllFiltered("table").First().Find("td").First()
	contentText := normalizeText(contentContainer.Text())
	contentHTML, _ := contentContainer.Html()

	if title == "" && contentText == "" {
		return domain.NewsResult{}, true, nil
	}

	result := domain.NewsResult{
		ID:          newsID,
		Date:        date,
		Title:       title,
		Category:    category,
		Type:        "article",
		Content:     contentText,
		ContentHTML: strings.TrimSpace(contentHTML),
	}
	return result, false, nil
}

func parseNewsTickerEntryByIndex(newsID int, htmlBody string) (domain.NewsResult, bool, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlBody))
	if err != nil {
		return domain.NewsResult{}, false, err
	}

	rows := doc.Find("#NewsTicker .Row")
	if rows.Length() == 0 {
		return domain.NewsResult{}, true, nil
	}

	index := newsID - 1
	if index < 0 || index >= rows.Length() {
		return domain.NewsResult{}, true, nil
	}

	row := rows.Eq(index)
	date := parseNewsDateToUTC(row.Find(".NewsTickerDate").First().Text())

	fullTextNode := row.Find(".NewsTickerFullText").First()
	shortTextNode := row.Find(".NewsTickerShortText").First()
	contentText := normalizeText(fullTextNode.Text())
	if contentText == "" {
		contentText = normalizeText(shortTextNode.Text())
	}
	contentHTML, _ := fullTextNode.Html()
	if strings.TrimSpace(contentHTML) == "" {
		contentHTML, _ = shortTextNode.Html()
	}

	category := extractNewsCategory(contentText)
	title := category
	if title == "" {
		title = "Ticker Entry"
	}

	result := domain.NewsResult{
		ID:          newsID,
		Date:        date,
		Title:       title,
		Category:    category,
		Type:        "ticker",
		Content:     contentText,
		ContentHTML: strings.TrimSpace(contentHTML),
	}
	return result, false, nil
}

func parseNewsArchiveListHTML(htmlBody string, archiveDays int) (domain.NewsListResult, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlBody))
	if err != nil {
		return domain.NewsListResult{}, err
	}

	// TODO: AMBIGUOUS — fixtures do not expose a reliable archive-days query parameter on rubinot upstream.
	result := domain.NewsListResult{
		Mode:        "archive",
		ArchiveDays: archiveDays,
		Entries:     make([]domain.NewsListEntry, 0),
	}

	doc.Find("table tr").Each(func(_ int, row *goquery.Selection) {
		link := row.Find("a[href*='news/archive/']").First()
		if link.Length() == 0 {
			return
		}

		href, _ := link.Attr("href")
		title := normalizeText(link.Text())
		if title == "" {
			return
		}

		cells := row.Find("td")
		if cells.Length() < 3 {
			return
		}

		date := parseNewsDateToUTC(cells.Eq(1).Text())
		id := parseNewsArchiveID(href)
		category := extractNewsCategory(title)
		if category == "" {
			category = "news"
		}

		result.Entries = append(result.Entries, domain.NewsListEntry{
			ID:       id,
			Date:     date,
			Title:    title,
			Category: category,
			Type:     "article",
			URL:      strings.TrimSpace(href),
		})
	})

	return result, nil
}

func parseNewsLatestListHTML(htmlBody string) (domain.NewsListResult, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlBody))
	if err != nil {
		return domain.NewsListResult{}, err
	}

	result := domain.NewsListResult{
		Mode:    "latest",
		Entries: make([]domain.NewsListEntry, 0),
	}

	doc.Find(".NewsHeadline").Each(func(_ int, headline *goquery.Selection) {
		title := normalizeText(headline.Find(".NewsHeadlineText").First().Text())
		if title == "" {
			return
		}

		date := parseNewsDateToUTC(headline.Find(".NewsHeadlineDate").First().Text())
		category := extractNewsCategory(title)
		if category == "" {
			category = "news"
		}

		result.Entries = append(result.Entries, domain.NewsListEntry{
			Date:     date,
			Title:    title,
			Category: category,
			Type:     "article",
		})
	})

	return result, nil
}

func parseNewsTickerListHTML(htmlBody string) (domain.NewsListResult, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlBody))
	if err != nil {
		return domain.NewsListResult{}, err
	}

	result := domain.NewsListResult{
		Mode:    "newsticker",
		Entries: make([]domain.NewsListEntry, 0),
	}

	doc.Find("#NewsTicker .Row").Each(func(index int, row *goquery.Selection) {
		date := parseNewsDateToUTC(row.Find(".NewsTickerDate").First().Text())
		contentText := normalizeText(row.Find(".NewsTickerFullText").First().Text())
		if contentText == "" {
			contentText = normalizeText(row.Find(".NewsTickerShortText").First().Text())
		}
		if contentText == "" {
			return
		}

		category := extractNewsCategory(contentText)
		title := category
		if title == "" {
			title = "Ticker Entry"
		}

		result.Entries = append(result.Entries, domain.NewsListEntry{
			ID:       index + 1,
			Date:     date,
			Title:    title,
			Category: category,
			Type:     "ticker",
		})
	})

	return result, nil
}

func parseNewsDateToUTC(raw string) string {
	value := strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(raw), "-"))
	if value == "" {
		return ""
	}

	parsed, err := parseRubinotDateToUTC(value)
	if err != nil {
		return ""
	}
	return parsed
}

func parseNewsArchiveID(urlValue string) int {
	match := newsArchiveIDPattern.FindStringSubmatch(urlValue)
	if len(match) != 2 {
		return 0
	}
	return parseInt(match[1])
}

func extractNewsCategory(text string) string {
	value := normalizeText(text)
	match := newsBracketCategoryRe.FindStringSubmatch(value)
	if len(match) == 2 {
		return strings.TrimSpace(match[1])
	}
	return ""
}
