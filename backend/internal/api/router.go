package api

import (
	"gokafka/internal/api/middleware"

	"github.com/gin-gonic/gin"
)

func NewRouter(handler *ApiHandler, reader *ReaderApiHandler, wsHandler *WsHandler, authHandler *AuthHandler) *gin.Engine {

	r := gin.Default()

	hooks := r.Group("/api/hooks")
	hooks.POST("/registration", authHandler.HandleKratosRegistrationHook)

	api := r.Group("/api/", middleware.AuthMiddleware(), middleware.TraceIDMiddleware())
	api.GET("/ws", wsHandler.ServeWebsocket)

	api.GET("/workspaces", reader.ListUserWorkspaces)

	commands := api.Group("", middleware.AsyncEnvelope())

	commands.POST("/workspaces", handler.CreateWorkspace)

	api.POST("/users/batch", reader.BatchGetUsers)

	workspaces := api.Group("/workspaces/:workspaceID")

	workspaces.GET("/members", reader.ListWorkspaceMembers)
	workspaces.GET("/members/ids", reader.ListWorkspaceMemberIDs)

	workspaceCommands := workspaces.Group("", middleware.AsyncEnvelope())
	workspaceCommands.POST("/channels", handler.CreateChannel)
	workspaceCommands.POST("/dms", handler.CreateDM)

	workspaces.GET("/channels", reader.ListUserChannels)
	workspaces.GET("/directmessages", reader.ListUserDirectMessages)
	workspaces.GET("/browsechannels", reader.BrowseChannels)

	channels := workspaces.Group("/channels/:channelID")
	channelCommands := channels.Group("", middleware.AsyncEnvelope())
	channelCommands.POST("/messages", handler.CreateMessage)

	channels.GET("/messages", reader.ListChannelMessages)
	channels.GET("/members/ids", reader.ListChannelMemberIDs)

	return r
}
