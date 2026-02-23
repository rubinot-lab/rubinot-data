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
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

var (
	houseBedsPattern       = regexp.MustCompile(`(?i)up to (\d+) beds?`)
	houseSizePattern       = regexp.MustCompile(`(?i)size of (\d+)\s+square meters`)
	houseRentPattern       = regexp.MustCompile(`(?i)monthly rent is ([\d,.]+)\s+gold`)
	houseBidPattern        = regexp.MustCompile(`(?i)highest bid so far is ([\d,.]+)\s+gold`)
	houseAuctionEndPattern = regexp.MustCompile(`(?i)auction will end at ([^.]+)\.`)
	housePaidUntilPattern  = regexp.MustCompile(`(?i)paid the rent until ([^.]+)\.`)
	houseMovingDatePattern = regexp.MustCompile(`(?i)move out on ([^.]+)\.`)
	houseLevelPattern      = regexp.MustCompile(`(?i)level (\d+)`)
	houseVocationPattern   = regexp.MustCompile(`(?i)\b(knight|paladin|sorcerer|druid|monk)\b`)
)

func FetchHouse(
	ctx context.Context,
	baseURL string,
	worldName string,
	worldID int,
	houseID int,
	towns []validation.Town,
	opts FetchOptions,
) (domain.HouseResult, string, error) {
	ctx, span := tracer.Start(ctx, "scraper.FetchHouse")
	defer span.End()

	client := NewClient(opts)
	lastSourceURL := ""
	span.SetAttributes(
		attribute.String("rubinot.endpoint", "house"),
		attribute.String("rubinot.world", worldName),
		attribute.Int("rubinot.world_id", worldID),
		attribute.Int("rubinot.house_id", houseID),
	)

	for _, town := range towns {
		if ctx.Err() != nil {
			return domain.HouseResult{}, lastSourceURL, validation.NewError(validation.ErrorFlareSolverrTimeout, fmt.Sprintf("house lookup cancelled: %v", ctx.Err()), ctx.Err())
		}

		sourceURL := buildHouseDetailsURL(baseURL, worldID, town.ID, houseID)
		lastSourceURL = sourceURL

		started := time.Now()
		htmlBody, fetchErr := client.Fetch(ctx, sourceURL)
		scrapeDuration.WithLabelValues("house").Observe(time.Since(started).Seconds())
		if fetchErr != nil {
			scrapeRequests.WithLabelValues("house", "error").Inc()
			return domain.HouseResult{}, sourceURL, fetchErr
		}
		scrapeRequests.WithLabelValues("house", "ok").Inc()

		parseStarted := time.Now()
		house, notFound, parseErr := parseHouseHTML(htmlBody, houseID, worldName, town.Name)
		parseDuration.WithLabelValues("house").Observe(time.Since(parseStarted).Seconds())
		if parseErr != nil {
			return domain.HouseResult{}, sourceURL, parseErr
		}
		if notFound {
			continue
		}

		return house, sourceURL, nil
	}

	if lastSourceURL == "" {
		lastSourceURL = buildHouseDetailsURL(baseURL, worldID, 0, houseID)
	}
	return domain.HouseResult{}, lastSourceURL, validation.NewError(validation.ErrorEntityNotFound, "house not found", nil)
}

func buildHouseDetailsURL(baseURL string, worldID int, townID int, houseID int) string {
	return fmt.Sprintf(
		"%s/?subtopic=houses&page=view&world=%d&town=%d&state=&type=houses&order=name&houseid=%d",
		strings.TrimRight(baseURL, "/"),
		worldID,
		townID,
		houseID,
	)
}

func parseHouseHTML(html string, houseID int, worldName, townName string) (domain.HouseResult, bool, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return domain.HouseResult{}, false, err
	}

	content := doc.Find("#houses .BoxContent")
	if content.Length() == 0 {
		return domain.HouseResult{}, true, nil
	}

	normalizedText := strings.Join(strings.Fields(content.Text()), " ")
	if strings.Contains(strings.ToLower(normalizedText), "no information about this house found") {
		return domain.HouseResult{}, true, nil
	}

	house := domain.HouseResult{
		HouseID: houseID,
		World:   worldName,
		Town:    townName,
		Name:    strings.TrimSpace(content.Find("b").First().Text()),
		Size:    findRegexInt(houseSizePattern, normalizedText),
		Beds:    findRegexInt(houseBedsPattern, normalizedText),
		Rent:    findRegexInt(houseRentPattern, normalizedText),
	}
	if house.Name == "" {
		return domain.HouseResult{}, true, nil
	}

	lowerText := strings.ToLower(normalizedText)
	switch {
	case strings.Contains(lowerText, "currently being auctioned"):
		house.Status = "auctioned"
		house.Auction = &domain.HouseAuction{
			CurrentBid: findRegexInt(houseBidPattern, normalizedText),
			EndDate:    findRegexString(houseAuctionEndPattern, normalizedText),
			NoBidYet:   strings.Contains(lowerText, "no bid has been submitted so far"),
		}
		if bidder := strings.TrimSpace(content.Find("a[href*='subtopic=characters']").First().Text()); bidder != "" {
			house.Auction.Bidder = bidder
		}
	case strings.Contains(lowerText, "currently vacant") || strings.Contains(lowerText, "currently unoccupied"):
		house.Status = "vacant"
	case strings.Contains(lowerText, "being transferred to"):
		house.Status = "transfer"
		house.Owner = extractHouseOwner(content, normalizedText)
	case strings.Contains(lowerText, "move out on"):
		house.Status = "moving"
		house.Owner = extractHouseOwner(content, normalizedText)
		house.Owner.MovingDate = findRegexString(houseMovingDatePattern, normalizedText)
	case strings.Contains(lowerText, "has been rented by"):
		house.Status = "rented"
		house.Owner = extractHouseOwner(content, normalizedText)
	default:
		return domain.HouseResult{}, false, validation.NewError(validation.ErrorUpstreamUnknown, "unable to determine house status", nil)
	}

	return house, false, nil
}

func extractHouseOwner(content *goquery.Selection, normalizedText string) *domain.HouseOwner {
	owner := &domain.HouseOwner{
		Name:      strings.TrimSpace(content.Find("a[href*='subtopic=characters']").First().Text()),
		PaidUntil: findRegexString(housePaidUntilPattern, normalizedText),
		Level:     findRegexInt(houseLevelPattern, normalizedText),
	}
	if owner.Name == "" {
		return nil
	}
	owner.Vocation = findRegexString(houseVocationPattern, normalizedText)
	if owner.Vocation != "" {
		owner.Vocation = cases.Title(language.English).String(strings.ToLower(owner.Vocation))
	}
	return owner
}

func findRegexInt(pattern *regexp.Regexp, text string) int {
	match := pattern.FindStringSubmatch(text)
	if len(match) != 2 {
		return 0
	}
	return parseInt(match[1])
}

func findRegexString(pattern *regexp.Regexp, text string) string {
	match := pattern.FindStringSubmatch(text)
	if len(match) != 2 {
		return ""
	}
	return strings.TrimSpace(match[1])
}
