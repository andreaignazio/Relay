package api

import (
	"encoding/json"
	"gokafka/internal/commandpayloads"
	"gokafka/internal/kafkaclient/producer"
	"gokafka/internal/shared"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type AuthHandler struct {
	Producer *producer.KafkaProducer
}

func NewAuthHandler(producer *producer.KafkaProducer) *AuthHandler {
	return &AuthHandler{
		Producer: producer,
	}
}

func (h *AuthHandler) HandleKratosRegistrationHook(c *gin.Context) {
	webhookSecret := c.GetHeader("X-Webhook-Secret")
	expectedSecret := "relay-webhook-secret-change-me"
	if webhookSecret != expectedSecret {
		c.JSON(401, gin.H{"error": "Unauthorized"})
		return
	}
	var req KratosRegistrationHookRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "Invalid request body"})
		return
	}
	identityIDStr := req.IdentityID
	identityID, err := uuid.Parse(identityIDStr)
	if err != nil {
		c.JSON(400, gin.H{"error": "Invalid identity_id format"})
		return
	}

	payload := commandpayloads.RegisterUserPayload{
		IdentityID:  identityID,
		Email:       req.Email,
		DisplayName: req.DisplayName,
		Username:    req.Username,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to marshal command payload"})
		return
	}

	commandEvent := shared.Command{
		MessageID:    uuid.New(),
		AggregateID:  identityID,
		ActionKey:    shared.ActionKeyUserRegister,
		PartitionKey: shared.NewUserCommandPartitionKey(identityID.String()),
		Payload:      payloadBytes,
	}
	if err := h.Producer.WriteMessage(c.Request.Context(), commandEvent); err != nil {
		c.JSON(500, gin.H{"error": "Failed to produce command event"})
		return
	}

	c.JSON(200, gin.H{"message": "User registration command produced", "user_id": identityID})
}
