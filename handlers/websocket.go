package handlers

import (
	"encoding/json"
	"fmt"
	"github.com/4cecoder/multiplayer/models"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"log"
	"math"
	"math/rand"
	"net/http"
	"sync"
	"time"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:    1024,
	WriteBufferSize:   1024,
	CheckOrigin:       func(r *http.Request) bool { return true },
	EnableCompression: false, // Disable compression
}

var players = make(map[string]*models.Player)
var gameStateMutex sync.Mutex

const (
	FieldWidth  = 800
	FieldHeight = 600
)

func HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}
	// Get the client's IP address
	clientIP := r.RemoteAddr

	// Check if a player with the same IP already exists
	var existingPlayer *models.Player
	for _, p := range players {
		if p.Conn.RemoteAddr().String() == clientIP {
			existingPlayer = p
			break
		}
	}

	playerID := uuid.New().String()
	if existingPlayer != nil {
		// Remove the old player
		handlePlayerRemoval(existingPlayer)
		playerID = existingPlayer.ID
	}

	// Generate random color for the player
	playerColor := fmt.Sprintf("#%06x", rand.Intn(0xFFFFFF))

	landCapture := make([][]bool, FieldHeight/20)
	for i := range landCapture {
		landCapture[i] = make([]bool, FieldWidth/20)
	}
	startingLand := make([][]bool, 3)
	for i := range startingLand {
		startingLand[i] = make([]bool, 3)
		for j := range startingLand[i] {
			startingLand[i][j] = true
			landCapture[FieldHeight/40+i][FieldWidth/40+j] = true
		}
	}
	player := &models.Player{
		ID:               playerID,
		StartingPosition: models.Point{X: FieldWidth / 2, Y: FieldHeight / 2},
		Name:             "Player",
		Color:            playerColor,
		X:                FieldWidth / 2,
		Y:                FieldHeight / 2,
		Acceleration:     0.13,
		MaxVelocity:      5,
		Conn:             conn,
		LandCapture:      landCapture,
		StartingLand:     startingLand,
		IsAlive:          true,
		WriteChan:        make(chan models.RenderInstruction, 10), // Buffer size can be adjusted
	}
	players[playerID] = player

	go func() {
		for instruction := range player.WriteChan {
			jsonData, err := json.Marshal(instruction)
			if err != nil {
				log.Println("marshal:", err)
				continue
			}

			err = player.Conn.WriteMessage(websocket.TextMessage, jsonData)
			if err != nil {
				log.Println("write:", err)
				player.Conn.Close()
				delete(players, player.ID)
				return
			}
		}
	}()

	go func() {
		for {
			if player.VelocityX == 0 && player.VelocityY == 0 {
				continue
			}
			handlePlayerMovement(player, "")
			err := sendRenderInstruction(player)
			if err != nil {
				log.Println("error sending render instruction:", err)
				return
			}
			time.Sleep(100 * time.Millisecond) // adjust the sleep duration to control the speed of the player
		}
	}()

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			log.Println("read:", err)
			break
		}
		handlePlayerMovement(player, string(message))
		err = sendRenderInstruction(player)
		if err != nil {
			log.Println("error sending render instruction:", err)
			break
		}
	}

	delete(players, playerID)
	err = conn.Close()
	if err != nil {
		log.Println("close:", err)
	}
}

func updatePlayerPosition(player *models.Player) {
	newX, newY := player.X+player.VelocityX, player.Y+player.VelocityY
	if newX < 0 {
		newX = 0
	} else if newX > FieldWidth-20 {
		newX = FieldWidth - 20
	}
	if newY < 0 {
		newY = 0
	} else if newY > FieldHeight-20 {
		newY = FieldHeight - 20
	}
	player.X, player.Y = newX, newY
}

func sendRenderInstruction(player *models.Player) error {
	gameStateMutex.Lock()
	defer gameStateMutex.Unlock()

	playerState := models.PlayerState{
		ID:               player.ID,
		StartingPosition: player.StartingPosition,
		Name:             player.Name,
		Color:            player.Color,
		X:                player.X,
		Y:                player.Y,
		VelocityX:        player.VelocityX,
		VelocityY:        player.VelocityY,
		LandCapture:      player.LandCapture,
		PlayerTrail:      player.PlayerTrail,
		StartingLand:     player.StartingLand,
		IsAlive:          player.IsAlive,
	}

	instruction := models.RenderInstruction{
		Type:    "updatePlayer",
		Payload: playerState,
	}

	player.WriteChan <- instruction
	return nil
}

func handlePlayerMovement(player *models.Player, message string) {
	// Calculate speed multiplier based on kill streak
	player.SpeedMultiplier = 1 + float64(player.KillStreak)*0.01
	if player.SpeedMultiplier > 1.09 { // Cap the speed multiplier at 1.09
		player.SpeedMultiplier = 1.09
	}

	if message != "" {
		switch message {
		case "up":
			if player.VelocityY >= 0 {
				player.VelocityY = -player.MaxVelocity * player.SpeedMultiplier
				player.VelocityX = 0
			}
		case "down":
			if player.VelocityY <= 0 {
				player.VelocityY = player.MaxVelocity * player.SpeedMultiplier
				player.VelocityX = 0
			}
		case "left":
			if player.VelocityX >= 0 {
				player.VelocityX = -player.MaxVelocity * player.SpeedMultiplier
				player.VelocityY = 0
			}
		case "right":
			if player.VelocityX <= 0 {
				player.VelocityX = player.MaxVelocity * player.SpeedMultiplier
				player.VelocityY = 0
			}
		case "stop":
			player.VelocityX = 0
			player.VelocityY = 0
		}
	}
	updatePlayerPosition(player)

	// Add the current position to the player's trail
	player.PlayerTrail = append(player.PlayerTrail, models.Point{X: player.X, Y: player.Y})

	// Check if the player has run into their own trail (at least 40px away from the back)
	if len(player.PlayerTrail) > 2 {
		for i := 0; i < len(player.PlayerTrail)-2; i++ {
			if math.Sqrt(math.Pow(player.PlayerTrail[i].X-player.X, 2)+math.Pow(player.PlayerTrail[i].Y-player.Y, 2)) >= 40 {
				if int(player.X) == int(player.PlayerTrail[i].X) && int(player.Y) == int(player.PlayerTrail[i].Y) {
					// The player has run into their own trail
					handlePlayerDeath(player)
					return
				}
			}
		}
	}

	// Check if the player has run into another player's trail
	for _, otherPlayer := range players {
		if otherPlayer.ID != player.ID {
			for _, point := range otherPlayer.PlayerTrail {
				if int(player.X) == int(point.X) && int(player.Y) == int(point.Y) {
					// The player has run into another player's trail
					handlePlayerDeath(player)
					otherPlayer.KillStreak++
					// Transfer the player's territory to the other player
					for i := range player.LandCapture {
						for j := range player.LandCapture[i] {
							if player.LandCapture[i][j] {
								otherPlayer.LandCapture[i][j] = true
							}
						}
					}
					return
				}
			}
		}
	}
}

func handlePlayerDeath(player *models.Player) {
	player.IsAlive = false

	// Convert the player's territory to neutral
	for i := range player.LandCapture {
		for j := range player.LandCapture[i] {
			player.LandCapture[i][j] = false
		}
	}

	// Send a "removePlayer" instruction to the player's WriteChan
	playerState := models.PlayerState{
		ID:               player.ID,
		StartingPosition: player.StartingPosition,
		Name:             player.Name,
		Color:            player.Color,
		X:                player.X,
		Y:                player.Y,
		VelocityX:        player.VelocityX,
		VelocityY:        player.VelocityY,
		LandCapture:      player.LandCapture,
		PlayerTrail:      player.PlayerTrail,
		StartingLand:     player.StartingLand,
		IsAlive:          player.IsAlive,
	}

	instruction := models.RenderInstruction{
		Type:    "removePlayer",
		Payload: playerState,
	}

	player.WriteChan <- instruction

	// Disconnect the player from the game
	err := player.Conn.Close()
	if err != nil {
		log.Println("close:", err)
	}

	// Remove the player from the server-side game state
	delete(players, player.ID)
}

func isEncapsulated(player *models.Player, x, y int) bool {
	trail := player.PlayerTrail
	crossings := 0
	for i := 0; i < len(trail); i++ {
		p1 := trail[i]
		p2 := trail[(i+1)%len(trail)]
		if (p1.Y > float64(y)) != (p2.Y > float64(y)) &&
			float64(x) < (p2.X-p1.X)*(float64(y)-p1.Y)/(p2.Y-p1.Y)+p1.X {
			crossings++
		}
	}
	return crossings%2 != 0
}

func handlePlayerRemoval(player *models.Player) {
	// Send a "removePlayer" instruction to all other connected players
	for _, p := range players {
		if p.ID != player.ID {
			instruction := models.RenderInstruction{
				Type:    "removePlayer",
				Payload: models.PlayerState{ID: player.ID},
			}
			p.WriteChan <- instruction
		}
	}

	// Remove the player from the game state
	delete(players, player.ID)
	err := player.Conn.Close()
	if err != nil {
		log.Println("close:", err)
	}
}
