package main

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:    1024,
	WriteBufferSize:   1024,
	CheckOrigin:       func(r *http.Request) bool { return true },
	EnableCompression: true,
}

type Player struct {
	ID           string          `json:"id"`
	Name         string          `json:"name"`
	Color        string          `json:"color"`
	X            float64         `json:"x"`
	Y            float64         `json:"y"`
	VelocityX    float64         `json:"velocityX"`
	VelocityY    float64         `json:"velocityY"`
	Acceleration float64         `json:"-"`
	MaxVelocity  float64         `json:"-"`
	Conn         *websocket.Conn `json:"-"`
}

type GameState struct {
	Players []Player `json:"players"`
}

var players = make(map[string]*Player)
var previousGameState GameState
var gameStateMutex sync.Mutex

func main() {
	setupServer()
}

func setupServer() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./static/index.html")
	})

	http.HandleFunc("/ws", handleWebSocket)

	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	log.Println("Server started on :8080")
	err := http.ListenAndServe("0.0.0.0:8080", nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}

	playerID := uuid.New().String()
	player := &Player{
		ID:           playerID,
		Name:         "Player",
		Color:        "#00ff00",
		X:            0,
		Y:            0,
		Acceleration: 0.13,
		MaxVelocity:  5,
		Conn:         conn,
	}
	players[playerID] = player

	var wg sync.WaitGroup
	wg.Add(2) // One for the reader goroutine, one for the game loop

	go func() {
		defer wg.Done()
		defer func() {
			if r := recover(); r != nil {
				log.Println("Recovered from panic:", r)
			}
		}()
		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				log.Println("read:", err)
				break
			}

			var msg map[string]string
			err = json.Unmarshal(message, &msg)
			if err == nil && msg["type"] == "updatePlayer" {
				player.Name = msg["name"]
				player.Color = msg["color"]
			} else {
				handlePlayerMovement(player, string(message))
			}
		}
	}()

	go func() {
		defer wg.Done()
		defer func() {
			if r := recover(); r != nil {
				log.Println("Recovered from panic:", r)
			}
		}()
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				updatePlayerPositions()
				sendGameState()
			}
		}
	}()

	wg.Wait() // Wait for all goroutines to finish

	delete(players, playerID)
	err = conn.Close()
	if err != nil {
		log.Println("close:", err)
	}
}

const (
	FieldWidth  = 800
	FieldHeight = 600
)

func handlePlayerMovement(player *Player, message string) {
	switch message {
	case "up":
		player.VelocityY -= player.Acceleration
	case "down":
		player.VelocityY += player.Acceleration
	case "left":
		player.VelocityX -= player.Acceleration
	case "right":
		player.VelocityX += player.Acceleration
	case "stop":
		player.VelocityX = 0
		player.VelocityY = 0
	}
}

func updatePlayerPositions() {
	for _, player := range players {
		// Limit the velocity to the maximum velocity
		if player.VelocityX > player.MaxVelocity {
			player.VelocityX = player.MaxVelocity
		} else if player.VelocityX < -player.MaxVelocity {
			player.VelocityX = -player.MaxVelocity
		}
		if player.VelocityY > player.MaxVelocity {
			player.VelocityY = player.MaxVelocity
		} else if player.VelocityY < -player.MaxVelocity {
			player.VelocityY = -player.MaxVelocity
		}

		player.X += player.VelocityX * 10
		player.Y += player.VelocityY * 10

		if player.X < 0 {
			player.X = 0
		} else if player.X > FieldWidth-20 {
			player.X = FieldWidth - 20
		}

		if player.Y < 0 {
			player.Y = 0
		} else if player.Y > FieldHeight-20 {
			player.Y = FieldHeight - 20
		}
	}
}

func sendGameState() {
	gameStateMutex.Lock()
	defer gameStateMutex.Unlock()

	gameState := GameState{Players: gameStatePlayers()}
	sendMessage(gameState)
}

func sendMessage(message interface{}) {
	jsonData, err := json.Marshal(message)
	if err != nil {
		log.Println("marshal:", err)
		return
	}

	for id, player := range players {
		err := player.Conn.WriteMessage(websocket.TextMessage, jsonData)
		if err != nil {
			log.Println("write:", err)
			delete(players, id)
			err := player.Conn.Close()
			if err != nil {
				log.Println("close:", err)
			}
			continue
		}
	}
}

func gameStatePlayers() []Player {
	statePlayers := make([]Player, 0, len(players))
	for _, player := range players {
		statePlayers = append(statePlayers, *player)
	}
	return statePlayers
}
