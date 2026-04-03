package api

import (
	"github.com/gin-gonic/gin"
	"github.com/giovannirco/rubinot-data/internal/scraper"
)

func registerV2Routes(router *gin.Engine, oc *scraper.OptimizedClient) {
	v2 := router.Group("/v2")
	{
		v2.GET("/worlds", handleEndpoint(func(c *gin.Context) (endpointResult, error) {
			return v2GetWorlds(c, oc)
		}))
		v2.GET("/world/:name", handleEndpoint(func(c *gin.Context) (endpointResult, error) {
			return v2GetWorld(c, getValidator(), oc)
		}))
		v2.GET("/world/:name/details", handleEndpoint(func(c *gin.Context) (endpointResult, error) {
			return v2GetWorldDetails(c, getValidator(), oc)
		}))
		v2.GET("/world/:name/dashboard", handleEndpoint(func(c *gin.Context) (endpointResult, error) {
			return v2GetWorldDashboard(c, getValidator(), oc)
		}))
		v2.GET("/highscores/:world/:category/:vocation", handleEndpoint(func(c *gin.Context) (endpointResult, error) {
			return v2GetHighscores(c, getValidator(), oc)
		}))
		v2.GET("/killstatistics/:world", handleEndpoint(func(c *gin.Context) (endpointResult, error) {
			return v2GetKillstatistics(c, getValidator(), oc)
		}))
		v2.GET("/deaths/:world", handleEndpoint(func(c *gin.Context) (endpointResult, error) {
			return v2GetDeaths(c, getValidator(), oc)
		}))
		v2.GET("/deaths/:world/all", handleEndpoint(func(c *gin.Context) (endpointResult, error) {
			return v2GetAllDeaths(c, getValidator(), oc)
		}))
		v2.GET("/banishments/:world", handleEndpoint(func(c *gin.Context) (endpointResult, error) {
			return v2GetBanishments(c, getValidator(), oc)
		}))
		v2.GET("/banishments/:world/all", handleEndpoint(func(c *gin.Context) (endpointResult, error) {
			return v2GetAllBanishments(c, getValidator(), oc)
		}))
		v2.GET("/transfers", handleEndpoint(func(c *gin.Context) (endpointResult, error) {
			return v2GetTransfers(c, getValidator(), oc)
		}))
		v2.GET("/transfers/all", handleEndpoint(func(c *gin.Context) (endpointResult, error) {
			return v2GetAllTransfers(c, getValidator(), oc)
		}))
		v2.GET("/character/:name", handleEndpoint(func(c *gin.Context) (endpointResult, error) {
			return v2GetCharacter(c, oc)
		}))
		v2.GET("/guild/:name", handleEndpoint(func(c *gin.Context) (endpointResult, error) {
			return v2GetGuild(c, oc)
		}))
		v2.GET("/guilds/:world", handleEndpoint(func(c *gin.Context) (endpointResult, error) {
			return v2GetGuilds(c, getValidator(), oc)
		}))
		v2.GET("/guilds/:world/all", handleEndpoint(func(c *gin.Context) (endpointResult, error) {
			return v2GetAllGuilds(c, getValidator(), oc)
		}))
		v2.GET("/boosted", handleEndpoint(func(c *gin.Context) (endpointResult, error) {
			return v2GetBoosted(c, oc)
		}))
		v2.GET("/maintenance", handleEndpoint(func(c *gin.Context) (endpointResult, error) {
			return v2GetMaintenance(c, oc)
		}))
		v2.GET("/auctions/current/all", handleEndpoint(func(c *gin.Context) (endpointResult, error) {
			return v2GetAllCurrentAuctions(c, oc)
		}))
		v2.GET("/auctions/current/:page", handleEndpoint(func(c *gin.Context) (endpointResult, error) {
			return v2GetCurrentAuctions(c, oc)
		}))
		v2.GET("/auctions/history/all", handleEndpoint(func(c *gin.Context) (endpointResult, error) {
			return v2GetAllAuctionHistory(c, oc)
		}))
		v2.GET("/auctions/history/:page", handleEndpoint(func(c *gin.Context) (endpointResult, error) {
			return v2GetAuctionHistory(c, oc)
		}))
		v2.GET("/auctions/:id", handleEndpoint(func(c *gin.Context) (endpointResult, error) {
			return v2GetAuctionDetail(c, oc)
		}))
		v2.GET("/news/id/:news_id", handleEndpoint(func(c *gin.Context) (endpointResult, error) {
			return v2GetNewsByID(c, oc)
		}))
		v2.GET("/news/archive", handleEndpoint(func(c *gin.Context) (endpointResult, error) {
			return v2GetNewsArchive(c, oc)
		}))
		v2.GET("/news/latest", handleEndpoint(func(c *gin.Context) (endpointResult, error) {
			return v2GetNewsLatest(c, oc)
		}))
		v2.GET("/news/newsticker", handleEndpoint(func(c *gin.Context) (endpointResult, error) {
			return v2GetNewsTicker(c, oc)
		}))
		v2.GET("/outfit", getOutfit)
		v2.GET("/outfit/:name", getOutfitByCharacterName)
		v2.GET("/assets/creatures/:name", handleCreatureAsset(resolvedAssetsDir))
		v2.GET("/assets/items/:itemId", handleItemAsset(resolvedAssetsDir, "https://static.rubinot.com"))
		v2.GET("/assets/charms/:name", handleStaticAsset(resolvedAssetsDir, "charms", "image/png", ".png"))
		v2.GET("/assets/creature-types/:type", handleStaticAsset(resolvedAssetsDir, "creature-types", "image/png", ".png"))
	}
}
