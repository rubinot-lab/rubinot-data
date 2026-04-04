package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/giovannirco/rubinot-data/internal/domain"
	"github.com/giovannirco/rubinot-data/internal/scraper"
	"github.com/giovannirco/rubinot-data/internal/validation"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	defaultRubinotBaseURL    = "https://rubinot.com.br"
	defaultScrapeTimeoutMS   = 120000
	defaultRequestTimeoutMS  = 45000
	defaultServiceVersion    = "dev"
)

var (
	resolvedBaseURL        string
	resolvedAssetsDir      string
	resolvedRequestTimeout int
	currentValidator       atomic.Pointer[validation.Validator]
	globalOC               *scraper.OptimizedClient
)

func NewRouter() (*gin.Engine, error) {
	resolvedBaseURL = getEnv("RUBINOT_BASE_URL", defaultRubinotBaseURL)
	resolvedAssetsDir = getEnv("ASSETS_DIR", "assets")
	resolvedRequestTimeout = getEnvInt("REQUEST_TIMEOUT_MS", defaultRequestTimeoutMS)

	ctx := context.Background()

	oc, ocErr := initOptimizedClient(ctx)
	if ocErr != nil {
		return nil, fmt.Errorf("init optimized client: %w", ocErr)
	}
	globalOC = oc

	validator, err := bootstrapValidatorV2(ctx, oc)
	if err != nil {
		return nil, err
	}
	currentValidator.Store(validator)
	startValidatorRefresh(oc)

	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(gin.Logger())
	router.Use(metricsMiddleware())

	router.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "rubinot-data api up"})
	})
	router.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "pong"})
	})
	router.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	router.GET("/readyz", func(c *gin.Context) {
		if oc != nil && !oc.Fetcher.IsReady() {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "not_ready", "reason": "cloudflare_rewarming"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	router.GET("/versions", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"service": "rubinot-data",
			"version": getEnv("APP_VERSION", defaultServiceVersion),
			"commit":  getEnv("APP_COMMIT", defaultAPICommit),
		})
	})
	router.GET("/openapi.json", docsSpec(router))
	router.GET("/docs", docsPage)
	router.GET("/metrics", gin.WrapH(promhttp.Handler()))

	registerV2Routes(router, oc)
	registerV1Routes(router, oc)

	return router, nil
}

func registerV1Routes(router *gin.Engine, oc *scraper.OptimizedClient) {
	v1 := router.Group("/v1")
	{
		v1.GET("/worlds", handleEndpoint(func(c *gin.Context) (endpointResult, error) {
			return v2GetWorlds(c, oc)
		}))
		v1.GET("/world/:name/details", handleEndpoint(func(c *gin.Context) (endpointResult, error) {
			return v2GetWorldDetails(c, getValidator(), oc)
		}))
		v1.GET("/world/:name/dashboard", handleEndpoint(func(c *gin.Context) (endpointResult, error) {
			return v2GetWorldDashboard(c, getValidator(), oc)
		}))
		v1.GET("/world/:name", handleEndpoint(func(c *gin.Context) (endpointResult, error) {
			return v2GetWorld(c, getValidator(), oc)
		}))
		v1.GET("/highscores/categories", handleEndpoint(func(c *gin.Context) (endpointResult, error) {
			return v2GetHighscoreCategories(c, oc)
		}))
		v1.GET("/highscores/:world", redirectHighscoresWorld)
		v1.GET("/highscores/:world/:category", redirectHighscoresCategory)
		v1.GET("/highscores/:world/:category/:vocation", redirectHighscoresVocation)
		v1.GET("/highscores/:world/:category/:vocation/all", handleEndpoint(func(c *gin.Context) (endpointResult, error) {
			return v1GetAllHighscores(c, getValidator(), oc)
		}))
		v1.GET("/highscores/:world/:category/:vocation/:page", handleEndpoint(func(c *gin.Context) (endpointResult, error) {
			return v2GetHighscores(c, getValidator(), oc)
		}))
		v1.GET("/killstatistics/:world", handleEndpoint(func(c *gin.Context) (endpointResult, error) {
			return v2GetKillstatistics(c, getValidator(), oc)
		}))
		v1.GET("/news/id/:news_id", handleEndpoint(func(c *gin.Context) (endpointResult, error) {
			return v2GetNewsByID(c, oc)
		}))
		v1.GET("/news/archive", handleEndpoint(func(c *gin.Context) (endpointResult, error) {
			return v2GetNewsArchive(c, oc)
		}))
		v1.GET("/news/latest", handleEndpoint(func(c *gin.Context) (endpointResult, error) {
			return v2GetNewsLatest(c, oc)
		}))
		v1.GET("/news/newsticker", handleEndpoint(func(c *gin.Context) (endpointResult, error) {
			return v2GetNewsTicker(c, oc)
		}))
		v1.GET("/boosted", handleEndpoint(func(c *gin.Context) (endpointResult, error) {
			return v2GetBoosted(c, oc)
		}))
		v1.GET("/maintenance", handleEndpoint(func(c *gin.Context) (endpointResult, error) {
			return v2GetMaintenance(c, oc)
		}))
		v1.GET("/geo-language", handleEndpoint(func(c *gin.Context) (endpointResult, error) {
			return v1GetGeoLanguage(c, oc)
		}))
		v1.GET("/outfit", getOutfit)
		v1.GET("/outfit/:name", getOutfitByCharacterName)
		v1.GET("/assets/creatures/:name", handleCreatureAsset(resolvedAssetsDir))
		v1.GET("/assets/items/:itemId", handleItemAsset(resolvedAssetsDir, "https://static.rubinot.com"))
		v1.GET("/assets/charms/:name", handleStaticAsset(resolvedAssetsDir, "charms", "image/png", ".png"))
		v1.GET("/assets/creature-types/:type", handleStaticAsset(resolvedAssetsDir, "creature-types", "image/png", ".png"))
		v1.GET("/events/schedule", handleEndpoint(func(c *gin.Context) (endpointResult, error) {
			return v1GetEventsSchedule(c, oc)
		}))
		v1.GET("/events/calendar", handleEndpoint(func(c *gin.Context) (endpointResult, error) {
			return v2GetEventsCalendar(c, oc)
		}))
		v1.GET("/auctions/current/all/details", handleEndpoint(func(c *gin.Context) (endpointResult, error) {
			return v1GetAllCurrentAuctionsDetails(c, oc)
		}))
		v1.GET("/auctions/current/:page/details", handleEndpoint(func(c *gin.Context) (endpointResult, error) {
			return v2GetCurrentAuctionDetails(c, oc)
		}))
		v1.GET("/auctions/current/all", handleEndpoint(func(c *gin.Context) (endpointResult, error) {
			return v2GetAllCurrentAuctions(c, oc)
		}))
		v1.GET("/auctions/current/:page", handleEndpoint(func(c *gin.Context) (endpointResult, error) {
			return v2GetCurrentAuctions(c, oc)
		}))
		v1.GET("/auctions/history/all/details", handleEndpoint(func(c *gin.Context) (endpointResult, error) {
			return v1GetAllAuctionHistoryDetails(c, oc)
		}))
		v1.GET("/auctions/history/:page/details", handleEndpoint(func(c *gin.Context) (endpointResult, error) {
			return v2GetAuctionHistoryDetails(c, oc)
		}))
		v1.GET("/auctions/history/all", handleEndpoint(func(c *gin.Context) (endpointResult, error) {
			return v2GetAllAuctionHistory(c, oc)
		}))
		v1.GET("/auctions/history/:page", handleEndpoint(func(c *gin.Context) (endpointResult, error) {
			return v2GetAuctionHistory(c, oc)
		}))
		v1.GET("/auctions/:id", handleEndpoint(func(c *gin.Context) (endpointResult, error) {
			return v2GetAuctionDetail(c, oc)
		}))
		v1.GET("/deaths/:world/all", handleEndpoint(func(c *gin.Context) (endpointResult, error) {
			return v2GetAllDeaths(c, getValidator(), oc)
		}))
		v1.GET("/deaths/:world", handleEndpoint(func(c *gin.Context) (endpointResult, error) {
			return v2GetDeaths(c, getValidator(), oc)
		}))
		v1.GET("/banishments/:world/all", handleEndpoint(func(c *gin.Context) (endpointResult, error) {
			return v2GetAllBanishments(c, getValidator(), oc)
		}))
		v1.GET("/banishments/:world", handleEndpoint(func(c *gin.Context) (endpointResult, error) {
			return v2GetBanishments(c, getValidator(), oc)
		}))
		v1.GET("/transfers/all", handleEndpoint(func(c *gin.Context) (endpointResult, error) {
			return v2GetAllTransfers(c, getValidator(), oc)
		}))
		v1.GET("/transfers", handleEndpoint(func(c *gin.Context) (endpointResult, error) {
			return v2GetTransfers(c, getValidator(), oc)
		}))
		v1.GET("/character/:name", handleEndpoint(func(c *gin.Context) (endpointResult, error) {
			return v2GetCharacter(c, oc)
		}))
		v1.GET("/guild/:name", handleEndpoint(func(c *gin.Context) (endpointResult, error) {
			return v2GetGuild(c, oc)
		}))
		v1.GET("/guilds/:world/all/details", handleEndpoint(func(c *gin.Context) (endpointResult, error) {
			return v2GetAllGuildsDetails(c, getValidator(), oc)
		}))
		v1.GET("/guilds/:world/all", handleEndpoint(func(c *gin.Context) (endpointResult, error) {
			return v2GetAllGuilds(c, getValidator(), oc)
		}))
		v1.GET("/guilds/:world/:page", handleEndpoint(func(c *gin.Context) (endpointResult, error) {
			return v2GetGuilds(c, getValidator(), oc)
		}))
		v1.GET("/guilds/:world", handleEndpoint(func(c *gin.Context) (endpointResult, error) {
			return v2GetGuilds(c, getValidator(), oc)
		}))
		v1.GET("/house/:world/:house_id", handleEndpoint(deprecatedHousesEndpoint))
		v1.GET("/houses/towns", handleEndpoint(deprecatedHousesEndpoint))
		v1.GET("/houses/:world/:town", handleEndpoint(deprecatedHousesEndpoint))

		if getEnv("ENABLE_RAW_PROXY", "") == "true" {
			v1.POST("/upstream/raw", handleEndpoint(func(c *gin.Context) (endpointResult, error) {
				return v1PostUpstreamRaw(c, oc)
			}))
		}
		v1.POST("/characters/batch", handleEndpoint(func(c *gin.Context) (endpointResult, error) {
			return v2PostCharactersBatch(c, oc)
		}))
		v1.POST("/characters/compare", handleEndpoint(func(c *gin.Context) (endpointResult, error) {
			return v1PostCharactersCompare(c, oc)
		}))
		v1.POST("/highscores/:category/cross-world", handleEndpoint(func(c *gin.Context) (endpointResult, error) {
			return v1PostHighscoresCrossWorld(c, getValidator(), oc)
		}))
		v1.POST("/highscores/multi-category", handleEndpoint(func(c *gin.Context) (endpointResult, error) {
			return v1PostHighscoresMultiCategory(c, getValidator(), oc)
		}))
		v1.POST("/guilds/batch", handleEndpoint(func(c *gin.Context) (endpointResult, error) {
			return v2PostGuildsBatch(c, oc)
		}))
		v1.POST("/auctions/filter", handleEndpoint(func(c *gin.Context) (endpointResult, error) {
			return v1PostAuctionsFilter(c, oc)
		}))
		v1.POST("/killstatistics/batch", handleEndpoint(func(c *gin.Context) (endpointResult, error) {
			return v2PostKillstatisticsBatch(c, getValidator(), oc)
		}))
		v1.GET("/bans", handleEndpoint(func(c *gin.Context) (endpointResult, error) {
			return v1GetBans(c, oc)
		}))
		v1.GET("/news/all", handleEndpoint(func(c *gin.Context) (endpointResult, error) {
			return v1GetNewsAll(c, oc)
		}))
	}
}

func initOptimizedClient(ctx context.Context) (*scraper.OptimizedClient, error) {
	cdpURL := getEnv("CDP_URL", "")
	if cdpURL == "" {
		return nil, fmt.Errorf("CDP_URL not set")
	}
	poolSize := getEnvInt("CDP_POOL_SIZE", 4)
	pool := scraper.NewCDPPool(cdpURL, resolvedBaseURL, poolSize)
	if err := pool.Init(ctx); err != nil {
		return nil, fmt.Errorf("cdp pool init: %w", err)
	}
	cacheTTLSeconds := getEnvInt("CDP_CACHE_TTL_SECONDS", 5)
	fetcher := scraper.NewCachedFetcher(pool, time.Duration(cacheTTLSeconds)*time.Second)
	fetcher.SetLastWarmAt(time.Now())
	return scraper.NewOptimizedClient(fetcher), nil
}

func bootstrapValidatorV2(ctx context.Context, oc *scraper.OptimizedClient) (*validation.Validator, error) {
	refreshStarted := time.Now()

	worlds, _, err := scraper.V2FetchWorlds(ctx, oc, resolvedBaseURL)
	if err != nil {
		scraper.ValidatorRefresh.WithLabelValues("error").Inc()
		scraper.ValidatorRefreshDuration.Observe(time.Since(refreshStarted).Seconds())
		return nil, err
	}

	validationWorlds := make([]validation.World, 0, len(worlds.Worlds))
	for _, w := range worlds.Worlds {
		name := strings.TrimSpace(w.Name)
		if name == "" {
			continue
		}
		id := w.ID
		if id <= 0 {
			if mapped, ok := scraper.WorldIDByName(name); ok {
				id = mapped
			}
		}
		if id <= 0 {
			log.Printf("world %q skipped: no ID from API or mapping", name)
			continue
		}
		validationWorlds = append(validationWorlds, validation.World{ID: id, Name: name})
	}
	if len(validationWorlds) == 0 {
		scraper.ValidatorRefresh.WithLabelValues("error").Inc()
		scraper.ValidatorRefreshDuration.Observe(time.Since(refreshStarted).Seconds())
		return nil, validation.NewError(validation.ErrorUpstreamUnknown, "validator world bootstrap failed: no worlds discovered", nil)
	}

	mappings := make([]scraper.WorldMapping, 0, len(validationWorlds))
	for _, w := range validationWorlds {
		mappings = append(mappings, scraper.WorldMapping{ID: w.ID, Name: w.Name})
	}
	scraper.UpdateWorldMappings(mappings)

	validator := validation.NewValidator(validationWorlds)
	categories := validator.AllCategories()

	scraper.ValidatorRefresh.WithLabelValues("ok").Inc()
	scraper.ValidatorRefreshDuration.Observe(time.Since(refreshStarted).Seconds())
	scraper.WorldsDiscovered.Set(float64(len(validationWorlds)))
	scraper.DiscoveredCount.WithLabelValues("worlds").Set(float64(len(validationWorlds)))
	scraper.DiscoveredCount.WithLabelValues("towns").Set(0)
	scraper.DiscoveredCount.WithLabelValues("categories").Set(float64(len(categories)))
	return validator, nil
}

func startValidatorRefresh(oc *scraper.OptimizedClient) {
	interval := time.Duration(getEnvInt("VALIDATOR_REFRESH_INTERVAL_SECONDS", 3600)) * time.Second
	ticker := time.NewTicker(interval)
	go func() {
		defer ticker.Stop()
		for range ticker.C {
			validator, err := bootstrapValidatorV2(context.Background(), oc)
			if err != nil {
				log.Printf("validator refresh failed: %v", err)
				continue
			}
			currentValidator.Store(validator)
		}
	}()
}

func getValidator() *validation.Validator {
	validator := currentValidator.Load()
	if validator == nil {
		panic("validator is not initialized")
	}
	return validator
}

func deprecatedHousesEndpoint(_ *gin.Context) (endpointResult, error) {
	return endpointResult{}, validation.NewError(
		validation.ErrorEndpointDeprecated,
		"houses endpoints are deprecated: house data is available via /v1/character/:name",
		nil,
	)
}

func redirectHighscoresWorld(c *gin.Context) {
	world := strings.TrimSpace(c.Param("world"))
	location := fmt.Sprintf("/v1/highscores/%s/experience/all/1", url.PathEscape(world))
	c.Redirect(http.StatusFound, location)
}

func redirectHighscoresCategory(c *gin.Context) {
	world := strings.TrimSpace(c.Param("world"))
	category := strings.TrimSpace(c.Param("category"))
	location := fmt.Sprintf("/v1/highscores/%s/%s/all/1", url.PathEscape(world), url.PathEscape(category))
	c.Redirect(http.StatusFound, location)
}

func redirectHighscoresVocation(c *gin.Context) {
	world := strings.TrimSpace(c.Param("world"))
	category := strings.TrimSpace(c.Param("category"))
	vocation := strings.TrimSpace(c.Param("vocation"))
	location := fmt.Sprintf("/v1/highscores/%s/%s/%s/1", url.PathEscape(world), url.PathEscape(category), url.PathEscape(vocation))
	c.Redirect(http.StatusFound, location)
}

func getOutfit(c *gin.Context) {
	oc := globalOC
	body, contentType, sourceURL, err := scraper.V2FetchOutfitImage(c.Request.Context(), oc, resolvedBaseURL, c.Request.URL.RawQuery)
	if err != nil {
		writeOutfitError(c, err, []string{sourceURL})
		return
	}
	writeOutfitResponse(c, body, contentType, sourceURL)
}

func getOutfitByCharacterName(c *gin.Context) {
	oc := globalOC
	characterInput := strings.TrimSpace(c.Param("name"))
	canonicalName, validationErr := validation.IsCharacterNameValid(characterInput)
	if validationErr != nil {
		writeOutfitError(c, validationErr, []string{})
		return
	}

	character, characterSourceURL, err := scraper.V2FetchCharacter(c.Request.Context(), oc, resolvedBaseURL, canonicalName)
	if err != nil {
		writeOutfitError(c, err, []string{characterSourceURL})
		return
	}

	if character.CharacterInfo.Outfit == nil {
		writeOutfitError(c, validation.NewError(validation.ErrorUpstreamUnknown, "character outfit is not available", nil), []string{characterSourceURL})
		return
	}

	query := c.Request.URL.Query()
	setOutfitDefaultParam(query, "type", "looktype", strconv.Itoa(character.CharacterInfo.Outfit.LookType))
	setOutfitDefaultParam(query, "head", "lookhead", strconv.Itoa(character.CharacterInfo.Outfit.LookHead))
	setOutfitDefaultParam(query, "body", "lookbody", strconv.Itoa(character.CharacterInfo.Outfit.LookBody))
	setOutfitDefaultParam(query, "legs", "looklegs", strconv.Itoa(character.CharacterInfo.Outfit.LookLegs))
	setOutfitDefaultParam(query, "feet", "lookfeet", strconv.Itoa(character.CharacterInfo.Outfit.LookFeet))
	setOutfitDefaultParam(query, "addons", "lookaddons", strconv.Itoa(character.CharacterInfo.Outfit.LookAddons))

	body, contentType, outfitSourceURL, err := scraper.V2FetchOutfitImage(c.Request.Context(), oc, resolvedBaseURL, query.Encode())
	if err != nil {
		writeOutfitError(c, err, []string{characterSourceURL, outfitSourceURL})
		return
	}

	writeOutfitResponse(c, body, contentType, outfitSourceURL)
}

func setOutfitDefaultParam(values url.Values, primary, legacy, fallback string) {
	if strings.TrimSpace(values.Get(primary)) == "" && strings.TrimSpace(values.Get(legacy)) == "" {
		values.Set(primary, fallback)
	}
}

func writeOutfitResponse(c *gin.Context, body []byte, contentType, sourceURL string) {
	formattedBody, formattedContentType, err := maybeFormatOutfitImage(c.Query("format"), body, contentType)
	if err != nil {
		writeOutfitError(c, validation.NewError(validation.ErrorUpstreamUnknown, fmt.Sprintf("failed to render outfit image: %v", err), err), []string{sourceURL})
		return
	}
	c.Header("Cache-Control", "public, max-age=300")
	c.Header("X-Source-URL", sourceURL)
	c.Data(http.StatusOK, formattedContentType, formattedBody)
}

func writeOutfitError(c *gin.Context, err error, sources []string) {
	errorCode := resolveErrorCode(err)
	httpCode := statusCodeFromErrorCode(errorCode)
	message := resolveErrorMessage(errorCode, err)
	if httpCode == http.StatusBadRequest {
		route := c.FullPath()
		if route == "" {
			route = "unknown"
		}
		validationRejections.WithLabelValues(route, strconv.Itoa(errorCode)).Inc()
	}
	c.JSON(httpCode, errorEnvelope(httpCode, errorCode, message, sources))
}

func isAllWorldsToken(value string) bool {
	return strings.EqualFold(strings.TrimSpace(value), "all")
}

func getEnvInt(key string, fallback int) int {
	raw := strings.TrimSpace(getEnv(key, ""))
	if raw == "" {
		return fallback
	}

	parsed, err := strconv.Atoi(raw)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}

func compareCharacters(a, b domain.CharacterResult) domain.ComparisonSignals {
	sameAccount := false
	if len(a.OtherCharacters) > 0 && len(b.OtherCharacters) > 0 {
		aNames := make(map[string]bool, len(a.OtherCharacters))
		for _, oc := range a.OtherCharacters {
			aNames[strings.ToLower(strings.TrimSpace(oc.Name))] = true
		}
		bNames := make(map[string]bool, len(b.OtherCharacters))
		for _, oc := range b.OtherCharacters {
			bNames[strings.ToLower(strings.TrimSpace(oc.Name))] = true
		}
		if len(aNames) == len(bNames) && len(aNames) > 0 {
			match := true
			for k := range aNames {
				if !bNames[k] {
					match = false
					break
				}
			}
			sameAccount = match
		}
	}

	sameVipTime := a.CharacterInfo.VIPTime > 0 && a.CharacterInfo.VIPTime == b.CharacterInfo.VIPTime
	sameAccountCreated := a.CharacterInfo.AccountCreated != "" && a.CharacterInfo.AccountCreated == b.CharacterInfo.AccountCreated
	sameCreated := a.CharacterInfo.Created != "" && a.CharacterInfo.Created == b.CharacterInfo.Created
	sameLoyaltyPoints := a.CharacterInfo.LoyaltyPoints > 0 && a.CharacterInfo.LoyaltyPoints == b.CharacterInfo.LoyaltyPoints

	sameHouse := false
	if a.CharacterInfo.House != nil && b.CharacterInfo.House != nil {
		sameHouse = a.CharacterInfo.House.HouseID == b.CharacterInfo.House.HouseID && a.CharacterInfo.House.HouseID > 0
	}

	sameGuild := false
	if a.CharacterInfo.Guild != nil && b.CharacterInfo.Guild != nil {
		sameGuild = a.CharacterInfo.Guild.ID == b.CharacterInfo.Guild.ID && a.CharacterInfo.Guild.ID > 0
	}

	sameOutfit := false
	if a.CharacterInfo.Outfit != nil && b.CharacterInfo.Outfit != nil {
		oa, ob := a.CharacterInfo.Outfit, b.CharacterInfo.Outfit
		sameOutfit = oa.LookType == ob.LookType && oa.LookHead == ob.LookHead &&
			oa.LookBody == ob.LookBody && oa.LookLegs == ob.LookLegs &&
			oa.LookFeet == ob.LookFeet && oa.LookAddons == ob.LookAddons
	}

	return domain.ComparisonSignals{
		SameAccount:        sameAccount,
		SameVipTime:        sameVipTime,
		SameAccountCreated: sameAccountCreated,
		SameHouse:          sameHouse,
		SameGuild:          sameGuild,
		SameOutfit:         sameOutfit,
		SameCreated:        sameCreated,
		SameLoyaltyPoints:  sameLoyaltyPoints,
	}
}

func v1GetAllHighscores(c *gin.Context, validator *validation.Validator, oc *scraper.OptimizedClient) (endpointResult, error) {
	worldInput := strings.TrimSpace(c.Param("world"))
	categoryInput := strings.TrimSpace(c.Param("category"))
	vocationInput := strings.TrimSpace(c.Param("vocation"))

	category, categoryOK := validator.ResolveHighscoreCategory(categoryInput)
	if !categoryOK {
		return endpointResult{}, validation.NewError(validation.ErrorHighscoreCategoryDoesNotExist, "highscore category does not exist", nil)
	}

	vocation, vocationOK := validator.ResolveHighscoreVocation(vocationInput)
	if !vocationOK {
		return endpointResult{}, validation.NewError(validation.ErrorVocationDoesNotExist, "vocation does not exist", nil)
	}

	if isAllWorldsToken(worldInput) {
		worlds := validator.AllWorlds()
		results, sources, err := scraper.V2FetchHighscoresBatch(c.Request.Context(), oc, resolvedBaseURL, worlds, category, vocation)
		if err != nil {
			return endpointResult{Sources: sources}, err
		}
		totalRecords := 0
		totalEntries := 0
		for _, r := range results {
			totalRecords += r.HighscorePage.TotalRecords
			totalEntries += len(r.HighscoreList)
		}
		return endpointResult{
			PayloadKey: "highscores",
			Payload: domain.HighscoresByWorldResult{
				World:        "all",
				Category:     category.Slug,
				Vocation:     vocation.Name,
				TotalWorlds:  len(results),
				TotalRecords: totalRecords,
				TotalEntries: totalEntries,
				Worlds:       results,
			},
			Sources: sources,
		}, nil
	}

	return v2GetHighscores(c, validator, oc)
}

func v1GetGeoLanguage(c *gin.Context, oc *scraper.OptimizedClient) (endpointResult, error) {
	result, sourceURL, err := scraper.V2FetchGeoLanguage(c.Request.Context(), oc, resolvedBaseURL)
	if err != nil {
		return endpointResult{Sources: []string{sourceURL}}, err
	}
	return endpointResult{
		PayloadKey: "geo_language",
		Payload:    result,
		Sources:    []string{sourceURL},
	}, nil
}

func v1GetEventsSchedule(c *gin.Context, oc *scraper.OptimizedClient) (endpointResult, error) {
	return v2GetEventsCalendar(c, oc)
}

func v1GetAllCurrentAuctionsDetails(c *gin.Context, oc *scraper.OptimizedClient) (endpointResult, error) {
	result, sources, err := scraper.V2FetchAllCurrentAuctionDetails(c.Request.Context(), oc, resolvedBaseURL)
	if err != nil {
		return endpointResult{Sources: sources}, err
	}
	return endpointResult{PayloadKey: "auctions", Payload: result, Sources: sources}, nil
}

func v1GetAllAuctionHistoryDetails(c *gin.Context, oc *scraper.OptimizedClient) (endpointResult, error) {
	result, sources, err := scraper.V2FetchAllAuctionHistoryDetails(c.Request.Context(), oc, resolvedBaseURL)
	if err != nil {
		return endpointResult{Sources: sources}, err
	}
	return endpointResult{PayloadKey: "auctions", Payload: result, Sources: sources}, nil
}

func v1PostUpstreamRaw(c *gin.Context, oc *scraper.OptimizedClient) (endpointResult, error) {
	var req struct {
		Path string `json:"path" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		return endpointResult{}, validation.NewError(validation.ErrorInvalidParameter, "path is required", err)
	}
	cleaned := strings.TrimSpace(req.Path)
	if !strings.HasPrefix(cleaned, "/api/") {
		return endpointResult{}, validation.NewError(validation.ErrorInvalidParameter, "path must start with /api/", nil)
	}

	sourceURL := fmt.Sprintf("%s%s", strings.TrimRight(resolvedBaseURL, "/"), cleaned)
	rawBody, err := oc.Fetcher.FetchJSON(c.Request.Context(), sourceURL)
	if err != nil {
		return endpointResult{Sources: []string{sourceURL}}, err
	}

	var raw json.RawMessage
	if err := json.Unmarshal([]byte(rawBody), &raw); err != nil {
		return endpointResult{Sources: []string{sourceURL}}, err
	}
	return endpointResult{
		PayloadKey: "raw",
		Payload:    raw,
		Sources:    []string{sourceURL},
	}, nil
}

func v1PostCharactersCompare(c *gin.Context, oc *scraper.OptimizedClient) (endpointResult, error) {
	var req struct {
		Names []string `json:"names" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		return endpointResult{}, validation.NewError(validation.ErrorInvalidParameter, "names is required", err)
	}
	if len(req.Names) != 2 {
		return endpointResult{}, validation.NewError(validation.ErrorInvalidParameter, "exactly 2 names required", nil)
	}

	results, sources, err := scraper.V2FetchCharactersBatch(c.Request.Context(), oc, resolvedBaseURL, req.Names)
	if err != nil {
		return endpointResult{Sources: sources}, err
	}
	if len(results) < 2 {
		return endpointResult{Sources: sources}, validation.NewError(validation.ErrorEntityNotFound, "one or both characters not found", nil)
	}

	signals := compareCharacters(results[0], results[1])
	return endpointResult{
		PayloadKey: "comparison",
		Payload: domain.ComparisonResult{
			Characters: results,
			Signals:    signals,
		},
		Sources: sources,
	}, nil
}

func v1PostHighscoresCrossWorld(c *gin.Context, validator *validation.Validator, oc *scraper.OptimizedClient) (endpointResult, error) {
	categoryInput := strings.TrimSpace(c.Param("category"))
	category, categoryOK := validator.ResolveHighscoreCategory(categoryInput)
	if !categoryOK {
		return endpointResult{}, validation.NewError(validation.ErrorHighscoreCategoryDoesNotExist, "highscore category does not exist", nil)
	}

	var req struct {
		Vocations []int `json:"vocations" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		return endpointResult{}, validation.NewError(validation.ErrorInvalidParameter, "vocations is required", err)
	}
	if len(req.Vocations) == 0 || len(req.Vocations) > 15 {
		return endpointResult{}, validation.NewError(validation.ErrorInvalidParameter, "vocations must have 1-15 entries", nil)
	}

	worlds := validator.AllWorlds()
	allSources := make([]string, 0)
	resultMap := make(map[string][]domain.HighscoresResult)
	for _, vocID := range req.Vocations {
		voc, vocOK := validator.ResolveHighscoreVocation(strconv.Itoa(vocID))
		if !vocOK {
			continue
		}
		results, sources, err := scraper.V2FetchHighscoresBatch(c.Request.Context(), oc, resolvedBaseURL, worlds, category, voc)
		if err != nil {
			return endpointResult{Sources: append(allSources, sources...)}, err
		}
		allSources = append(allSources, sources...)
		resultMap[strconv.Itoa(vocID)] = results
	}

	return endpointResult{
		PayloadKey: "highscores",
		Payload:    resultMap,
		Sources:    allSources,
	}, nil
}

func v1PostHighscoresMultiCategory(c *gin.Context, validator *validation.Validator, oc *scraper.OptimizedClient) (endpointResult, error) {
	var req struct {
		Categories []string `json:"categories" binding:"required"`
		Vocations  []int    `json:"vocations" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		return endpointResult{}, validation.NewError(validation.ErrorInvalidParameter, "categories and vocations are required", err)
	}
	if len(req.Categories) > 10 {
		return endpointResult{}, validation.NewError(validation.ErrorInvalidParameter, "max 10 categories", nil)
	}
	if len(req.Vocations) == 0 || len(req.Vocations) > 15 {
		return endpointResult{}, validation.NewError(validation.ErrorInvalidParameter, "vocations must have 1-15 entries", nil)
	}

	worlds := validator.AllWorlds()
	allSources := make([]string, 0)
	resultMap := make(map[string]map[string][]domain.HighscoresResult)

	for _, catInput := range req.Categories {
		category, categoryOK := validator.ResolveHighscoreCategory(strings.TrimSpace(catInput))
		if !categoryOK {
			return endpointResult{}, validation.NewError(validation.ErrorHighscoreCategoryDoesNotExist, fmt.Sprintf("category not found: %s", catInput), nil)
		}
		vocMap := make(map[string][]domain.HighscoresResult)
		for _, vocID := range req.Vocations {
			voc, vocOK := validator.ResolveHighscoreVocation(strconv.Itoa(vocID))
			if !vocOK {
				continue
			}
			results, sources, err := scraper.V2FetchHighscoresBatch(c.Request.Context(), oc, resolvedBaseURL, worlds, category, voc)
			if err != nil {
				return endpointResult{Sources: append(allSources, sources...)}, err
			}
			allSources = append(allSources, sources...)
			vocMap[strconv.Itoa(vocID)] = results
		}
		resultMap[category.Slug] = vocMap
	}

	return endpointResult{
		PayloadKey: "highscores",
		Payload:    resultMap,
		Sources:    allSources,
	}, nil
}

func v1PostAuctionsFilter(c *gin.Context, oc *scraper.OptimizedClient) (endpointResult, error) {
	var req struct {
		Vocation int `json:"vocation"`
		MinLevel int `json:"minLevel"`
		MaxLevel int `json:"maxLevel"`
		World    int `json:"world"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		return endpointResult{}, validation.NewError(validation.ErrorInvalidParameter, "invalid filter parameters", err)
	}

	query := url.Values{}
	if req.Vocation > 0 {
		query.Set("vocation", strconv.Itoa(req.Vocation))
	}
	if req.MinLevel > 0 {
		query.Set("minLevel", strconv.Itoa(req.MinLevel))
	}
	if req.MaxLevel > 0 {
		query.Set("maxLevel", strconv.Itoa(req.MaxLevel))
	}
	if req.World > 0 {
		query.Set("world", strconv.Itoa(req.World))
	}

	sourceURL := fmt.Sprintf("%s/api/bazaar?%s", strings.TrimRight(resolvedBaseURL, "/"), query.Encode())
	rawBody, err := oc.Fetcher.FetchJSON(c.Request.Context(), sourceURL)
	if err != nil {
		return endpointResult{Sources: []string{sourceURL}}, err
	}

	var raw json.RawMessage
	if err := json.Unmarshal([]byte(rawBody), &raw); err != nil {
		return endpointResult{Sources: []string{sourceURL}}, err
	}
	return endpointResult{
		PayloadKey: "auctions",
		Payload:    raw,
		Sources:    []string{sourceURL},
	}, nil
}

func v1GetBans(c *gin.Context, oc *scraper.OptimizedClient) (endpointResult, error) {
	sourceURL := fmt.Sprintf("%s/api/bans", strings.TrimRight(resolvedBaseURL, "/"))
	var raw any
	if err := oc.FetchJSON(c.Request.Context(), sourceURL, &raw); err != nil {
		return endpointResult{Sources: []string{sourceURL}}, err
	}
	return endpointResult{
		PayloadKey: "bans",
		Payload:    raw,
		Sources:    []string{sourceURL},
	}, nil
}

func v1GetNewsAll(c *gin.Context, oc *scraper.OptimizedClient) (endpointResult, error) {
	sourceURL := fmt.Sprintf("%s/api/news", strings.TrimRight(resolvedBaseURL, "/"))
	var raw any
	if err := oc.FetchJSON(c.Request.Context(), sourceURL, &raw); err != nil {
		return endpointResult{Sources: []string{sourceURL}}, err
	}
	return endpointResult{
		PayloadKey: "news",
		Payload:    raw,
		Sources:    []string{sourceURL},
	}, nil
}
