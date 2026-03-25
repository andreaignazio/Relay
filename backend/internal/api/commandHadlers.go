package api

import (
	"encoding/json"
	"gokafka/internal/commandpayloads"
	"gokafka/internal/kafkaclient/producer"
	"gokafka/internal/models/entities"
	"gokafka/internal/shared"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type ApiHandler struct {
	Producer *producer.KafkaProducer
}

func NewHandler(producer *producer.KafkaProducer) *ApiHandler {
	return &ApiHandler{
		Producer: producer,
	}
}

func (h *ApiHandler) CreateWorkspace(c *gin.Context) {
	ctx := c.Request.Context()
	userIDStr := c.MustGet("UserID").(string)
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		c.JSON(400, gin.H{"error": "Invalid user ID"})
		return
	}
	traceID := c.MustGet("TraceID").(uuid.UUID)

	var req CreateWorkspaceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "Invalid request"})
		return
	}

	// ABAC Checks would go here
	aggregateID := uuid.New() // Generate a new UUID for the workspace
	metadata := shared.MessageMetadata{
		UserID: userID.String(),
	}
	payloadBytes, err := json.Marshal(commandpayloads.CreateWorkspacePayload{
		Name:    req.Name,
		IconURL: req.IconURL,
	})
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to marshal payload"})
		return
	}

	cmd := shared.NewCommand(
		aggregateID, // AggregateID for the new workspace
		shared.ActionKeyWorkspaceCreate,
		shared.NewWorkspaceCommandPartitionKey(aggregateID.String()), // Using workspace name as partition key
		traceID,
		metadata,
		payloadBytes, // Payload is the marshaled JSON
	)

	if err := h.Producer.WriteMessage(ctx, cmd); err != nil {
		c.JSON(500, gin.H{"error": "Failed to send command"})
		return
	}

	c.JSON(200, gin.H{"CommandID": cmd.GetMessageID()})
}

func (h *ApiHandler) CreateDM(c *gin.Context) {
	ctx := c.Request.Context()
	userIDStr := c.MustGet("UserID").(string)
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		c.JSON(400, gin.H{"error": "Invalid user ID"})
		return
	}
	traceID := c.MustGet("TraceID").(uuid.UUID)
	workspaceIDStr := c.Param("workspaceID")
	workspaceID, err := uuid.Parse(workspaceIDStr)
	if err != nil {
		c.JSON(400, gin.H{"error": "Invalid workspace ID"})
		return
	}

	var req CreateDirectMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "Invalid request"})
		return
	}

	participantIDs := make([]uuid.UUID, 0, len(req.RecipientIDs)+1)
	participantIDs = append(participantIDs, userID)
	for _, idStr := range req.RecipientIDs {
		id, err := uuid.Parse(idStr)
		if err != nil {
			c.JSON(400, gin.H{"error": "Invalid recipient ID: " + idStr})
			return
		}
		participantIDs = append(participantIDs, id)
	}

	aggregateID := uuid.New()
	metadata := shared.MessageMetadata{UserID: userID.String()}
	payloadBytes, err := json.Marshal(commandpayloads.CreateDMPayload{
		WorkspaceID:    workspaceID,
		ParticipantIDs: participantIDs,
	})
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to marshal payload"})
		return
	}

	cmd := shared.NewCommand(
		aggregateID,
		shared.ActionKeyDMCreate,
		shared.NewDMCommandPartitionKey(aggregateID.String()),
		traceID,
		metadata,
		payloadBytes,
	)

	if err := h.Producer.WriteMessage(ctx, cmd); err != nil {
		c.JSON(500, gin.H{"error": "Failed to send command"})
		return
	}

	c.JSON(200, gin.H{"CommandID": cmd.GetMessageID()})
}

func (h *ApiHandler) ListWorkspaces(c *gin.Context) {

	workspaces := []entities.Workspace{
		{ID: uuid.New(), Name: "Workspace 1"},
		{ID: uuid.New(), Name: "Workspace 2"},
		{ID: uuid.New(), Name: "Workspace 3"},
	}
	c.JSON(200, workspaces)
}

func (h *ApiHandler) CreateChannel(c *gin.Context) {
	ctx := c.Request.Context()
	userIDStr := c.MustGet("UserID").(string)
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		c.JSON(400, gin.H{"error": "Invalid user ID"})
		return
	}
	traceID := c.MustGet("TraceID").(uuid.UUID)

	workspaceIDStr := c.Param("workspaceID")
	workspaceID, err := uuid.Parse(workspaceIDStr)
	if err != nil {
		c.JSON(400, gin.H{"error": "Invalid workspace ID"})
		return
	}

	var req CreateChannelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "Invalid request"})
		return
	}

	// ABAC Checks would go here
	aggregateID := uuid.New() // Generate a new UUID for the channel
	metadata := shared.MessageMetadata{
		UserID: userID.String(),
	}
	payloadBytes, err := json.Marshal(commandpayloads.CreateChannelPayload{
		Name:              req.Name,
		Description:       req.Description,
		Topic:             req.Topic,
		Type:              req.Type,
		NotificationsPref: req.NotificationsPref,
		WorkspaceID:       workspaceID,
	})
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to marshal payload"})
		return
	}

	cmd := shared.NewCommand(
		aggregateID, // AggregateID for the new channel
		shared.ActionKeyChannelCreate,
		shared.NewChannelCommandPartitionKey(aggregateID.String()), // Using channel name as partition key
		traceID,
		metadata,
		payloadBytes, // Payload is the marshaled JSON
	)

	if err := h.Producer.WriteMessage(ctx, cmd); err != nil {
		c.JSON(500, gin.H{"error": "Failed to send command"})
		return
	}

	c.JSON(200, gin.H{"CommandID": cmd.GetMessageID()})
}

func (h *ApiHandler) CreateMessage(c *gin.Context) {
	ctx := c.Request.Context()
	userIDStr := c.MustGet("UserID").(string)
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		c.JSON(400, gin.H{"error": "Invalid user ID"})
		return
	}
	traceID := c.MustGet("TraceID").(uuid.UUID)
	workspaceIDStr := c.Param("workspaceID")
	workspaceID, err := uuid.Parse(workspaceIDStr)
	if err != nil {
		c.JSON(400, gin.H{"error": "Invalid workspace ID"})
		return
	}
	channelIDStr := c.Param("channelID")
	channelID, err := uuid.Parse(channelIDStr)
	if err != nil {
		c.JSON(400, gin.H{"error": "Invalid channel ID"})
		return
	}
	var req CreateMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "Invalid request"})
		return
	}

	aggregateID := uuid.New() // Generate a new UUID for the message
	metadata := shared.MessageMetadata{
		UserID: userID.String(),
	}
	payloadBytes, err := json.Marshal(commandpayloads.CreateMessagePayload{
		Content:     req.Content,
		ChannelID:   channelID,
		WorkspaceID: workspaceID,
	})
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to marshal payload"})
		return
	}

	cmd := shared.NewCommand(
		aggregateID, // AggregateID for the new message
		shared.ActionKeyMessageSend,
		shared.NewMessageCommandPartitionKey(channelID.String()), // Using channel ID as partition key
		traceID,
		metadata,
		payloadBytes, // Payload is the marshaled JSON
	)

	if err := h.Producer.WriteMessage(ctx, cmd); err != nil {
		c.JSON(500, gin.H{"error": "Failed to send command"})
		return
	}

	c.JSON(200, gin.H{"CommandID": cmd.GetMessageID()})
}
