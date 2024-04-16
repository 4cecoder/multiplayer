// Package handlers queueing.go contains the implementation of a message queue that stores messages in memory and persists them to the file system.
// Package handlers contains the implementation of a message queue that stores messages in memory and persists them to the file system.
package handlers

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
)

type MessageQueue struct {
	mu       sync.Mutex
	messages map[string][][]byte // map of client ID to message queue
}

func NewMessageQueue() *MessageQueue {
	return &MessageQueue{
		messages: make(map[string][][]byte),
	}
}

func (mq *MessageQueue) Enqueue(clientID string, message []byte) error {
	mq.mu.Lock()
	defer mq.mu.Unlock()

	mq.messages[clientID] = append(mq.messages[clientID], message)

	if err := mq.persistMessage(clientID, message); err != nil {
		log.Printf("Failed to persist message for client %s: %v", clientID, err)
		return fmt.Errorf("failed to persist message: %w", err)
	}
	return nil
}

func (mq *MessageQueue) Dequeue(clientID string) ([]byte, error) {
	mq.mu.Lock()
	defer mq.mu.Unlock()

	messages, ok := mq.messages[clientID]
	if !ok || len(messages) == 0 {
		return nil, fmt.Errorf("no messages for client %s", clientID)
	}

	message := messages[0]
	mq.messages[clientID] = messages[1:]

	return message, nil
}

func (mq *MessageQueue) persistMessage(clientID string, message []byte) error {
	filePath := filepath.Join("data", clientID, fmt.Sprintf("%d", len(mq.messages[clientID])-1))
	err := os.MkdirAll(filepath.Dir(filePath), 0755)
	if err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	err = os.WriteFile(filePath, message, 0644)
	if err != nil {
		return fmt.Errorf("failed to write message to file: %w", err)
	}
	return nil
}

func (mq *MessageQueue) QueueSize(clientID string) int {
	mq.mu.Lock()
	defer mq.mu.Unlock()

	return len(mq.messages[clientID])
}

func (mq *MessageQueue) ClearQueue(clientID string) {
	mq.mu.Lock()
	defer mq.mu.Unlock()

	delete(mq.messages, clientID)
}
