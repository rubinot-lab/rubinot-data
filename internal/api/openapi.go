package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"

	"github.com/gin-gonic/gin"
)

type openAPIDocument struct {
	OpenAPI    string                                 `json:"openapi"`
	Info       openAPIInfo                            `json:"info"`
	Servers    []openAPIServer                        `json:"servers"`
	Tags       []openAPITag                           `json:"tags,omitempty"`
	Paths      map[string]map[string]openAPIOperation `json:"paths"`
	Components map[string]map[string]map[string]any   `json:"components,omitempty"`
}

type openAPIInfo struct {
	Title       string `json:"title"`
	Version     string `json:"version"`
	Description string `json:"description"`
}

type openAPIServer struct {
	URL         string `json:"url"`
	Description string `json:"description"`
}

type openAPITag struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

type openAPIOperation struct {
	Summary     string                     `json:"summary,omitempty"`
	Tags        []string                   `json:"tags,omitempty"`
	Parameters  []openAPIParameter         `json:"parameters,omitempty"`
	RequestBody *openAPIRequestBody        `json:"requestBody,omitempty"`
	Responses   map[string]openAPIResponse `json:"responses"`
}

type openAPIParameter struct {
	Name     string        `json:"name"`
	In       string        `json:"in"`
	Required bool          `json:"required"`
	Schema   openAPISchema `json:"schema"`
}

type openAPISchema struct {
	Type    string `json:"type"`
	Enum    []any  `json:"enum,omitempty"`
	Minimum *int   `json:"minimum,omitempty"`
}

type openAPIResponse struct {
	Description string `json:"description"`
}

type openAPIRequestBody struct {
	Description string                      `json:"description,omitempty"`
	Required    bool                        `json:"required"`
	Content     map[string]openAPIMediaType `json:"content"`
}

type openAPIMediaType struct {
	Schema map[string]any `json:"schema,omitempty"`
}

type openAPIOperationOverride struct {
	Summary     string
	Tags        []string
	Parameters  []openAPIParameter
	RequestBody *openAPIRequestBody
	Responses   map[string]openAPIResponse
}

var openAPIOperationOverrides = map[string]openAPIOperationOverride{
	"GET /": {
		Summary:   "Service status",
		Responses: map[string]openAPIResponse{"200": {Description: "OK"}},
	},
	"GET /ping": {
		Summary:   "Pong check",
		Responses: map[string]openAPIResponse{"200": {Description: "OK"}},
	},
	"GET /healthz": {
		Summary:   "Kubernetes health check",
		Responses: map[string]openAPIResponse{"200": {Description: "OK"}},
	},
	"GET /readyz": {
		Summary:   "Kubernetes ready check",
		Responses: map[string]openAPIResponse{"200": {Description: "OK"}},
	},
	"GET /versions": {
		Summary: "Service version metadata",
	},
	"GET /metrics": {
		Summary: "Prometheus metrics",
	},
	"GET /openapi.json": {
		Summary: "OpenAPI spec",
	},
	"GET /docs": {
		Summary: "Interactive docs",
	},
	"POST /v1/upstream/raw": {
		Summary: "Proxy a raw upstream /api payload",
		Tags:    []string{"upstream"},
		RequestBody: jsonRequestBody(
			"Raw upstream endpoint request",
			map[string]any{
				"type":                 "object",
				"additionalProperties": false,
				"required":             []string{"path"},
				"properties": map[string]any{
					"path": map[string]any{
						"type":        "string",
						"description": "Path to proxy, must start with /api/",
						"pattern":     "^/api/.*",
					},
				},
			},
		),
		Responses: standardPostResponses("Raw upstream payload"),
	},
	"POST /v1/characters/batch": {
		Summary: "Batch character lookup",
		Tags:    []string{"characters"},
		RequestBody: jsonRequestBody(
			"Character names to fetch in one request",
			map[string]any{
				"type":                 "object",
				"additionalProperties": false,
				"required":             []string{"names"},
				"properties": map[string]any{
					"names": map[string]any{
						"type":        "array",
						"description": "Character names (max 1000)",
						"maxItems":    1000,
						"items": map[string]any{
							"type": "string",
						},
					},
				},
			},
		),
		Responses: standardPostResponses("Batch character results"),
	},
	"POST /v1/characters/compare": {
		Summary: "Compare two characters and return similarity signals",
		Tags:    []string{"characters"},
		RequestBody: jsonRequestBody(
			"Exactly two character names to compare",
			map[string]any{
				"type":                 "object",
				"additionalProperties": false,
				"required":             []string{"names"},
				"properties": map[string]any{
					"names": map[string]any{
						"type":        "array",
						"description": "Exactly two character names",
						"minItems":    2,
						"maxItems":    2,
						"items": map[string]any{
							"type": "string",
						},
					},
				},
			},
		),
		Responses: standardPostResponsesWithNotFound("Character comparison result", "One or both characters were not found"),
	},
	"POST /v1/highscores/{category}/cross-world": {
		Summary: "Fetch cross-world highscores for one category and multiple vocations",
		Tags:    []string{"highscores"},
		RequestBody: jsonRequestBody(
			"Vocation IDs to aggregate across all worlds",
			map[string]any{
				"type":                 "object",
				"additionalProperties": false,
				"required":             []string{"vocations"},
				"properties": map[string]any{
					"vocations": map[string]any{
						"type":        "array",
						"description": "Vocation IDs (1-15 entries)",
						"minItems":    1,
						"maxItems":    15,
						"items": map[string]any{
							"type": "integer",
						},
					},
				},
			},
		),
		Responses: standardPostResponses("Cross-world highscores grouped by world and vocation"),
	},
	"POST /v1/highscores/multi-category": {
		Summary: "Fetch cross-world highscores for multiple categories",
		Tags:    []string{"highscores"},
		RequestBody: jsonRequestBody(
			"Categories and vocation IDs to aggregate across all worlds",
			map[string]any{
				"type":                 "object",
				"additionalProperties": false,
				"required":             []string{"categories", "vocations"},
				"properties": map[string]any{
					"categories": map[string]any{
						"type":        "array",
						"description": "Highscore category slugs (max 10)",
						"maxItems":    10,
						"items": map[string]any{
							"type": "string",
						},
					},
					"vocations": map[string]any{
						"type":        "array",
						"description": "Vocation IDs (1-15 entries)",
						"minItems":    1,
						"maxItems":    15,
						"items": map[string]any{
							"type": "integer",
						},
					},
				},
			},
		),
		Responses: standardPostResponses("Cross-world highscores grouped by category, world, and vocation"),
	},
	"POST /v1/guilds/batch": {
		Summary: "Batch guild lookup",
		Tags:    []string{"guilds"},
		RequestBody: jsonRequestBody(
			"Guild names to fetch in one request",
			map[string]any{
				"type":                 "object",
				"additionalProperties": false,
				"required":             []string{"names"},
				"properties": map[string]any{
					"names": map[string]any{
						"type":        "array",
						"description": "Guild names (max 20)",
						"maxItems":    20,
						"items": map[string]any{
							"type": "string",
						},
					},
				},
			},
		),
		Responses: standardPostResponses("Batch guild results"),
	},
	"POST /v1/auctions/filter": {
		Summary: "Filter auctions by vocation, level range, and world",
		Tags:    []string{"auctions"},
		RequestBody: jsonRequestBody(
			"Auction filter parameters",
			map[string]any{
				"type":                 "object",
				"additionalProperties": false,
				"properties": map[string]any{
					"vocation": map[string]any{"type": "integer"},
					"minLevel": map[string]any{"type": "integer"},
					"maxLevel": map[string]any{"type": "integer"},
					"world":    map[string]any{"type": "integer"},
				},
			},
		),
		Responses: standardPostResponses("Filtered auction payload"),
	},
	"POST /v1/killstatistics/batch": {
		Summary: "Batch killstatistics lookup for multiple worlds",
		Tags:    []string{"killstatistics"},
		RequestBody: jsonRequestBody(
			"World names to fetch killstatistics for",
			map[string]any{
				"type":                 "object",
				"additionalProperties": false,
				"required":             []string{"worlds"},
				"properties": map[string]any{
					"worlds": map[string]any{
						"type":        "array",
						"description": "World names (max 20)",
						"maxItems":    20,
						"items": map[string]any{
							"type": "string",
						},
					},
				},
			},
		),
		Responses: standardPostResponses("Killstatistics results for requested worlds"),
	},
	"GET /v1/outfit": {
		Summary:    "Outfit image proxy",
		Tags:       []string{"outfit"},
		Parameters: outfitQueryParams(),
		Responses:  map[string]openAPIResponse{"200": {Description: "Outfit image"}},
	},
	"GET /v1/outfit/{name}": {
		Summary:    "Outfit image by character name",
		Tags:       []string{"outfit"},
		Parameters: outfitByNameQueryParams(),
		Responses:  map[string]openAPIResponse{"200": {Description: "Outfit image"}},
	},
	"GET /v1/houses/towns": {
		Summary:   "Deprecated houses towns endpoint",
		Responses: map[string]openAPIResponse{"410": {Description: "Gone"}},
	},
	"GET /v1/house/{world}/{house_id}": {
		Summary:   "Deprecated house endpoint",
		Responses: map[string]openAPIResponse{"410": {Description: "Gone"}},
	},
	"GET /v1/houses/{world}/{town}": {
		Summary:   "Deprecated houses endpoint",
		Responses: map[string]openAPIResponse{"410": {Description: "Gone"}},
	},

	"GET /v2/worlds": {
		Summary: "List all worlds with status and player counts",
		Tags:    []string{"v2/worlds"},
	},
	"GET /v2/world/{name}": {
		Summary: "World detail with online players (use 'all' for every world)",
		Tags:    []string{"v2/worlds"},
	},
	"GET /v2/world/{name}/details": {
		Summary: "World + full character info for each online player",
		Tags:    []string{"v2/worlds"},
	},
	"GET /v2/world/{name}/dashboard": {
		Summary: "World overview: players, recent deaths, killstatistics",
		Tags:    []string{"v2/worlds"},
	},
	"GET /v2/highscores/{world}/{category}/{vocation}": {
		Summary: "Highscores for a world/category/vocation combo",
		Tags:    []string{"v2/highscores"},
	},
	"GET /v2/killstatistics/{world}": {
		Summary: "Kill statistics for a world (use 'all' for every world)",
		Tags:    []string{"v2/killstatistics"},
	},
	"GET /v2/deaths/{world}": {
		Summary: "Recent deaths for a world (single page)",
		Tags:    []string{"v2/deaths"},
		Parameters: []openAPIParameter{
			intQueryParam("page", nil),
			intQueryParam("level", nil),
			stringQueryParam("pvp", []any{"true", "false"}),
			stringQueryParam("guild", nil),
		},
	},
	"GET /v2/deaths/{world}/all": {
		Summary: "All deaths across all pages for a world",
		Tags:    []string{"v2/deaths"},
		Parameters: []openAPIParameter{
			intQueryParam("level", nil),
			stringQueryParam("pvp", []any{"true", "false"}),
			stringQueryParam("guild", nil),
		},
	},
	"GET /v2/banishments/{world}": {
		Summary: "Banishments for a world (single page)",
		Tags:    []string{"v2/banishments"},
		Parameters: []openAPIParameter{
			intQueryParam("page", nil),
		},
	},
	"GET /v2/banishments/{world}/all": {
		Summary: "All banishments across all pages for a world",
		Tags:    []string{"v2/banishments"},
	},
	"GET /v2/transfers": {
		Summary: "World transfers (single page)",
		Tags:    []string{"v2/transfers"},
		Parameters: []openAPIParameter{
			stringQueryParam("world", nil),
			intQueryParam("level", nil),
			intQueryParam("page", nil),
		},
	},
	"GET /v2/transfers/all": {
		Summary: "All transfers across all pages",
		Tags:    []string{"v2/transfers"},
		Parameters: []openAPIParameter{
			stringQueryParam("world", nil),
			intQueryParam("level", nil),
		},
	},
	"GET /v2/character/{name}": {
		Summary: "Character lookup by name",
		Tags:    []string{"v2/characters"},
	},
	"GET /v2/guild/{name}": {
		Summary: "Guild detail by name",
		Tags:    []string{"v2/guilds"},
	},
	"GET /v2/guilds/{world}": {
		Summary: "Guild list for a world (single page, use 'all' for every world)",
		Tags:    []string{"v2/guilds"},
		Parameters: []openAPIParameter{
			intQueryParam("page", nil),
		},
	},
	"GET /v2/guilds/{world}/all": {
		Summary: "All guilds across all pages for a world",
		Tags:    []string{"v2/guilds"},
	},
	"GET /v2/boosted": {
		Summary: "Today's boosted boss and creature",
		Tags:    []string{"v2/boosted"},
	},
	"GET /v2/maintenance": {
		Summary: "Server maintenance status",
		Tags:    []string{"v2/maintenance"},
	},
	"GET /v2/auctions/current/{page}": {
		Summary: "Current auctions (single page)",
		Tags:    []string{"v2/auctions"},
	},
	"GET /v2/auctions/current/all": {
		Summary: "All current auctions across all pages",
		Tags:    []string{"v2/auctions"},
	},
	"GET /v2/auctions/history/{page}": {
		Summary: "Auction history (single page)",
		Tags:    []string{"v2/auctions"},
	},
	"GET /v2/auctions/history/all": {
		Summary: "All auction history across all pages",
		Tags:    []string{"v2/auctions"},
	},
	"GET /v2/auctions/{id}": {
		Summary: "Auction detail by ID",
		Tags:    []string{"v2/auctions"},
	},
	"GET /v2/news/id/{news_id}": {
		Summary: "News article or ticker by ID",
		Tags:    []string{"v2/news"},
	},
	"GET /v2/news/archive": {
		Summary: "News archive (default last 90 days)",
		Tags:    []string{"v2/news"},
		Parameters: []openAPIParameter{
			intQueryParam("days", nil),
		},
	},
	"GET /v2/news/latest": {
		Summary: "Latest news articles",
		Tags:    []string{"v2/news"},
	},
	"GET /v2/news/newsticker": {
		Summary: "News ticker entries",
		Tags:    []string{"v2/news"},
	},
	"GET /v2/outfit": {
		Summary:    "Outfit image proxy",
		Tags:       []string{"v2/outfit"},
		Parameters: outfitQueryParams(),
		Responses:  map[string]openAPIResponse{"200": {Description: "Outfit image"}},
	},
	"GET /v2/outfit/{name}": {
		Summary:    "Outfit image by character name",
		Tags:       []string{"v2/outfit"},
		Parameters: outfitByNameQueryParams(),
		Responses:  map[string]openAPIResponse{"200": {Description: "Outfit image"}},
	},
}

func docsSpec(router *gin.Engine) gin.HandlerFunc {
	return func(c *gin.Context) {
		spec, err := buildOpenAPISpec(router)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to build openapi spec: %v", err)})
			return
		}

		c.Header("Cache-Control", "public, max-age=300")
		c.Data(http.StatusOK, "application/json", spec)
	}
}

func buildOpenAPISpec(router *gin.Engine) ([]byte, error) {
	routes := append([]gin.RouteInfo(nil), router.Routes()...)
	sort.Slice(routes, func(i, j int) bool {
		if routes[i].Path == routes[j].Path {
			return routes[i].Method < routes[j].Method
		}
		return routes[i].Path < routes[j].Path
	})

	paths := make(map[string]map[string]openAPIOperation, len(routes))
	for _, route := range routes {
		openAPIPath := ginPathToOpenAPIPath(route.Path)
		operation := openAPIOperation{
			Summary:    defaultOperationSummary(route.Method, openAPIPath),
			Tags:       defaultOperationTags(openAPIPath),
			Parameters: pathParametersForGinPath(route.Path),
			Responses:  map[string]openAPIResponse{"200": {Description: "Success"}},
		}

		if override, ok := openAPIOperationOverrides[strings.ToUpper(route.Method)+" "+openAPIPath]; ok {
			if override.Summary != "" {
				operation.Summary = override.Summary
			}
			if len(override.Tags) > 0 {
				operation.Tags = override.Tags
			}
			if override.Parameters != nil {
				operation.Parameters = override.Parameters
			}
			if override.RequestBody != nil {
				operation.RequestBody = override.RequestBody
			}
			if len(override.Responses) > 0 {
				operation.Responses = override.Responses
			}
		}

		if _, ok := paths[openAPIPath]; !ok {
			paths[openAPIPath] = make(map[string]openAPIOperation)
		}
		paths[openAPIPath][strings.ToLower(route.Method)] = operation
	}

	spec := openAPIDocument{
		OpenAPI: "3.0.3",
		Info: openAPIInfo{
			Title:       "rubinot-data API",
			Version:     getEnv("APP_VERSION", defaultServiceVersion),
			Description: "Tibia scraper and data aggregation API.",
		},
		Servers: []openAPIServer{{
			URL:         "/",
			Description: "Current environment",
		}},
		Tags:  buildOpenAPITags(paths),
		Paths: paths,
		Components: map[string]map[string]map[string]any{
			"schemas": {
				"ErrorEnvelope": {
					"type": "object",
					"properties": map[string]any{
						"information": map[string]any{"type": "object"},
						"status":      map[string]any{"type": "integer"},
					},
				},
				"SuccessEnvelope": {
					"type": "object",
					"properties": map[string]any{
						"information": map[string]any{"type": "object"},
						"payload":     map[string]any{"type": "object"},
					},
				},
			},
		},
	}

	return json.Marshal(spec)
}

func ginPathToOpenAPIPath(ginPath string) string {
	if ginPath == "/" {
		return ginPath
	}

	parts := strings.Split(strings.Trim(ginPath, "/"), "/")
	for i, part := range parts {
		if strings.HasPrefix(part, ":") {
			parts[i] = "{" + strings.TrimPrefix(part, ":") + "}"
			continue
		}
		if strings.HasPrefix(part, "*") {
			parts[i] = "{" + strings.TrimPrefix(part, "*") + "}"
		}
	}
	return "/" + strings.Join(parts, "/")
}

func pathParametersForGinPath(ginPath string) []openAPIParameter {
	if ginPath == "/" {
		return nil
	}

	parts := strings.Split(strings.Trim(ginPath, "/"), "/")
	params := make([]openAPIParameter, 0, len(parts))
	for _, part := range parts {
		if strings.HasPrefix(part, ":") {
			params = append(params, pathParam(strings.TrimPrefix(part, ":")))
			continue
		}
		if strings.HasPrefix(part, "*") {
			params = append(params, pathParam(strings.TrimPrefix(part, "*")))
		}
	}
	if len(params) == 0 {
		return nil
	}
	return params
}

func defaultOperationSummary(method, openAPIPath string) string {
	return strings.ToUpper(method) + " " + openAPIPath
}

func defaultOperationTags(openAPIPath string) []string {
	if !strings.HasPrefix(openAPIPath, "/v1/") && !strings.HasPrefix(openAPIPath, "/v2/") {
		return []string{"system"}
	}

	parts := strings.Split(strings.Trim(openAPIPath, "/"), "/")
	if len(parts) < 2 {
		return []string{"api"}
	}

	version := parts[0]
	resource := strings.Trim(parts[1], "{}")
	if resource == "" {
		return []string{"api"}
	}
	if version == "v2" {
		return []string{version + "/" + resource}
	}
	return []string{resource}
}

func buildOpenAPITags(paths map[string]map[string]openAPIOperation) []openAPITag {
	unique := make(map[string]struct{})
	for _, methods := range paths {
		for _, operation := range methods {
			for _, tag := range operation.Tags {
				tag = strings.TrimSpace(tag)
				if tag == "" {
					continue
				}
				unique[tag] = struct{}{}
			}
		}
	}

	names := make([]string, 0, len(unique))
	for tag := range unique {
		names = append(names, tag)
	}
	sort.Strings(names)

	tags := make([]openAPITag, 0, len(names))
	for _, name := range names {
		tags = append(tags, openAPITag{
			Name:        name,
			Description: tagDescription(name),
		})
	}
	return tags
}

func tagDescription(name string) string {
	switch name {
	case "api":
		return "Generic API endpoints."
	case "auctions":
		return "Character bazaar auctions and filters."
	case "banishments":
		return "Banishments and punishments endpoints."
	case "bans":
		return "Account bans endpoint."
	case "boosted":
		return "Boosted boss and creature data."
	case "character", "characters":
		return "Character lookup, batch fetch, and comparison endpoints."
	case "deaths":
		return "Character deaths endpoints."
	case "events":
		return "Server event schedule and calendar endpoints."
	case "geo-language":
		return "Geo language detection endpoint."
	case "guild", "guilds":
		return "Guild lookup and batch endpoints."
	case "healthz", "readyz":
		return "Health and readiness checks."
	case "highscores":
		return "Highscores endpoints and cross-world aggregation."
	case "house", "houses":
		return "House endpoints (deprecated in this API)."
	case "killstatistics":
		return "Killstatistics endpoints and batch aggregation."
	case "maintenance":
		return "Maintenance mode status endpoint."
	case "metrics":
		return "Prometheus metrics endpoint."
	case "news":
		return "News, archives, and ticker endpoints."
	case "outfit":
		return "Outfit image and GIF rendering endpoints."
	case "ping":
		return "Simple liveness check endpoint."
	case "system":
		return "Service-level system endpoints."
	case "transfers":
		return "World transfer endpoints."
	case "upstream":
		return "Raw upstream proxy endpoints."
	case "versions":
		return "Service version metadata endpoint."
	case "world", "worlds":
		return "World and world-level aggregate endpoints."
	case "v2/worlds":
		return "v2 world endpoints — CDP-optimized with connection pooling, singleflight dedup, and 5s TTL cache. Use 'all' as world name for fan-out."
	case "v2/characters":
		return "v2 character lookup — CDP-optimized."
	case "v2/guilds":
		return "v2 guild endpoints — supports paginated fan-out via /all."
	case "v2/highscores":
		return "v2 highscores — single fetch per world/category/vocation."
	case "v2/killstatistics":
		return "v2 kill statistics — batch fan-out for 'all' worlds."
	case "v2/deaths":
		return "v2 deaths — supports paginated fan-out via /all."
	case "v2/banishments":
		return "v2 banishments — supports paginated fan-out via /all."
	case "v2/transfers":
		return "v2 transfers — supports paginated fan-out via /all."
	case "v2/boosted":
		return "v2 boosted boss and creature."
	case "v2/maintenance":
		return "v2 server maintenance status."
	case "v2/auctions":
		return "v2 character bazaar — supports paginated fan-out via /all."
	case "v2/news":
		return "v2 news, archive, and ticker endpoints."
	case "v2/outfit":
		return "v2 outfit image rendering."
	default:
		return ""
	}
}

func jsonRequestBody(description string, schema map[string]any) *openAPIRequestBody {
	return &openAPIRequestBody{
		Description: description,
		Required:    true,
		Content: map[string]openAPIMediaType{
			"application/json": {
				Schema: schema,
			},
		},
	}
}

func standardPostResponses(successDescription string) map[string]openAPIResponse {
	return responseSet(map[string]string{
		"200": successDescription,
		"400": "Validation error.",
		"502": "Upstream fetch or processing error.",
		"503": "Upstream maintenance mode.",
		"504": "Upstream timeout.",
	})
}

func standardPostResponsesWithNotFound(successDescription string, notFoundDescription string) map[string]openAPIResponse {
	responses := standardPostResponses(successDescription)
	responses["404"] = openAPIResponse{Description: notFoundDescription}
	return responses
}

func responseSet(values map[string]string) map[string]openAPIResponse {
	responses := make(map[string]openAPIResponse, len(values))
	for code, description := range values {
		responses[code] = openAPIResponse{Description: description}
	}
	return responses
}

func outfitQueryParams() []openAPIParameter {
	return []openAPIParameter{
		intQueryParam("type", nil),
		intQueryParam("head", nil),
		intQueryParam("body", nil),
		intQueryParam("legs", nil),
		intQueryParam("feet", nil),
		intQueryParam("addons", nil),
		intQueryParam("direction", []any{0, 1, 2, 3}),
		intQueryParam("animated", []any{0, 1}),
		intQueryParam("walk", []any{0, 1}),
		intQueryParam("size", nil),
		stringQueryParam("format", []any{"png", "gif"}),
	}
}

func outfitByNameQueryParams() []openAPIParameter {
	return []openAPIParameter{
		pathParam("name"),
		intQueryParam("direction", []any{0, 1, 2, 3}),
		intQueryParam("animated", []any{0, 1}),
		intQueryParam("walk", []any{0, 1}),
		intQueryParam("size", nil),
		stringQueryParam("format", []any{"png", "gif"}),
	}
}

func pathParam(name string) openAPIParameter {
	return openAPIParameter{
		Name:     name,
		In:       "path",
		Required: true,
		Schema:   openAPISchema{Type: "string"},
	}
}

func intQueryParam(name string, enumValues []any) openAPIParameter {
	return openAPIParameter{
		Name:     name,
		In:       "query",
		Required: false,
		Schema: openAPISchema{
			Type: "integer",
			Enum: enumValues,
		},
	}
}

func stringQueryParam(name string, enumValues []any) openAPIParameter {
	return openAPIParameter{
		Name:     name,
		In:       "query",
		Required: false,
		Schema: openAPISchema{
			Type: "string",
			Enum: enumValues,
		},
	}
}

func docsPage(c *gin.Context) {
	page := `<!doctype html>
<html>
  <head>
    <title>rubinot-data API</title>
    <meta charset="utf-8"/>
    <meta name="viewport" content="width=device-width, initial-scale=1"/>
  </head>
  <body>
    <div id="app"></div>
    <script src="https://cdn.jsdelivr.net/npm/@scalar/api-reference"></script>
    <script>
      Scalar.createApiReference('#app', {
        url: '/openapi.json',
        theme: 'deepSpace',
        layout: 'modern',
        darkMode: true,
        showSidebar: true,
        hideDownloadButton: false,
      })
    </script>
  </body>
</html>`
	c.Header("Cache-Control", "public, max-age=300")
	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(page))
}
