package websocketdomain

import (
	"context"
	"encoding/json"
	"fmt"
	"gokafka/internal/kafkaclient/producer"
	"gokafka/internal/websocketcommands"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
)

type Hub struct {
	Clients  map[uuid.UUID]*Client
	Producer *producer.KafkaProducer
	LogChan  chan string

	Register          chan *Client
	Unregister        chan *Client
	GlobalErrChan     chan error
	ConnectClientChan chan websocketcommands.WsCommandMessage
	ackChan           chan websocketcommands.AggregatorAck
}

func NewHub(logChan chan string, producer *producer.KafkaProducer, globalErrChan chan error, connectClientChan chan websocketcommands.WsCommandMessage, ackChan chan websocketcommands.AggregatorAck) *Hub {
	return &Hub{
		Clients:           make(map[uuid.UUID]*Client),
		Producer:          producer,
		LogChan:           logChan,
		Register:          make(chan *Client, 100),
		Unregister:        make(chan *Client, 100),
		GlobalErrChan:     globalErrChan,
		ConnectClientChan: connectClientChan,
		ackChan:           ackChan,
	}
}

func (h *Hub) Run(ctx context.Context) {
	for {
		select {
		case client := <-h.Register:
			h.Clients[client.ClientID] = client
			if err := h.ConnectClient(ctx, client); err != nil {
				h.GlobalErrChan <- err
			}

			go func() {
				if err := h.RunClientConsumer(client.ctx, client); err != nil {
					h.GlobalErrChan <- err
				}
			}()
			fmt.Println("Client registered:", client.ClientID)
		case client := <-h.Unregister:
			delete(h.Clients, client.ClientID)
			h.DisconnectClient(ctx, client.ClientID)
			fmt.Println("Client unregistered:", client.ClientID)
		case ack := <-h.ackChan:
			if client, exists := h.Clients[ack.ClientID]; exists {
				ackBytes, err := json.Marshal(ack.Ack)
				if err != nil {
					h.LogChan <- "Error marshaling ack for client " + ack.ClientID.String() + ": " + err.Error()
					continue
				}
				select {
				case client.send <- ackBytes:
				default:
					h.LogChan <- "Client send buffer full, dropping ack for " + ack.ClientID.String()
				}
			}
		case <-ctx.Done():
			return
		}
	}
}

func (h *Hub) Close() {
	for _, client := range h.Clients {
		client.Conn.Close()
	}
}

func (h *Hub) RunClientConsumer(ctx context.Context, client *Client) error {
	clientConsumer := client.Consumer
	if clientConsumer == nil {
		return fmt.Errorf("no consumer found for client %s", client.ClientID)
	}
	return clientConsumer.Run(ctx, func(ctx context.Context, msg kafka.Message) error {
		h.LogChan <- fmt.Sprintf("[Hub] Received message for client %s: %s", client.ClientID, string(msg.Value))
		client.send <- msg.Value
		return nil
	})
}

func (h *Hub) DisconnectClient(ctx context.Context, clientID uuid.UUID) {

	payload := websocketcommands.DisconnectCommandPayload{
		ClientID: clientID,
	}
	payloadJson, err := json.Marshal(payload)
	if err != nil {
		h.LogChan <- "Error marshaling disconnect command payload for client " + clientID.String() + ": " + err.Error()
		return
	}
	disconnectCommand := websocketcommands.NewWsCommandMessage(websocketcommands.WsCommandKeyDisconnect, clientID.String(), payloadJson)
	if err := h.Producer.WriteMessage(ctx, disconnectCommand); err != nil {
		h.LogChan <- "Error sending disconnect command for client " + clientID.String() + ": " + err.Error()
	} else {
		h.LogChan <- "Sent disconnect command for client " + clientID.String()
	}
}

func (h *Hub) ConnectClient(ctx context.Context, client *Client) error {
	payload := websocketcommands.ConnectCommandPayload{
		UserID:   client.UserID,
		ClientID: client.ClientID,
	}
	payloadJson, err := json.Marshal(payload)
	if err != nil {
		h.LogChan <- "Error marshaling connect command payload for client " + client.ClientID.String() + ": " + err.Error()
		return err
	}
	connectCommand := websocketcommands.NewWsCommandMessage(websocketcommands.WsCommandKeyConnect, client.ClientID.String(), payloadJson)
	/*if err := h.Producer.WriteMessage(ctx, connectCommand); err != nil {
		h.LogChan <- "Error sending connect command for client " + client.ClientID.String() + ": " + err.Error()
		return err
	} else {
		h.LogChan <- "Sent connect command for client " + client.ClientID.String()
	}*/
	h.ConnectClientChan <- connectCommand
	return nil
}
