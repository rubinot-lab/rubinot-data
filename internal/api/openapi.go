package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

const openAPISpec = `{
  "openapi": "3.0.3",
  "info": {
    "title": "rubinot-data API",
    "version": "v0.2.0",
    "description": "Tibia scraper and data aggregation API."
  },
  "servers": [
    { "url": "/", "description": "Current environment" }
  ],
  "paths": {
    "/": {"get":{"summary":"Service status","responses":{"200":{"description":"OK"}}}},
    "/ping": {"get":{"summary":"Pong check","responses":{"200":{"description":"OK"}}}},
    "/healthz": {"get":{"summary":"Kubernetes health check","responses":{"200":{"description":"OK"}}}},
    "/readyz": {"get":{"summary":"Kubernetes ready check","responses":{"200":{"description":"OK"}}}},
    "/versions": {"get":{"summary":"Service version metadata","responses":{"200":{"description":"Version info"}}}},
    "/metrics": {"get":{"summary":"Prometheus metrics","responses":{"200":{"description":"Metrics payload"}}}},
    "/openapi.json": {"get":{"summary":"OpenAPI spec","responses":{"200":{"description":"OpenAPI document"}}}},
    "/docs": {"get":{"summary":"Interactive docs","responses":{"200":{"description":"Swagger UI"}}}},

    "/v1/worlds": {"get":{"summary":"List all worlds","tags":["v1"],"responses":{"200":{"description":"Worlds"}}}},
    "/v1/world/{name}": {"get":{"summary":"Get one world","tags":["v1"],"parameters":[{"name":"name","in":"path","required":true,"schema":{"type":"string"}}],"responses":{"200":{"description":"World detail"},"400":{"description":"Invalid world"}}}},
    "/v1/world/{name}/details": {"get":{"summary":"Get world with all online character details","tags":["v1","world"],"parameters":[{"name":"name","in":"path","required":true,"schema":{"type":"string"}}],"responses":{"200":{"description":"World details aggregate"}}}},
    "/v1/world/{name}/dashboard": {"get":{"summary":"Get world dashboard aggregate","tags":["v1","world"],"parameters":[{"name":"name","in":"path","required":true,"schema":{"type":"string"}}],"responses":{"200":{"description":"World dashboard aggregate"}}}},
    "/v1/highscores/categories": {"get":{"summary":"List highscores categories","tags":["v1","highscores"],"responses":{"200":{"description":"Highscores categories"}}}},
    "/v1/highscores/{world}": {"get":{"summary":"Redirect to experience highscores","tags":["v1","highscores"],"parameters":[{"name":"world","in":"path","required":true,"schema":{"type":"string"}}],"responses":{"302":{"description":"Redirect"}}}},
    "/v1/highscores/{world}/{category}": {"get":{"summary":"Redirect to category highscores","tags":["v1","highscores"],"parameters":[{"name":"world","in":"path","required":true,"schema":{"type":"string"}},{"name":"category","in":"path","required":true,"schema":{"type":"string","enum":["achievements","battlepass","totalbountypoints","axe","club","bosstotalpoints","distance","dromelevel","linked_tasks","experience","exp_today","fishing","fist","charmtotalpoints","magic","prestigepoints","shielding","sword","totalweeklytasks","charmunlockpoints"]}}],"responses":{"302":{"description":"Redirect"}}}},
    "/v1/highscores/{world}/{category}/{vocation}": {"get":{"summary":"Redirect to highscores page 1","tags":["v1","highscores"],"parameters":[{"name":"world","in":"path","required":true,"schema":{"type":"string"}},{"name":"category","in":"path","required":true,"schema":{"type":"string","enum":["achievements","battlepass","totalbountypoints","axe","club","bosstotalpoints","distance","dromelevel","linked_tasks","experience","exp_today","fishing","fist","charmtotalpoints","magic","prestigepoints","shielding","sword","totalweeklytasks","charmunlockpoints"]}},{"name":"vocation","in":"path","required":true,"schema":{"type":"string","enum":["all","none","knights","paladins","sorcerers","druids","monks","(all)"]}}],"responses":{"302":{"description":"Redirect"}}}},
    "/v1/highscores/{world}/{category}/{vocation}/all": {"get":{"summary":"Get highscores aggregated all pages","tags":["v1","highscores"],"parameters":[{"name":"world","in":"path","required":true,"schema":{"type":"string"}},{"name":"category","in":"path","required":true,"schema":{"type":"string","enum":["achievements","battlepass","totalbountypoints","axe","club","bosstotalpoints","distance","dromelevel","linked_tasks","experience","exp_today","fishing","fist","charmtotalpoints","magic","prestigepoints","shielding","sword","totalweeklytasks","charmunlockpoints"]}},{"name":"vocation","in":"path","required":true,"schema":{"type":"string","enum":["all","none","knights","paladins","sorcerers","druids","monks","(all)"]}}],"responses":{"200":{"description":"All highscores entries"}}}},
    "/v1/highscores/{world}/{category}/{vocation}/{page}": {"get":{"summary":"Get highscores by world/category/vocation/page","tags":["v1","highscores"],"parameters":[{"name":"world","in":"path","required":true,"schema":{"type":"string"}},{"name":"category","in":"path","required":true,"schema":{"type":"string","enum":["achievements","battlepass","totalbountypoints","axe","club","bosstotalpoints","distance","dromelevel","linked_tasks","experience","exp_today","fishing","fist","charmtotalpoints","magic","prestigepoints","shielding","sword","totalweeklytasks","charmunlockpoints"]}},{"name":"vocation","in":"path","required":true,"schema":{"type":"string","enum":["all","none","knights","paladins","sorcerers","druids","monks","(all)"]}},{"name":"page","in":"path","required":true,"schema":{"type":"integer","minimum":1}}],"responses":{"200":{"description":"Highscores"}}}},
    "/v1/killstatistics/{world}": {"get":{"summary":"Get killstatistics","tags":["v1"],"parameters":[{"name":"world","in":"path","required":true,"schema":{"type":"string"}}],"responses":{"200":{"description":"Killstats"}}}},
    "/v1/news/id/{news_id}": {"get":{"summary":"News by id","tags":["v1","news"],"parameters":[{"name":"news_id","in":"path","required":true,"schema":{"type":"string"}}],"responses":{"200":{"description":"News entry"},"404":{"description":"Not found"}}}},
    "/v1/news/archive": {"get":{"summary":"News archive","tags":["v1","news"],"responses":{"200":{"description":"News archive"}}}},
    "/v1/news/latest": {"get":{"summary":"Latest news","tags":["v1","news"],"responses":{"200":{"description":"Latest news"}}}},
    "/v1/news/newsticker": {"get":{"summary":"News ticker","tags":["v1","news"],"responses":{"200":{"description":"News ticker"}}}},
    "/v1/events/schedule": {"get":{"summary":"Events schedule","tags":["v1","events"],"responses":{"200":{"description":"Event schedule"}}}},
    "/v1/auctions/current/all/details": {"get":{"summary":"Current auctions details (all pages)","tags":["v1","auctions"],"responses":{"200":{"description":"All current auction details"}}}},
    "/v1/auctions/current/{page}/details": {"get":{"summary":"Current auctions details by page","tags":["v1","auctions"],"parameters":[{"name":"page","in":"path","required":true,"schema":{"type":"integer","minimum":1}}],"responses":{"200":{"description":"Current auction detail page"}}}},
    "/v1/auctions/current/all": {"get":{"summary":"Current auctions (all pages)","tags":["v1","auctions"],"responses":{"200":{"description":"All current auctions"}}}},
    "/v1/auctions/current/{page}": {"get":{"summary":"Current auctions","tags":["v1","auctions"],"parameters":[{"name":"page","in":"path","required":true,"schema":{"type":"integer","minimum":1}}],"responses":{"200":{"description":"Current auction page"}}}},
    "/v1/auctions/history/all/details": {"get":{"summary":"Auction history details (all pages)","tags":["v1","auctions"],"responses":{"200":{"description":"All auction history details"}}}},
    "/v1/auctions/history/{page}/details": {"get":{"summary":"Auction history details by page","tags":["v1","auctions"],"parameters":[{"name":"page","in":"path","required":true,"schema":{"type":"integer","minimum":1}}],"responses":{"200":{"description":"Auction history detail page"}}}},
    "/v1/auctions/history/all": {"get":{"summary":"Auction history (all pages)","tags":["v1","auctions"],"responses":{"200":{"description":"All auction history"}}}},
    "/v1/auctions/history/{page}": {"get":{"summary":"Auction history","tags":["v1","auctions"],"parameters":[{"name":"page","in":"path","required":true,"schema":{"type":"integer","minimum":1}}],"responses":{"200":{"description":"Auction history page"}}}},
    "/v1/auctions/{id}": {"get":{"summary":"Auction detail","tags":["v1","auctions"],"parameters":[{"name":"id","in":"path","required":true,"schema":{"type":"integer","minimum":1}}],"responses":{"200":{"description":"Auction details"},"404":{"description":"Not found"}}}},
    "/v1/deaths/{world}/all": {"get":{"summary":"Deaths (all pages)","tags":["v1"],"parameters":[{"name":"world","in":"path","required":true,"schema":{"type":"string"}}],"responses":{"200":{"description":"All deaths"}}}},
    "/v1/deaths/{world}": {"get":{"summary":"Deaths","tags":["v1"],"parameters":[{"name":"world","in":"path","required":true,"schema":{"type":"string"}}],"responses":{"200":{"description":"Deaths"}}}},
    "/v1/banishments/{world}/all": {"get":{"summary":"Banishments (all pages)","tags":["v1"],"parameters":[{"name":"world","in":"path","required":true,"schema":{"type":"string"}}],"responses":{"200":{"description":"All banishments"}}}},
    "/v1/banishments/{world}": {"get":{"summary":"Banishments","tags":["v1"],"parameters":[{"name":"world","in":"path","required":true,"schema":{"type":"string"}}],"responses":{"200":{"description":"Banishments"}}}},
    "/v1/transfers/all": {"get":{"summary":"Transfers (all pages)","tags":["v1"],"responses":{"200":{"description":"All transfers"}}}},
    "/v1/transfers": {"get":{"summary":"Transfers","tags":["v1"],"responses":{"200":{"description":"Transfers"}}}},
    "/v1/character/{name}": {"get":{"summary":"Character by name","tags":["v1"],"parameters":[{"name":"name","in":"path","required":true,"schema":{"type":"string"}}],"responses":{"200":{"description":"Character details"},"404":{"description":"Not found"}}}},
    "/v1/guild/{name}": {"get":{"summary":"Guild by name","tags":["v1","guild"],"parameters":[{"name":"name","in":"path","required":true,"schema":{"type":"string"}}],"responses":{"200":{"description":"Guild detail"},"404":{"description":"Not found"}}}},
    "/v1/guilds/{world}/all/details": {"get":{"summary":"Guilds details by world (all pages)","tags":["v1","guild"],"parameters":[{"name":"world","in":"path","required":true,"schema":{"type":"string"}}],"responses":{"200":{"description":"Guild details list"}}}},
    "/v1/guilds/{world}/all": {"get":{"summary":"Guilds by world (all pages)","tags":["v1","guild"],"parameters":[{"name":"world","in":"path","required":true,"schema":{"type":"string"}}],"responses":{"200":{"description":"Guilds list all pages"}}}},
    "/v1/guilds/{world}/{page}": {"get":{"summary":"Guilds by world page","tags":["v1","guild"],"parameters":[{"name":"world","in":"path","required":true,"schema":{"type":"string"}},{"name":"page","in":"path","required":true,"schema":{"type":"integer","minimum":1}}],"responses":{"200":{"description":"Guilds list page"}}}},
    "/v1/guilds/{world}": {"get":{"summary":"Guilds by world","tags":["v1","guild"],"parameters":[{"name":"world","in":"path","required":true,"schema":{"type":"string"}}],"responses":{"200":{"description":"Guilds list"}}}},
    "/v1/house/{world}/{house_id}": {"get":{"summary":"House by id","tags":["v1","houses"],"parameters":[{"name":"world","in":"path","required":true,"schema":{"type":"string"}},{"name":"house_id","in":"path","required":true,"schema":{"type":"integer","minimum":1}}],"responses":{"200":{"description":"House detail"},"404":{"description":"Not found"}}}},
    "/v1/houses/towns": {"get":{"summary":"Deprecated houses towns endpoint","tags":["v1","houses"],"responses":{"410":{"description":"Gone"}}}},
    "/v1/houses/{world}/{town}": {"get":{"summary":"Houses by world/town","tags":["v1","houses"],"parameters":[{"name":"world","in":"path","required":true,"schema":{"type":"string"}},{"name":"town","in":"path","required":true,"schema":{"type":"string"}}],"responses":{"200":{"description":"House list"},"400":{"description":"Invalid input"}}}}
  },
  "components": {
    "schemas": {
      "ErrorEnvelope": {
        "type": "object",
        "properties": {
          "information": {"type":"object"},
          "status": {"type":"integer"}
        }
      },
      "SuccessEnvelope": {
        "type": "object",
        "properties": {
          "information": {"type":"object"},
          "payload": {"type":"object"}
        }
      }
    }
  }
}`

func docsSpec(c *gin.Context) {
	c.Header("Cache-Control", "public, max-age=300")
	c.Data(http.StatusOK, "application/json", []byte(openAPISpec))
}

func docsPage(c *gin.Context) {
	page := `<!doctype html>
<html lang="en">
  <head>
    <meta charset="utf-8"/>
    <meta name="viewport" content="width=device-width, initial-scale=1"/>
    <title>rubinot-data API docs</title>
    <link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/swagger-ui-dist@5.18.2/swagger-ui.css"/>
    <style>
      html, body { height: 100%; margin: 0; }
      body { display: flex; flex-direction: column; }
      #toolbar { padding: 10px 16px; background: #111; color: #fff; display:flex; justify-content: space-between; align-items: center; }
      #swagger-ui { flex: 1; }
      a { color: #8dcaff; }
    </style>
  </head>
  <body>
    <div id="toolbar">
      <strong>rubinot-data API</strong>
      <div><a href="/openapi.json">openapi.json</a> · <a href="/v1/worlds">v1/worlds</a></div>
    </div>
    <div id="swagger-ui"></div>
    <script src="https://cdn.jsdelivr.net/npm/swagger-ui-dist@5.18.2/swagger-ui-bundle.js"></script>
    <script>
      window.onload = function() {
        window.ui = SwaggerUIBundle({
          url: '/openapi.json',
          dom_id: '#swagger-ui',
          presets: [SwaggerUIBundle.presets.apis, SwaggerUIBundle.SwaggerUIStandalonePreset],
          layout: 'BaseLayout'
        })
      }
    </script>
  </body>
</html>`
	c.Header("Cache-Control", "public, max-age=300")
	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(page))
}
