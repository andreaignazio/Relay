package websocketdomain

import (
	"context"
	"encoding/json"
	"fmt"
	"gokafka/internal/kafkaclient/consumer"
	"gokafka/internal/kafkaconn"
	"gokafka/internal/shared"
	"gokafka/internal/topics"
	"gokafka/internal/websocketcommands"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

type Client struct {
	ClientID     uuid.UUID
	ConnectionId uuid.UUID
	UserID       uuid.UUID
	Conn         *websocket.Conn
	send         chan []byte
	hub          *Hub
	LogChan      chan string
	Consumer     *consumer.KafkaConsumer
	Topic        shared.Topic
	kafkaConn    *kafkaconn.KafkaConn
	ctx          context.Context
	cancel       context.CancelFunc
}

func NewClient(clientID, connectionId, userID uuid.UUID,
	conn *websocket.Conn,
	consumer *consumer.KafkaConsumer,
	logChan chan string,
	hub *Hub,
	kafkaConn *kafkaconn.KafkaConn,
	topic shared.Topic,
	ctx context.Context,
	cancel context.CancelFunc) *Client {
	return &Client{
		ClientID:     clientID,
		ConnectionId: connectionId,
		UserID:       userID,
		Conn:         conn,
		send:         make(chan []byte, 256),
		hub:          hub,
		LogChan:      logChan,
		Consumer:     consumer,
		kafkaConn:    kafkaConn,
		Topic:        topic,
		ctx:          ctx,
		cancel:       cancel,
	}
}

func (c *Client) ReadPump() {
	defer func() {
		c.cancel()
		c.hub.Unregister <- c
		c.Close()
	}()
	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			c.LogChan <- "Error reading message: " + err.Error()
			break
		}
		c.LogChan <- fmt.Sprintf("[Client ReadPump] Raw message received from client %s: %s", c.ClientID, string(message))
		if err := c.forwardToAggregator(message); err != nil {
			c.LogChan <- "Error forwarding message: " + err.Error()
		}
	}
}

func (c *Client) forwardToAggregator(raw []byte) error {
	var msg websocketcommands.FrontendWsMessage
	if err := json.Unmarshal(raw, &msg); err != nil {
		return fmt.Errorf("unmarshal frontend message: %w", err)
	}

	var kafkaPayload any
	switch msg.WsCommandKey {
	case websocketcommands.WsCommandKeySubscribe:
		var p websocketcommands.FrontendSubscribePayload
		if err := json.Unmarshal(msg.Payload, &p); err != nil {
			return fmt.Errorf("unmarshal subscribe payload: %w", err)
		}
		kafkaPayload = websocketcommands.SubscribeCommandPayload{
			ClientID:      c.ClientID,
			UserID:        c.UserID,
			Subscriptions: p.Subscriptions,
		}
		c.LogChan <- fmt.Sprintf("[Client forwardToAggregator] SubscribeCommandPayload: ClientID=%s, UserID=%s, Subscriptions=%v", c.ClientID, c.UserID, p.Subscriptions)
	case websocketcommands.WsCommandKeyUnsubscribe:
		var p websocketcommands.FrontendUnsubscribePayload
		if err := json.Unmarshal(msg.Payload, &p); err != nil {
			return fmt.Errorf("unmarshal unsubscribe payload: %w", err)
		}
		kafkaPayload = websocketcommands.UnsubscribeCommandPayload{
			ClientID:      c.ClientID,
			UserID:        c.UserID,
			Subscriptions: p.Subscriptions,
		}
		c.LogChan <- fmt.Sprintf("[Client forwardToAggregator] UnsubscribeCommandPayload: ClientID=%s, UserID=%s, Subscriptions=%v", c.ClientID, c.UserID, p.Subscriptions)
	case websocketcommands.WsCommandKeyFocus:
		var p websocketcommands.FrontendFocusPayload
		if err := json.Unmarshal(msg.Payload, &p); err != nil {
			return fmt.Errorf("unmarshal focus payload: %w", err)
		}
		kafkaPayload = websocketcommands.FocusCommandPayload{
			ClientID:  c.ClientID,
			UserID:    c.UserID,
			ChannelID: p.ChannelID,
		}
		c.LogChan <- fmt.Sprintf("[Client forwardToAggregator] FocusCommandPayload: ClientID=%s, UserID=%s, ChannelID=%s", c.ClientID, c.UserID, p.ChannelID)
	default:
		c.LogChan <- "Unknown WsCommandKey: " + string(msg.WsCommandKey)
		return nil
	}

	payloadJson, err := json.Marshal(kafkaPayload)
	if err != nil {
		return fmt.Errorf("marshal kafka payload: %w", err)
	}
	wsCmd := websocketcommands.NewWsCommandMessage(msg.WsCommandKey, msg.WsCommandID, c.ClientID.String(), payloadJson)
	return c.hub.Producer.WriteMessage(c.ctx, wsCmd)
}

func (c *Client) WritePump() {
	for {
		select {
		case message := <-c.send:
			if err := c.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
				c.LogChan <- "[Client WritePump] Error writing message: " + err.Error()
				return
			}
			c.LogChan <- fmt.Sprintf("[Client WritePump] Sent message to client %s: %s", c.ClientID, string(message))
		case <-c.ctx.Done():
			c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
			return
		}
	}
}

func (c *Client) Close() {
	c.Conn.Close()
	if c.Consumer != nil {
		c.Consumer.Close()
	}
	if err := topics.DeleteTopic(c.Topic, c.kafkaConn.Conn); err != nil {
		c.LogChan <- fmt.Sprintf("[Client Close] Error deleting topic %s: %s", c.Topic.Name, err.Error())
	}
}
