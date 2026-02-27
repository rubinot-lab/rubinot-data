package scraper

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/giovannirco/rubinot-data/internal/domain"
	"github.com/giovannirco/rubinot-data/internal/validation"
	"go.opentelemetry.io/otel/attribute"
)

type newsAPIResponse struct {
	Tickers []struct {
		ID         int    `json:"id"`
		Message    string `json:"message"`
		CategoryID int    `json:"category_id"`
		Category   struct {
			ID      int    `json:"id"`
			Name    string `json:"name"`
			Slug    string `json:"slug"`
			Color   string `json:"color"`
			Icon    string `json:"icon"`
			IconURL string `json:"icon_url"`
		} `json:"category"`
		Author    string `json:"author"`
		CreatedAt string `json:"created_at"`
	} `json:"tickers"`
	Articles []struct {
		ID         int    `json:"id"`
		Title      string `json:"title"`
		Slug       string `json:"slug"`
		Summary    string `json:"summary"`
		Content    string `json:"content"`
		CoverImage string `json:"cover_image"`
		Author     string `json:"author"`
		Category   struct {
			ID      int    `json:"id"`
			Name    string `json:"name"`
			Slug    string `json:"slug"`
			Color   string `json:"color"`
			Icon    string `json:"icon"`
			IconURL string `json:"icon_url"`
		} `json:"category"`
		PublishedAt string `json:"published_at"`
	} `json:"articles"`
}

type newsListEntryWithTime struct {
	entry domain.NewsListEntry
	time  time.Time
}

func FetchNewsByID(
	ctx context.Context,
	baseURL string,
	newsID int,
	opts FetchOptions,
) (domain.NewsResult, []string, error) {
	ctx, span := tracer.Start(ctx, "scraper.FetchNewsByID")
	defer span.End()

	sourceURL := fmt.Sprintf("%s/api/news", strings.TrimRight(baseURL, "/"))
	span.SetAttributes(
		attribute.String("rubinot.endpoint", "news"),
		attribute.String("rubinot.source_url", sourceURL),
		attribute.Int("rubinot.news_id", newsID),
	)

	payload, err := fetchNewsPayload(ctx, sourceURL, opts)
	if err != nil {
		return domain.NewsResult{}, []string{sourceURL}, err
	}

	parseStarted := time.Now()
	article, ok := findNewsArticleByID(payload, newsID)
	if ok {
		parseDuration.WithLabelValues("news").Observe(time.Since(parseStarted).Seconds())
		return article, []string{sourceURL}, nil
	}

	ticker, ok := findNewsTickerByID(payload, newsID)
	parseDuration.WithLabelValues("news").Observe(time.Since(parseStarted).Seconds())
	if ok {
		return ticker, []string{sourceURL}, nil
	}

	return domain.NewsResult{}, []string{sourceURL}, validation.NewError(validation.ErrorEntityNotFound, "news entry not found", nil)
}

func FetchNewsArchive(
	ctx context.Context,
	baseURL string,
	archiveDays int,
	opts FetchOptions,
) (domain.NewsListResult, string, error) {
	sourceURL := fmt.Sprintf("%s/api/news", strings.TrimRight(baseURL, "/"))
	payload, err := fetchNewsPayload(ctx, sourceURL, opts)
	if err != nil {
		return domain.NewsListResult{}, sourceURL, err
	}

	cutoff := time.Now().UTC().AddDate(0, 0, -archiveDays)
	entries := buildNewsArchiveEntries(payload, cutoff)
	return domain.NewsListResult{
		Mode:        "archive",
		ArchiveDays: archiveDays,
		Entries:     entries,
	}, sourceURL, nil
}

func FetchNewsLatest(ctx context.Context, baseURL string, opts FetchOptions) (domain.NewsListResult, string, error) {
	sourceURL := fmt.Sprintf("%s/api/news", strings.TrimRight(baseURL, "/"))
	payload, err := fetchNewsPayload(ctx, sourceURL, opts)
	if err != nil {
		return domain.NewsListResult{}, sourceURL, err
	}

	entries := make([]newsListEntryWithTime, 0, len(payload.Articles))
	for _, article := range payload.Articles {
		at := parseNewsTimestamp(article.PublishedAt)
		entries = append(entries, newsListEntryWithTime{
			entry: domain.NewsListEntry{
				ID:          article.ID,
				Date:        article.PublishedAt,
				Title:       strings.TrimSpace(article.Title),
				Category:    strings.TrimSpace(article.Category.Name),
				Type:        "article",
				URL:         fmt.Sprintf("/news/%s", strings.TrimSpace(article.Slug)),
				Author:      strings.TrimSpace(article.Author),
				Slug:        strings.TrimSpace(article.Slug),
				Summary:     strings.TrimSpace(article.Summary),
				CategoryRef: toNewsCategory(article.Category.ID, article.Category.Name, article.Category.Slug, article.Category.Color, article.Category.Icon, article.Category.IconURL),
			},
			time: at,
		})
	}
	sortNewsEntries(entries)

	resultEntries := make([]domain.NewsListEntry, 0, len(entries))
	for _, entry := range entries {
		resultEntries = append(resultEntries, entry.entry)
	}

	return domain.NewsListResult{
		Mode:    "latest",
		Entries: resultEntries,
	}, sourceURL, nil
}

func FetchNewsTicker(ctx context.Context, baseURL string, opts FetchOptions) (domain.NewsListResult, string, error) {
	sourceURL := fmt.Sprintf("%s/api/news", strings.TrimRight(baseURL, "/"))
	payload, err := fetchNewsPayload(ctx, sourceURL, opts)
	if err != nil {
		return domain.NewsListResult{}, sourceURL, err
	}

	entries := make([]newsListEntryWithTime, 0, len(payload.Tickers))
	for _, ticker := range payload.Tickers {
		at := parseNewsTimestamp(ticker.CreatedAt)
		entries = append(entries, newsListEntryWithTime{
			entry: domain.NewsListEntry{
				ID:          ticker.ID,
				Date:        ticker.CreatedAt,
				Category:    strings.TrimSpace(ticker.Category.Name),
				Type:        "ticker",
				Message:     ticker.Message,
				Author:      strings.TrimSpace(ticker.Author),
				CategoryRef: toNewsCategory(ticker.Category.ID, ticker.Category.Name, ticker.Category.Slug, ticker.Category.Color, ticker.Category.Icon, ticker.Category.IconURL),
			},
			time: at,
		})
	}
	sortNewsEntries(entries)

	resultEntries := make([]domain.NewsListEntry, 0, len(entries))
	for _, entry := range entries {
		resultEntries = append(resultEntries, entry.entry)
	}

	return domain.NewsListResult{
		Mode:    "newsticker",
		Entries: resultEntries,
	}, sourceURL, nil
}

func fetchNewsPayload(ctx context.Context, sourceURL string, opts FetchOptions) (newsAPIResponse, error) {
	started := time.Now()
	client := NewClient(opts)

	var payload newsAPIResponse
	err := client.FetchJSON(ctx, sourceURL, &payload)
	scrapeDuration.WithLabelValues("news").Observe(time.Since(started).Seconds())
	if err != nil {
		scrapeRequests.WithLabelValues("news", "error").Inc()
		return newsAPIResponse{}, err
	}
	scrapeRequests.WithLabelValues("news", "ok").Inc()
	ParseItems.WithLabelValues("news").Set(float64(len(payload.Articles) + len(payload.Tickers)))
	return payload, nil
}

func findNewsArticleByID(payload newsAPIResponse, id int) (domain.NewsResult, bool) {
	for _, article := range payload.Articles {
		if article.ID != id {
			continue
		}
		return domain.NewsResult{
			ID:          article.ID,
			Date:        article.PublishedAt,
			Title:       strings.TrimSpace(article.Title),
			Category:    strings.TrimSpace(article.Category.Name),
			CategoryRef: toNewsCategory(article.Category.ID, article.Category.Name, article.Category.Slug, article.Category.Color, article.Category.Icon, article.Category.IconURL),
			Type:        "article",
			Content:     article.Content,
			ContentHTML: article.Content,
			Author:      strings.TrimSpace(article.Author),
			Slug:        strings.TrimSpace(article.Slug),
			Summary:     strings.TrimSpace(article.Summary),
			CoverImage:  strings.TrimSpace(article.CoverImage),
		}, true
	}
	return domain.NewsResult{}, false
}

func findNewsTickerByID(payload newsAPIResponse, id int) (domain.NewsResult, bool) {
	for _, ticker := range payload.Tickers {
		if ticker.ID != id {
			continue
		}
		return domain.NewsResult{
			ID:          ticker.ID,
			Date:        ticker.CreatedAt,
			Category:    strings.TrimSpace(ticker.Category.Name),
			CategoryRef: toNewsCategory(ticker.Category.ID, ticker.Category.Name, ticker.Category.Slug, ticker.Category.Color, ticker.Category.Icon, ticker.Category.IconURL),
			Type:        "ticker",
			Content:     ticker.Message,
			ContentHTML: ticker.Message,
			Author:      strings.TrimSpace(ticker.Author),
		}, true
	}
	return domain.NewsResult{}, false
}

func buildNewsArchiveEntries(payload newsAPIResponse, cutoff time.Time) []domain.NewsListEntry {
	entries := make([]newsListEntryWithTime, 0, len(payload.Articles)+len(payload.Tickers))

	for _, article := range payload.Articles {
		at := parseNewsTimestamp(article.PublishedAt)
		if !at.IsZero() && at.Before(cutoff) {
			continue
		}
		entries = append(entries, newsListEntryWithTime{
			entry: domain.NewsListEntry{
				ID:          article.ID,
				Date:        article.PublishedAt,
				Title:       strings.TrimSpace(article.Title),
				Category:    strings.TrimSpace(article.Category.Name),
				Type:        "article",
				URL:         fmt.Sprintf("/news/%s", strings.TrimSpace(article.Slug)),
				Author:      strings.TrimSpace(article.Author),
				Slug:        strings.TrimSpace(article.Slug),
				Summary:     strings.TrimSpace(article.Summary),
				CategoryRef: toNewsCategory(article.Category.ID, article.Category.Name, article.Category.Slug, article.Category.Color, article.Category.Icon, article.Category.IconURL),
			},
			time: at,
		})
	}

	for _, ticker := range payload.Tickers {
		at := parseNewsTimestamp(ticker.CreatedAt)
		if !at.IsZero() && at.Before(cutoff) {
			continue
		}
		entries = append(entries, newsListEntryWithTime{
			entry: domain.NewsListEntry{
				ID:          ticker.ID,
				Date:        ticker.CreatedAt,
				Category:    strings.TrimSpace(ticker.Category.Name),
				Type:        "ticker",
				Message:     ticker.Message,
				Author:      strings.TrimSpace(ticker.Author),
				CategoryRef: toNewsCategory(ticker.Category.ID, ticker.Category.Name, ticker.Category.Slug, ticker.Category.Color, ticker.Category.Icon, ticker.Category.IconURL),
			},
			time: at,
		})
	}

	sortNewsEntries(entries)

	result := make([]domain.NewsListEntry, 0, len(entries))
	for _, entry := range entries {
		result = append(result, entry.entry)
	}
	return result
}

func sortNewsEntries(entries []newsListEntryWithTime) {
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].time.After(entries[j].time)
	})
}

func parseNewsTimestamp(raw string) time.Time {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return time.Time{}
	}
	parsed, err := time.Parse(time.RFC3339, trimmed)
	if err != nil {
		return time.Time{}
	}
	return parsed.UTC()
}

func toNewsCategory(id int, name, slug, color, icon, iconURL string) domain.NewsCategory {
	return domain.NewsCategory{
		ID:      id,
		Name:    strings.TrimSpace(name),
		Slug:    strings.TrimSpace(slug),
		Color:   strings.TrimSpace(color),
		Icon:    strings.TrimSpace(icon),
		IconURL: strings.TrimSpace(iconURL),
	}
}
