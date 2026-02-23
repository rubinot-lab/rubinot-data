package scraper

import (
	"context"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/giovannirco/rubinot-data/internal/domain"
	"github.com/giovannirco/rubinot-data/internal/validation"
	"go.opentelemetry.io/otel/attribute"
)

var (
	highscoresAgePattern      = regexp.MustCompile(`(?i)last update:\s*(\d+)\s*minutes?`)
	highscoresResultsPattern  = regexp.MustCompile(`(?i)results:\s*([\d.,]+)`)
	highscoresPageHrefPattern = regexp.MustCompile(`(?i)(?:\?|&)currentpage=(\d+)`)
)

func FetchHighscores(
	ctx context.Context,
	baseURL string,
	worldName string,
	category validation.HighscoreCategory,
	vocation validation.HighscoreVocation,
	page int,
	opts FetchOptions,
) (domain.HighscoresResult, string, error) {
	ctx, span := tracer.Start(ctx, "scraper.FetchHighscores")
	defer span.End()

	started := time.Now()
	sourceURL := fmt.Sprintf(
		"%s/?subtopic=highscores&world=%s&category=%d&currentpage=%d&profession=%d",
		strings.TrimRight(baseURL, "/"),
		url.QueryEscape(worldName),
		category.ID,
		page,
		vocation.ProfessionID,
	)
	client := NewClient(opts)

	span.SetAttributes(
		attribute.String("rubinot.endpoint", "highscores"),
		attribute.String("rubinot.world", worldName),
		attribute.String("rubinot.category", category.Slug),
		attribute.String("rubinot.vocation", vocation.Name),
		attribute.Int("rubinot.page", page),
		attribute.String("rubinot.source_url", sourceURL),
	)

	htmlBody, err := client.Fetch(ctx, sourceURL)
	scrapeDuration.WithLabelValues("highscores").Observe(time.Since(started).Seconds())
	if err != nil {
		scrapeRequests.WithLabelValues("highscores", "error").Inc()
		return domain.HighscoresResult{}, sourceURL, err
	}
	scrapeRequests.WithLabelValues("highscores", "ok").Inc()

	parseStarted := time.Now()
	result, parseErr := parseHighscoresHTML(htmlBody, worldName, category.Slug, vocation.Name, page)
	parseDuration.WithLabelValues("highscores").Observe(time.Since(parseStarted).Seconds())
	if parseErr != nil {
		return domain.HighscoresResult{}, sourceURL, parseErr
	}

	return result, sourceURL, nil
}

func parseHighscoresHTML(
	htmlBody string,
	worldName string,
	categorySlug string,
	vocation string,
	requestedPage int,
) (domain.HighscoresResult, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlBody))
	if err != nil {
		return domain.HighscoresResult{}, err
	}

	result := domain.HighscoresResult{
		World:         worldName,
		Category:      categorySlug,
		Vocation:      vocation,
		HighscoreAge:  parseHighscoreAge(doc),
		HighscoreList: make([]domain.Highscore, 0),
		HighscorePage: domain.HighscorePage{
			CurrentPage: requestedPage,
		},
	}

	table := findHighscoresTable(doc)
	if table == nil {
		result.HighscorePage.TotalPages = parseHighscoresTotalPages(doc, requestedPage)
		result.HighscorePage.TotalRecords = parseHighscoresTotalRecords(doc)
		return result, nil
	}

	table.Find("tr[bgcolor]").Each(func(_ int, row *goquery.Selection) {
		entry, ok := parseHighscoreRow(row)
		if !ok {
			return
		}
		result.HighscoreList = append(result.HighscoreList, entry)
	})

	result.HighscorePage.TotalPages = parseHighscoresTotalPages(doc, requestedPage)
	result.HighscorePage.TotalRecords = parseHighscoresTotalRecords(doc)
	return result, nil
}

func findHighscoresTable(doc *goquery.Document) *goquery.Selection {
	var target *goquery.Selection
	doc.Find("table.TableContent").EachWithBreak(func(_ int, table *goquery.Selection) bool {
		header := strings.ToLower(normalizeText(table.Find("tr.LabelH").First().Text()))
		if header == "" {
			header = strings.ToLower(normalizeText(table.Find("tr").First().Text()))
		}
		if strings.Contains(header, "rank") &&
			strings.Contains(header, "name") &&
			(strings.Contains(header, "points") ||
				strings.Contains(header, "skill level") ||
				strings.Contains(header, "score")) {
			target = table
			return false
		}
		return true
	})
	return target
}

func parseHighscoreRow(row *goquery.Selection) (domain.Highscore, bool) {
	cells := row.Find("td")
	if cells.Length() < 3 {
		return domain.Highscore{}, false
	}

	entry := domain.Highscore{
		Rank: parseInt(normalizeText(cells.Eq(0).Text())),
	}
	if entry.Rank <= 0 {
		return domain.Highscore{}, false
	}

	nameCell := cells.Eq(1)
	name := normalizeText(nameCell.Find("a[href*='subtopic=characters']").First().Text())
	if name == "" {
		name = normalizeText(nameCell.Find("a").First().Text())
	}
	if name == "" {
		return domain.Highscore{}, false
	}
	entry.Name = name

	if auctionURL, ok := nameCell.Find("a[href*='currentcharactertrades/'],a[href*='pastcharactertrades/']").First().Attr("href"); ok {
		entry.Traded = true
		entry.AuctionURL = strings.TrimSpace(auctionURL)
	}

	switch {
	case cells.Length() >= 7:
		entry.Title = normalizeText(cells.Eq(2).Text())
		entry.Vocation = normalizeText(cells.Eq(3).Text())
		entry.World = normalizeText(cells.Eq(4).Text())
		entry.Level = parseInt(normalizeText(cells.Eq(5).Text()))
		entry.Value = parseInt(normalizeText(cells.Eq(6).Text()))
	case cells.Length() >= 6:
		entry.Vocation = normalizeText(cells.Eq(2).Text())
		entry.World = normalizeText(cells.Eq(3).Text())
		entry.Level = parseInt(normalizeText(cells.Eq(4).Text()))
		entry.Value = parseInt(normalizeText(cells.Eq(5).Text()))
	default:
		entry.Value = parseInt(normalizeText(cells.Last().Text()))
	}

	return entry, true
}

func parseHighscoreAge(doc *goquery.Document) int {
	for _, row := range doc.Find(".CaptionContainer .Text").Nodes {
		text := normalizeText(goquery.NewDocumentFromNode(row).Text())
		if match := highscoresAgePattern.FindStringSubmatch(text); len(match) == 2 {
			return parseInt(match[1])
		}
	}
	return 0
}

func parseHighscoresTotalRecords(doc *goquery.Document) int {
	navText := normalizeText(doc.Find(".PageNavigation").First().Text())
	if match := highscoresResultsPattern.FindStringSubmatch(navText); len(match) == 2 {
		return parseInt(match[1])
	}
	return 0
}

func parseHighscoresTotalPages(doc *goquery.Document, fallbackCurrentPage int) int {
	maxPage := fallbackCurrentPage

	doc.Find(".PageNavigation a[href*='currentpage=']").Each(func(_ int, link *goquery.Selection) {
		if href, exists := link.Attr("href"); exists {
			if match := highscoresPageHrefPattern.FindStringSubmatch(href); len(match) == 2 {
				if page := parseInt(match[1]); page > maxPage {
					maxPage = page
				}
			}
		}

		pageText := normalizeText(link.Text())
		if page := parseInt(pageText); page > maxPage {
			maxPage = page
		}
	})

	doc.Find(".PageNavigation .CurrentPageLink").Each(func(_ int, current *goquery.Selection) {
		page := parseInt(normalizeText(current.Text()))
		if page > maxPage {
			maxPage = page
		}
	})

	if maxPage < 0 {
		return 0
	}
	return maxPage
}
