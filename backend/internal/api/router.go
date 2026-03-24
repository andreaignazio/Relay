package api

import (
	"gokafka/internal/api/middleware"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func NewRouter(handler *ApiHandler, reader *ReaderApiHandler, wsHandler *WsHandler) *gin.Engine {

	r := gin.Default()

	r.Use(cors.New(cors.Config{

		//AllowOrigins:     []string{"https://*.vercel.app", "http://localhost:5173", "http://127.0.0.1:5173"},
		AllowWildcard:    true,
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", "x-userID", "x-correlationID", "x-correlation-id"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		AllowWebSockets:  true,
		AllowAllOrigins:  true,
	}))

	api := r.Group("/api/", middleware.TestAuthMiddleware(), middleware.TraceIDMiddleware())

	api.POST("/workspaces", handler.CreateWorkspace)
	api.GET("/workspaces", reader.ListUserWorkspaces)
	api.GET("/ws", wsHandler.ServeWebsocket)
	workspaces := api.Group("/workspaces/:workspaceID")
	workspaces.POST("/channels", handler.CreateChannel)
	workspaces.GET("/channels", reader.ListUserChannels)
	workspaces.GET("/directmessages", reader.ListUserDirectMessages)
	workspaces.GET("/browsechannels", reader.BrowseChannels)

	channels := workspaces.Group("/channels/:channelID")
	channels.POST("/messages", handler.CreateMessage)
	//channels.GET("/messages", reader.ListMessages)

	return r
}
