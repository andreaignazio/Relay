package api

import (
	"encoding/json"
	"gokafka/internal/commandpayloads"
	"gokafka/internal/kafkaclient/producer"
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

type CommandsHeadesr struct {
	UserID    uuid.UUID
	TraceID   uuid.UUID
	MessageID uuid.UUID
	ActionKey string
}

func (h *ApiHandler) CommandsMustHeader(c *gin.Context) (headers *CommandsHeadesr, err error) {
	userIDStr := c.MustGet("UserID").(string)
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return nil, err
	}
	traceID := c.MustGet("TraceID").(uuid.UUID)
	messageID := c.MustGet("X-Message-ID").(uuid.UUID)
	actionKey := c.MustGet("X-Action-Key").(string)
	return &CommandsHeadesr{
		UserID:    userID,
		TraceID:   traceID,
		MessageID: messageID,
		ActionKey: actionKey,
	}, nil
}

func (h *ApiHandler) CreateWorkspace(c *gin.Context) {
	ctx := c.Request.Context()
	headers, err := h.CommandsMustHeader(c)
	if err != nil {
		c.JSON(400, gin.H{"error": "Invalid headers: " + err.Error()})
		return
	}

	userID := headers.UserID
	traceID := headers.TraceID
	messageID := headers.MessageID
	actionKey := headers.ActionKey
	if actionKey != string(shared.ActionKeyWorkspaceCreate) {
		c.JSON(400, gin.H{"error": "Invalid action key"})
		return
	}

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
		Name:        req.Name,
		IconURL:     req.IconURL,
		Description: req.Description,
	})
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to marshal payload"})
		return
	}

	cmd := shared.NewCommand(
		messageID,
		aggregateID, // AggregateID for the new workspace
		shared.ActionKeyWorkspaceCreate,
		shared.NewWorkspaceCommandPartitionKey(aggregateID.String()), // Using workspace name as partition key
		traceID,
		metadata,
		payloadBytes, // Payload is the marshaled JSON
		userID,
	)

	if err := h.Producer.WriteMessage(ctx, cmd); err != nil {
		c.JSON(500, gin.H{"error": "Failed to send command"})
		return
	}

	c.JSON(200, gin.H{"CommandID": cmd.GetMessageID()})
}

func (h *ApiHandler) CreateDM(c *gin.Context) {
	ctx := c.Request.Context()

	headers, err := h.CommandsMustHeader(c)
	if err != nil {
		c.JSON(400, gin.H{"error": "Invalid headers: " + err.Error()})
		return
	}
	userID := headers.UserID
	traceID := headers.TraceID
	messageID := headers.MessageID
	actionKey := headers.ActionKey
	if actionKey != string(shared.ActionKeyDMCreate) {
		c.JSON(400, gin.H{"error": "Invalid action key"})
		return
	}

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
		messageID,
		aggregateID,
		shared.ActionKeyDMCreate,
		shared.NewDMCommandPartitionKey(aggregateID.String()),
		traceID,
		metadata,
		payloadBytes,
		userID,
	)

	if err := h.Producer.WriteMessage(ctx, cmd); err != nil {
		c.JSON(500, gin.H{"error": "Failed to send command"})
		return
	}

	c.JSON(200, gin.H{"CommandID": cmd.GetMessageID()})
}

func (h *ApiHandler) CreateChannel(c *gin.Context) {
	ctx := c.Request.Context()

	headers, err := h.CommandsMustHeader(c)
	if err != nil {
		c.JSON(400, gin.H{"error": "Invalid headers: " + err.Error()})
		return
	}
	userID := headers.UserID
	traceID := headers.TraceID
	messageID := headers.MessageID
	actionKey := headers.ActionKey
	if actionKey != string(shared.ActionKeyChannelCreate) {
		c.JSON(400, gin.H{"error": "Invalid action key"})
		return
	}

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
		messageID,
		aggregateID, // AggregateID for the new channel
		shared.ActionKeyChannelCreate,
		shared.NewChannelCommandPartitionKey(aggregateID.String()), // Using channel name as partition key
		traceID,
		metadata,
		payloadBytes, // Payload is the marshaled JSON
		userID,
	)

	if err := h.Producer.WriteMessage(ctx, cmd); err != nil {
		c.JSON(500, gin.H{"error": "Failed to send command"})
		return
	}

	c.JSON(200, gin.H{"CommandID": cmd.GetMessageID()})
}

func (h *ApiHandler) CreateMessage(c *gin.Context) {
	ctx := c.Request.Context()

	headers, err := h.CommandsMustHeader(c)
	if err != nil {
		c.JSON(400, gin.H{"error": "Invalid headers: " + err.Error()})
		return
	}
	userID := headers.UserID
	traceID := headers.TraceID
	messageID := headers.MessageID
	actionKey := headers.ActionKey
	if actionKey != string(shared.ActionKeyMessageSend) {
		c.JSON(400, gin.H{"error": "Invalid action key"})
		return
	}

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
	mentionedUserIDs := make([]uuid.UUID, 0, len(req.MentionedUserIDs))
	for _, idStr := range req.MentionedUserIDs {
		id, err := uuid.Parse(idStr)
		if err != nil {
			c.JSON(400, gin.H{"error": "Invalid mentioned user ID: " + idStr})
			return
		}
		mentionedUserIDs = append(mentionedUserIDs, id)
	}

	payloadBytes, err := json.Marshal(commandpayloads.CreateMessagePayload{
		Content:          req.Content,
		ChannelID:        channelID,
		WorkspaceID:      workspaceID,
		MentionedUserIDs: mentionedUserIDs,
		MentionChannel:   req.MentionChannel,
		MentionHere:      req.MentionHere,
	})
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to marshal payload"})
		return
	}

	cmd := shared.NewCommand(
		messageID,
		aggregateID, // AggregateID for the new message
		shared.ActionKeyMessageSend,
		shared.NewMessageCommandPartitionKey(channelID.String()), // Using channel ID as partition key
		traceID,
		metadata,
		payloadBytes, // Payload is the marshaled JSON
		userID,
	)

	if err := h.Producer.WriteMessage(ctx, cmd); err != nil {
		c.JSON(500, gin.H{"error": "Failed to send command"})
		return
	}

	c.JSON(200, gin.H{"CommandID": cmd.GetMessageID()})
}
