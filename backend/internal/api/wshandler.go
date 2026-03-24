package api

import (
	"context"
	"fmt"
	"gokafka/internal/kafkaclient/consumer"
	"gokafka/internal/kafkaconn"
	"gokafka/internal/shared"
	"gokafka/internal/topics"
	"gokafka/internal/websocketdomain"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

type WsHandler struct {
	Hub             *websocketdomain.Hub
	KafkaConn       *kafkaconn.KafkaConn
	ConsumerGroupID shared.ConsumerGroupId
	LogChan         chan string
	ctx             context.Context
}

func NewWsHandler(hub *websocketdomain.Hub, kafkaConn *kafkaconn.KafkaConn, consumerGroupID shared.ConsumerGroupId, logChan chan string, ctx context.Context) *WsHandler {
	return &WsHandler{
		Hub:             hub,
		KafkaConn:       kafkaConn,
		ConsumerGroupID: consumerGroupID,
		LogChan:         logChan,
		ctx:             ctx,
	}
}

func (h *WsHandler) ServeWebsocket(c *gin.Context) {
	fmt.Println("Websocket connection established")
	userIDStr := c.MustGet("UserID").(string)
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		c.JSON(500, gin.H{"error": "Invalid user ID"})
		return
	}
	connectionIdStr := c.Query("connectionId")
	if connectionIdStr == "" {
		c.JSON(400, gin.H{"error": "Missing connectionId"})
		return
	}
	connectionId, err := uuid.Parse(connectionIdStr)
	if err != nil {
		c.JSON(400, gin.H{"error": "Invalid connectionId"})
		return
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to upgrade to websocket"})
		return
	}
	clientId := uuid.New()
	topic := shared.NewWsClientTopic(clientId.String(), 1, 1)
	if err := topics.CreateTopic(topic, h.KafkaConn.Conn); err != nil {
		fmt.Println("Error creating topic:", err)
	}
	clientCtx, clientCancel := context.WithCancel(h.ctx)
	clientConsumer := consumer.NewKafkaConsumer(topic, h.ConsumerGroupID, h.KafkaConn)
	client := websocketdomain.NewClient(clientId, connectionId, userID, conn, clientConsumer, h.LogChan, h.Hub, h.KafkaConn, topic, clientCtx, clientCancel)
	h.Hub.Register <- client
	go client.ReadPump()
	go client.WritePump()
}
