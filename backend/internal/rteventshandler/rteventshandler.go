package rteventshandler

import (
	"context"
	"fmt"
	"gokafka/internal/shared"
)

type Handler struct {
	logChan chan string
	// Define any dependencies or fields needed for the handler
}

func NewHandler(logChan chan string) *Handler {
	return &Handler{
		logChan: logChan,
	}
}

func (h *Handler) HandleCreateWorkspaceEvent(ctx context.Context, event shared.Event) error {
	//fmt.Println("[RTHandler]Handling CreateWorkspace event for workspace ID:", event.GetPartitionKey())
	h.logChan <- fmt.Sprintf("[RTHandler]Handling CreateWorkspace event for workspace ID: %s", event.GetPartitionKey())
	return nil
}
