package scraper

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/giovannirco/rubinot-data/internal/domain"
	"github.com/giovannirco/rubinot-data/internal/validation"
	"go.opentelemetry.io/otel/attribute"
)

const (
	auctionTypeCurrent = "current"
	auctionTypeHistory = "history"
)

var (
	auctionIDPattern            = regexp.MustCompile(`(?:currentcharactertrades|pastcharactertrades)/(\d+)`)
	auctionSummaryLinePattern   = regexp.MustCompile(`(?i)Level:\s*(\d+)\s*\|\s*Vocation:\s*([^|]+)\|\s*(Male|Female)\s*\|\s*World:\s*([^\n<]+)`)
	auctionResultsPattern       = regexp.MustCompile(`(?i)results:\s*([\d.,]+)`)
	auctionPagePattern          = regexp.MustCompile(`(?i)currentpage=(\d+)`)
	auctionDatePattern          = regexp.MustCompile(`(?i)Auction (Start|End):\s*([A-Za-z]{3}\s+\d{1,2}\s+\d{4},\s+\d{2}:\d{2}\s+BRA)`)
	auctionBidTypeValuePattern  = regexp.MustCompile(`(?i)(Current|Winning|Minimum)\s+Bid:\s*([\d.,]+)`)
	auctionNotFoundTextPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)an internal error has occurred`),
		regexp.MustCompile(`(?i)requested url .+ was not found on this server`),
		regexp.MustCompile(`(?i)no character auctions found`),
	}
)

func FetchCurrentAuctions(
	ctx context.Context,
	baseURL string,
	page int,
	opts FetchOptions,
) (domain.AuctionsResult, string, error) {
	return fetchAuctionsList(ctx, baseURL, auctionTypeCurrent, page, opts)
}

func FetchAuctionHistory(
	ctx context.Context,
	baseURL string,
	page int,
	opts FetchOptions,
) (domain.AuctionsResult, string, error) {
	return fetchAuctionsList(ctx, baseURL, auctionTypeHistory, page, opts)
}

func fetchAuctionsList(
	ctx context.Context,
	baseURL string,
	auctionType string,
	page int,
	opts FetchOptions,
) (domain.AuctionsResult, string, error) {
	ctx, span := tracer.Start(ctx, "scraper.fetchAuctionsList")
	defer span.End()

	sourceURL := buildAuctionsListURL(baseURL, auctionType, page)
	client := NewClient(opts)

	endpointMetric := "auctions_" + auctionType
	span.SetAttributes(
		attribute.String("rubinot.endpoint", endpointMetric),
		attribute.String("rubinot.source_url", sourceURL),
		attribute.String("rubinot.auction_type", auctionType),
		attribute.Int("rubinot.page", page),
	)

	started := time.Now()
	htmlBody, err := client.Fetch(ctx, sourceURL)
	scrapeDuration.WithLabelValues(endpointMetric).Observe(time.Since(started).Seconds())
	if err != nil {
		scrapeRequests.WithLabelValues(endpointMetric, "error").Inc()
		return domain.AuctionsResult{}, sourceURL, err
	}
	scrapeRequests.WithLabelValues(endpointMetric, "ok").Inc()

	parseStarted := time.Now()
	result, parseErr := parseAuctionsListHTML(auctionType, page, htmlBody)
	parseDuration.WithLabelValues(endpointMetric).Observe(time.Since(parseStarted).Seconds())
	if parseErr != nil {
		return domain.AuctionsResult{}, sourceURL, parseErr
	}
	return result, sourceURL, nil
}

func FetchAuctionDetail(
	ctx context.Context,
	baseURL string,
	auctionID int,
	opts FetchOptions,
) (domain.AuctionDetail, []string, error) {
	ctx, span := tracer.Start(ctx, "scraper.FetchAuctionDetail")
	defer span.End()

	idStr := fmt.Sprintf("%d", auctionID)
	sourceURLs := []string{
		fmt.Sprintf("%s/?currentcharactertrades/%s", strings.TrimRight(baseURL, "/"), idStr),
		fmt.Sprintf("%s/?pastcharactertrades/%s", strings.TrimRight(baseURL, "/"), idStr),
	}
	client := NewClient(opts)

	span.SetAttributes(
		attribute.String("rubinot.endpoint", "auction_detail"),
		attribute.Int("rubinot.auction_id", auctionID),
	)

	for _, sourceURL := range sourceURLs {
		span.SetAttributes(attribute.String("rubinot.source_url", sourceURL))

		started := time.Now()
		htmlBody, err := client.Fetch(ctx, sourceURL)
		scrapeDuration.WithLabelValues("auction_detail").Observe(time.Since(started).Seconds())
		if err != nil {
			if shouldIgnoreAuctionSourceError(err) {
				continue
			}
			scrapeRequests.WithLabelValues("auction_detail", "error").Inc()
			return domain.AuctionDetail{}, sourceURLs, err
		}

		isPastURL := strings.Contains(sourceURL, "pastcharactertrades")
		parseStarted := time.Now()
		detail, parseErr := parseAuctionDetailHTML(auctionID, htmlBody, isPastURL)
		parseDuration.WithLabelValues("auction_detail").Observe(time.Since(parseStarted).Seconds())
		if parseErr != nil {
			var validationErr validation.Error
			if errors.As(parseErr, &validationErr) && validationErr.Code() == validation.ErrorEntityNotFound {
				continue
			}
			scrapeRequests.WithLabelValues("auction_detail", "error").Inc()
			return domain.AuctionDetail{}, sourceURLs, parseErr
		}

		scrapeRequests.WithLabelValues("auction_detail", "ok").Inc()
		return detail, []string{sourceURL}, nil
	}

	scrapeRequests.WithLabelValues("auction_detail", "error").Inc()
	return domain.AuctionDetail{}, sourceURLs, validation.NewError(validation.ErrorEntityNotFound, "auction not found", nil)
}

func buildAuctionsListURL(baseURL string, auctionType string, page int) string {
	trimmedBase := strings.TrimRight(baseURL, "/")
	if page < 1 {
		page = 1
	}

	// TODO: AMBIGUOUS — contract says /currentcharactertrades?currentpage=N, but upstream currently requires ?subtopic=currentcharactertrades&currentpage=N.
	switch auctionType {
	case auctionTypeHistory:
		if page == 1 {
			return fmt.Sprintf("%s/pastcharactertrades", trimmedBase)
		}
		return fmt.Sprintf("%s/?subtopic=pastcharactertrades&currentpage=%d", trimmedBase, page)
	default:
		if page == 1 {
			return fmt.Sprintf("%s/currentcharactertrades", trimmedBase)
		}
		return fmt.Sprintf("%s/?subtopic=currentcharactertrades&currentpage=%d", trimmedBase, page)
	}
}

func parseAuctionsListHTML(auctionType string, page int, htmlBody string) (domain.AuctionsResult, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlBody))
	if err != nil {
		return domain.AuctionsResult{}, err
	}

	if page < 1 {
		page = 1
	}
	result := domain.AuctionsResult{
		Type:    auctionType,
		Page:    page,
		Entries: make([]domain.AuctionEntry, 0),
	}

	if strings.Contains(strings.ToLower(doc.Text()), "no character auctions found") {
		result.TotalPages = maxInt(parseAuctionsTotalPages(doc), 1)
		return result, nil
	}

	doc.Find("div.Auction").Each(func(_ int, auction *goquery.Selection) {
		entry, ok := parseAuctionListEntry(auction, auctionType)
		if !ok {
			return
		}
		result.Entries = append(result.Entries, entry)
	})

	totalResults := parseAuctionsTotalResults(doc)
	if totalResults == 0 {
		totalResults = len(result.Entries)
	}
	result.TotalResults = totalResults

	totalPages := parseAuctionsTotalPages(doc)
	if totalPages == 0 {
		if page > 1 {
			totalPages = page
		} else {
			totalPages = 1
		}
	}
	result.TotalPages = totalPages

	return result, nil
}

func parseAuctionListEntry(auction *goquery.Selection, auctionType string) (domain.AuctionEntry, bool) {
	header := auction.Find(".AuctionHeader").First()
	if header.Length() == 0 {
		return domain.AuctionEntry{}, false
	}

	name := normalizeText(header.Find(".AuctionCharacterName a").First().Text())
	if name == "" {
		name = normalizeText(header.Find(".AuctionCharacterName").First().Text())
	}
	if name == "" {
		return domain.AuctionEntry{}, false
	}

	summaryLine := normalizeText(header.Text())
	summaryMatch := auctionSummaryLinePattern.FindStringSubmatch(summaryLine)
	if len(summaryMatch) != 5 {
		return domain.AuctionEntry{}, false
	}

	level := parseInt(summaryMatch[1])
	vocation := normalizeText(summaryMatch[2])
	sex := normalizeText(summaryMatch[3])
	world := normalizeText(summaryMatch[4])
	if level <= 0 || vocation == "" || sex == "" || world == "" {
		return domain.AuctionEntry{}, false
	}

	auctionID := extractAuctionID(header.Find("a[href*='charactertrades/']").First())

	auctionBodyText := normalizeText(auction.Find(".ShortAuctionData").Text())
	bidType, bidValue := parseAuctionBidTypeAndValue(auctionBodyText)
	auctionEnd := parseAuctionDateField(auctionBodyText, "End")

	status := "active"
	if auctionType == auctionTypeHistory {
		status = "ended"
	}
	if strings.Contains(strings.ToLower(auction.Text()), "finished") || strings.EqualFold(bidType, "winning") {
		status = "ended"
	}
	if strings.Contains(strings.ToLower(auction.Text()), "cancelled") || strings.Contains(strings.ToLower(auction.Text()), "canceled") {
		status = "cancelled"
	}

	return domain.AuctionEntry{
		AuctionID:     auctionID,
		CharacterName: name,
		Level:         level,
		Vocation:      vocation,
		Sex:           sex,
		World:         world,
		BidType:       strings.ToLower(bidType),
		BidValue:      bidValue,
		AuctionEnd:    auctionEnd,
		Status:        status,
	}, true
}

func parseAuctionDetailHTML(auctionID int, htmlBody string, isPast bool) (domain.AuctionDetail, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlBody))
	if err != nil {
		return domain.AuctionDetail{}, err
	}

	if isAuctionNotFoundPage(doc) {
		return domain.AuctionDetail{}, validation.NewError(validation.ErrorEntityNotFound, "auction not found", nil)
	}

	auction := doc.Find("div.Auction").First()
	if auction.Length() == 0 {
		return domain.AuctionDetail{}, validation.NewError(validation.ErrorEntityNotFound, "auction not found", nil)
	}

	header := auction.Find(".AuctionHeader").First()
	name := normalizeText(header.Find(".AuctionCharacterName").First().Text())
	summaryMatch := auctionSummaryLinePattern.FindStringSubmatch(normalizeText(header.Text()))
	if name == "" || len(summaryMatch) != 5 {
		return domain.AuctionDetail{}, validation.NewError(validation.ErrorEntityNotFound, "auction not found", nil)
	}

	defaultStatus := "active"
	if isPast {
		defaultStatus = "ended"
	}

	detail := domain.AuctionDetail{
		AuctionID:     auctionID,
		CharacterName: name,
		Level:         parseInt(summaryMatch[1]),
		Vocation:      normalizeText(summaryMatch[2]),
		Sex:           normalizeText(summaryMatch[3]),
		World:         normalizeText(summaryMatch[4]),
		Status:        defaultStatus,
	}

	bodyText := normalizeText(auction.Find(".AuctionBody").Text())
	detail.BidType, detail.BidValue = parseAuctionBidTypeAndValue(bodyText)
	detail.BidType = strings.ToLower(detail.BidType)
	detail.AuctionStart = parseAuctionDateField(bodyText, "Start")
	detail.AuctionEnd = parseAuctionDateField(bodyText, "End")

	auctionText := strings.ToLower(auction.Text())
	if strings.Contains(auctionText, "finished") || detail.BidType == "winning" {
		detail.Status = "ended"
	}
	if strings.Contains(auctionText, "cancelled") || strings.Contains(auctionText, "canceled") {
		detail.Status = "cancelled"
	}

	return detail, nil
}

func parseAuctionsTotalResults(doc *goquery.Document) int {
	match := auctionResultsPattern.FindStringSubmatch(doc.Text())
	if len(match) != 2 {
		return 0
	}
	return parseInt(match[1])
}

func parseAuctionsTotalPages(doc *goquery.Document) int {
	maxPage := 0
	doc.Find("a[href*='currentpage=']").Each(func(_ int, link *goquery.Selection) {
		href, ok := link.Attr("href")
		if !ok {
			return
		}
		match := auctionPagePattern.FindStringSubmatch(href)
		if len(match) != 2 {
			return
		}
		page := parseInt(match[1])
		if page > maxPage {
			maxPage = page
		}
	})
	return maxPage
}

func parseAuctionBidTypeAndValue(raw string) (string, int) {
	match := auctionBidTypeValuePattern.FindStringSubmatch(raw)
	if len(match) != 3 {
		return "", 0
	}
	return normalizeText(match[1]), parseInt(match[2])
}

func parseAuctionDateField(raw string, field string) string {
	matches := auctionDatePattern.FindAllStringSubmatch(raw, -1)
	for _, match := range matches {
		if len(match) != 3 {
			continue
		}
		if !strings.EqualFold(normalizeText(match[1]), field) {
			continue
		}
		if parsed, err := parseAuctionDateTimeToUTC(match[2]); err == nil {
			return parsed
		}
	}
	return ""
}

func parseAuctionDateTimeToUTC(raw string) (string, error) {
	clean := strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(raw), "BRA"))
	parsed, err := time.ParseInLocation("Jan 2 2006, 15:04", clean, rubinotBrazilLocation)
	if err != nil {
		return "", err
	}
	return parsed.UTC().Format(time.RFC3339), nil
}

func extractAuctionID(link *goquery.Selection) int {
	if link == nil || link.Length() == 0 {
		return 0
	}
	href, ok := link.Attr("href")
	if !ok {
		return 0
	}
	match := auctionIDPattern.FindStringSubmatch(href)
	if len(match) != 2 {
		return 0
	}
	return parseInt(match[1])
}

func isAuctionNotFoundPage(doc *goquery.Document) bool {
	pageText := strings.ToLower(normalizeText(doc.Text()))
	if strings.Contains(pageText, "auction details") && doc.Find("div.Auction").Length() > 0 {
		return false
	}

	for _, pattern := range auctionNotFoundTextPatterns {
		if pattern.MatchString(pageText) {
			return true
		}
	}
	return false
}

func shouldIgnoreAuctionSourceError(err error) bool {
	var validationErr validation.Error
	if !errors.As(err, &validationErr) {
		return false
	}
	if validationErr.Code() != validation.ErrorUpstreamUnknown {
		return false
	}
	return strings.Contains(strings.ToLower(validationErr.Error()), "404")
}

func maxInt(left, right int) int {
	if left > right {
		return left
	}
	return right
}
