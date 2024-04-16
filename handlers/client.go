// Package handlers/client.go
package handlers

import (
	"encoding/json"
	"github.com/4cecoder/multiplayer/models"
	"github.com/gorilla/websocket"
	"log"
	"net/http"
	"os"
	"sync"
	"time"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:    1022,
	WriteBufferSize:   1022,
	CheckOrigin:       func(r *http.Request) bool { return true },
	EnableCompression: false, // Disable compression
}

// Mutex to protect access to the clients map
var clientsMutex sync.Mutex

// Map to keep track of connected clients
var clients map[string]*Client = make(map[string]*Client)

type EventType int

const (
	EventTypeMessage EventType = iota
	EventTypeLogin
	EventTypeLogout
	EventTypeError
	EventTypeReconnect
	EventTypeSignal
)

type Event struct {
	Type    EventType
	Client  *Client
	Message []byte
	Err     error
}

type Client struct {
	ID                string
	Conn              *websocket.Conn
	Send              chan []byte
	Mutex             sync.Mutex
	reconnectInterval time.Duration
	maxRetryAttempts  int
	retryAttempts     int
	isClosed          bool
	messageQueue      *MessageQueue
	Player            *models.Player
	EventQueue        chan Event
	SignalChannel     chan SignalMessage
}

type SignalMessage struct {
	Type    string `json:"type"`
	Content string `json:"content"` // Could be JSON of the offer, answer, or ICE candidate
}

func NewClient(conn *websocket.Conn, id string, messageQueue *MessageQueue) *Client {
	return &Client{
		ID:                id,
		Conn:              conn,
		Send:              make(chan []byte, 256),
		reconnectInterval: 5 * time.Second,
		maxRetryAttempts:  5,
		messageQueue:      messageQueue,
		Player:            &models.Player{},
		EventQueue:        make(chan Event, 16),
		SignalChannel:     make(chan SignalMessage, 16),
	}
}

func (c *Client) ReadPump() {
	defer func() {
		c.emitEvent(Event{Type: EventTypeLogout, Client: c})
		c.Conn.Close()
		c.isClosed = true
		close(c.Send)
		close(c.SignalChannel)
	}()

	for {
		messageType, message, err := c.Conn.ReadMessage()
		if err != nil {
			c.emitEvent(Event{Type: EventTypeError, Client: c, Err: err})
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("websocket error: %v", err)
				c.handleReconnect()
			}
			break
		}

		if messageType == websocket.TextMessage {
			var signalMessage SignalMessage
			err := json.Unmarshal(message, &signalMessage)
			if err != nil {
				c.emitEvent(Event{Type: EventTypeMessage, Client: c, Message: message})
			} else {
				c.emitEvent(Event{Type: EventTypeSignal, Client: c, Message: message})
			}
		}
	}
}

func (c *Client) WritePump() {
	for {
		select {
		case message := <-c.Send:
			c.Mutex.Lock()
			err := c.Conn.WriteMessage(websocket.TextMessage, message)
			c.Mutex.Unlock()
			if err != nil {
				c.emitEvent(Event{Type: EventTypeError, Client: c, Err: err})
				log.Printf("error writing to websocket: %v", err)
				c.handleReconnect()
				return
			}
		case signal := <-c.SignalChannel:
			signalMessage, err := json.Marshal(signal)
			if err != nil {
				c.emitEvent(Event{Type: EventTypeError, Client: c, Err: err})
				log.Printf("error marshaling signal message: %v", err)
				continue
			}
			c.Mutex.Lock()
			err = c.Conn.WriteMessage(websocket.TextMessage, signalMessage)
			c.Mutex.Unlock()
			if err != nil {
				c.emitEvent(Event{Type: EventTypeError, Client: c, Err: err})
				log.Printf("error writing signal to websocket: %v", err)
				c.handleReconnect()
				return
			}
		default:
			messages, err := c.messageQueue.Dequeue(c.ID)
			if err != nil {
				time.Sleep(100 * time.Millisecond)
				continue
			}

			for _, message := range messages {
				c.Mutex.Lock()
				err := c.Conn.WriteMessage(websocket.TextMessage, []byte{message})
				c.Mutex.Unlock()
				if err != nil {
					c.emitEvent(Event{Type: EventTypeError, Client: c, Err: err})
					log.Printf("error writing to websocket: %v", err)
					c.handleReconnect()
					return
				}
			}
		}
	}
}

func (c *Client) SendMessage(message []byte) {
	c.Mutex.Lock()
	defer c.Mutex.Unlock()
	select {
	case c.Send <- message:
	default:
		log.Printf("Send buffer is full, buffering message for client %s", c.ID)
		c.messageQueue.Enqueue(c.ID, message)
	}
}

func (c *Client) SendSignal(signal SignalMessage) {
	c.Mutex.Lock()
	defer c.Mutex.Unlock()
	select {
	case c.SignalChannel <- signal:
	default:
		log.Printf("Signal channel is full, dropping signal for client %s", c.ID)
	}
}

func (c *Client) handleReconnect() {
	if c.retryAttempts < c.maxRetryAttempts && !c.isClosed {
		c.retryAttempts++
		log.Printf("Attempting to reconnect client %s (attempt %d/%d)", c.ID, c.retryAttempts, c.maxRetryAttempts)
		c.emitEvent(Event{Type: EventTypeReconnect, Client: c})
		time.Sleep(c.reconnectInterval)
		c.reconnect()
	} else {
		log.Printf("Maximum reconnect attempts reached for client %s, disconnecting", c.ID)
		err := c.Conn.Close()
		if err != nil {
			log.Println("error closing connection:", err)
			return
		}
		c.isClosed = true
		close(c.Send)
		close(c.SignalChannel)
	}
}

func (c *Client) reconnect() {
	conn, _, err := websocket.DefaultDialer.Dial(c.getWebSocketURL(), nil)
	if err != nil {
		log.Printf("Error reconnecting client %s: %v", c.ID, err)
		return
	}
	c.Conn = conn

	messages, err := c.messageQueue.Dequeue(c.ID)
	if err != nil {
		return
	}

	for _, message := range messages {
		c.Mutex.Lock()
		err := c.Conn.WriteMessage(websocket.TextMessage, []byte{message})
		c.Mutex.Unlock()
		if err != nil {
			log.Printf("Error resending message for client %s: %v", c.ID, err)
			err := c.messageQueue.Enqueue(c.ID, []byte{message})
			if err != nil {
				log.Println("error re-enqueueing message:", err)
				return
			}
		}
	}

	c.retryAttempts = 0

	go c.ReadPump()
	go c.WritePump()
}

func (c *Client) getWebSocketURL() string {
	host := os.Getenv("HOST")
	if host == "" {
		host = "ws://localhost"
	}
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	return host + ":" + port + "/ws"
}

func (c *Client) emitEvent(event Event) {
	go c.handleEvent(event)
}

func (c *Client) handleEvent(event Event) {
	switch event.Type {
	case EventTypeMessage:
		log.Printf("New message from %s: %s", c.ID, string(event.Message))
	case EventTypeLogin:
		log.Printf("User %s logged in", c.ID)
	case EventTypeLogout:
		log.Printf("User %s logged out", c.ID)
	case EventTypeError:
		log.Printf("Error from %s: %v", c.ID, event.Err)
	case EventTypeReconnect:
		log.Printf("Attempting to reconnect client %s", c.ID)
	case EventTypeSignal:
		c.handleSignalMessage(event.Message)
	}
}

func (c *Client) handleSignalMessage(message []byte) {
	var signalMessage SignalMessage
	err := json.Unmarshal(message, &signalMessage)
	if err != nil {
		log.Printf("Error decoding signal message for client %s: %v", c.ID, err)
		return
	}

	switch signalMessage.Type {
	case "offer":
		log.Printf("Received offer from client %s: %s", c.ID, signalMessage.Content)
		// Handle the WebRTC offer
	case "answer":
		log.Printf("Received answer from client %s: %s", c.ID, signalMessage.Content)
		// Handle the WebRTC answer
	case "iceCandidate":
		log.Printf("Received ICE candidate from client %s: %s", c.ID, signalMessage.Content)
		// Handle the new ICE candidate
	default:
		log.Printf("Unknown signal message type from client %s: %s", c.ID, signalMessage.Type)
	}
}
