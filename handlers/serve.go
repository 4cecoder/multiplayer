// Package handlers serve.go
package handlers

import (
	"encoding/json"
	"github.com/4cecoder/multiplayer/models"
	"github.com/google/uuid"
	"log"
	"math"
	"math/rand"
	"net/http"
	"sync"
)

const (
	FieldHeight = 600
	FieldWidth  = 800
)

var players = make(map[string]*models.Player)
var playersMutex sync.Mutex

func ServeWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Error upgrading to WebSocket:", err)
		return
	}

	clientID := r.Header.Get("X-Client-ID")
	if clientID == "" {
		clientID = generateClientID()
	}

	messageQueue := NewMessageQueue()
	client := NewClient(conn, clientID, messageQueue)

	// Create a new Player instance and associate it with the Client
	player := &models.Player{
		ID:               clientID,
		StartingPosition: models.Point{X: 0, Y: 0}, // Set the starting position
		Name:             "Player " + clientID,
		Color:            randomColor(),
		X:                0,
		Y:                0,
		VelocityX:        0,
		VelocityY:        0,
		Acceleration:     0.1,
		MaxVelocity:      5,
		Conn:             conn,
		LandCapture:      make([][]bool, 100, 100), // Initialize the land capture grid
		PlayerTrail:      make([]models.Point, 0),
		StartingLand:     make([][]bool, 100, 100), // Initialize the starting land grid
		IsAlive:          true,
		KillStreak:       0,
		SpeedMultiplier:  1,
		WriteChan:        make(chan models.RenderInstruction, 16),
	}
	client.Player = player

	// Add the player to the players map
	playersMutex.Lock()
	players[clientID] = player
	playersMutex.Unlock()

	registerClient(client)

	go client.ReadPump()
	go client.WritePump()
	go handleClientMessages(client)
}

func registerClient(client *Client) {
	clientsMutex.Lock()         // Lock the mutex before accessing the map
	defer clientsMutex.Unlock() // Ensure the mutex is unlocked at the end of the function

	clients[client.ID] = client // Add the client to the map
	log.Printf("Registered new client: %s", client.ID)
}

func unregisterClient(client *Client) {
	clientsMutex.Lock()         // Lock the mutex before accessing the map
	defer clientsMutex.Unlock() // Ensure the mutex is unlocked at the end of the function

	delete(clients, client.ID) // Remove the client from the map
	log.Printf("Unregistered client: %s", client.ID)
	// Additional cleanup can be added here
}

func randomColor() string {
	colors := []string{"#FF0000", "#00FF00", "#0000FF", "#FFFF00", "#00FFFF", "#FF00FF"}
	return colors[rand.Intn(len(colors))]
}

func generateClientID() string {
	// just use uuid for now
	return uuid.New().String()
}

func handleClientMessages(client *Client) {
	for {
		event, ok := <-client.EventQueue
		if !ok {
			// The event queue was closed, client has disconnected
			unregisterClient(client)
			return
		}

		switch event.Type {
		case EventTypeMessage:
			handleMessageEvent(client, event.Message)
		case EventTypeError:
			log.Printf("Error from client %s: %v", client.ID, event.Err)
		case EventTypeReconnect:
			// Handle reconnect event
			log.Printf("Attempting to reconnect client %s", client.ID)
		default:
			panic("unhandled default case")
		}
	}
}

func handleMessageEvent(client *Client, message []byte) {
	// Decode the message
	var gameMessage models.RenderInstruction
	err := json.Unmarshal(message, &gameMessage)
	if err != nil {
		log.Printf("Error decoding message from client %s: %v", client.ID, err)
		return
	}

	// Handle the game message based on the type
	switch gameMessage.Type {
	case "move":
		handleMoveMessage(client, gameMessage)
	case "capture":
		handleCaptureMessage(client, gameMessage)
	case "chat":
		handleChatMessage(client, gameMessage)
	case "join":
		// Broadcast the new player's information to all other clients
		broadcastNewPlayer(client.Player)
	default:
		log.Printf("Unknown game message type from client %s: %s", client.ID, gameMessage.Type)
	}
}

func handleMoveMessage(client *Client, message models.RenderInstruction) {
	// Calculate speed multiplier based on kill streak
	client.Player.SpeedMultiplier = 1 + float64(client.Player.KillStreak)*0.01
	if client.Player.SpeedMultiplier > 1.09 { // Cap the speed multiplier at 1.09
		client.Player.SpeedMultiplier = 1.09
	}

	// Handle player movement based on the message
	switch message.Payload.Direction {
	case "up":
		if client.Player.VelocityY >= 0 {
			client.Player.VelocityY = -client.Player.MaxVelocity * client.Player.SpeedMultiplier
			client.Player.VelocityX = 0
		}
	case "down":
		if client.Player.VelocityY <= 0 {
			client.Player.VelocityY = client.Player.MaxVelocity * client.Player.SpeedMultiplier
			client.Player.VelocityX = 0
		}
	case "left":
		if client.Player.VelocityX >= 0 {
			client.Player.VelocityX = -client.Player.MaxVelocity * client.Player.SpeedMultiplier
			client.Player.VelocityY = 0
		}
	case "right":
		if client.Player.VelocityX <= 0 {
			client.Player.VelocityX = client.Player.MaxVelocity * client.Player.SpeedMultiplier
			client.Player.VelocityY = 0
		}
	case "stop":
		client.Player.VelocityX = 0
		client.Player.VelocityY = 0
	}

	updatePlayerPosition(client.Player)
	// New logic to check and capture territory
	newPoint := models.Point{X: client.Player.X, Y: client.Player.Y}
	client.Player.PlayerTrail = append(client.Player.PlayerTrail, newPoint) // Update trail with new position
	checkAndCaptureTerritory(client.Player.ID, newPoint)                    // Check for and handle territory encapsulation

	// Add the current position to the player's trail
	client.Player.PlayerTrail = append(client.Player.PlayerTrail, models.Point{X: client.Player.X, Y: client.Player.Y})

	// Check if the player has run into their own trail (at least 40px away from the back)
	checkPlayerTrailCollision(client.Player)

	// Check if the player has run into another player's trail
	checkPlayerTrailCollisions(client)
}

func handleCaptureMessage(client *Client, message models.RenderInstruction) {
	// Implement your game logic for handling capture messages
	log.Printf("Received capture message from client %s: %+v", client.ID, message.Payload)
	// Update the player's land capture based on the message
	client.Player.LandCapture = message.Payload.LandCapture
	// Check for and handle territory encapsulation
	checkAndCaptureTerritory(client.Player.ID, models.Point{X: client.Player.X, Y: client.Player.Y})
}

func handleChatMessage(client *Client, message models.RenderInstruction) {
	// Implement your game logic for handling chat messages
	log.Printf("Received chat message from client %s: %+v", client.ID, message.Payload)
	// Handle the chat message as needed
}

// updatePlayerPosition updates the player's position based on their velocity.
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

// checkAndCaptureTerritory checks if the player has encapsulated an area and captures the territory.
func checkAndCaptureTerritory(playerID string, newPos models.Point) {
	player := players[playerID]
	if player == nil {
		log.Printf("Player %s not found in checkAndCaptureTerritory", playerID)
		return
	}

	player.PlayerTrail = append(player.PlayerTrail, newPos) // Append new position to trail

	// Detect if the trail closes a loop
	if loopClosed(player.PlayerTrail) {
		capturedPoints := calculateEnclosedArea(player.PlayerTrail)
		updatePlayerLand(player, capturedPoints)
		broadcastCapture(player, capturedPoints)
	}
}

func loopClosed(points []models.Point) bool {
	// Example logic to determine if the points form a loop
	if len(points) < 4 {
		return false
	}
	return points[0] == points[len(points)-1]
}

// updatePlayerLand updates the player's land based on captured territory
func updatePlayerLand(player *models.Player, capturedPoints []models.Point) {
	// Merge captured land with player's existing land
	for _, point := range capturedPoints {
		// Assuming LandCapture is adequately sized and coordinates are valid
		x, y := int(point.X), int(point.Y)
		if x >= 0 && x < len(player.LandCapture) && y >= 0 && y < len(player.LandCapture[x]) {
			player.LandCapture[x][y] = true
		}
	}
}

// calculateEnclosedArea calculates the points inside the loop defined by the player's trail.
func calculateEnclosedArea(trail []models.Point) []models.Point {
	if len(trail) < 3 {
		return nil // A valid closed area requires at least three points
	}

	var minX, maxX, minY, maxY float64
	minX, maxX = trail[0].X, trail[0].X
	minY, maxY = trail[0].Y, trail[0].Y

	// Determine the bounding box of the trail
	for _, point := range trail {
		if point.X < minX {
			minX = point.X
		}
		if point.X > maxX {
			maxX = point.X
		}
		if point.Y < minY {
			minY = point.Y
		}
		if point.Y > maxY {
			maxY = point.Y
		}
	}

	// Prepare to collect enclosed points
	var enclosedPoints []models.Point

	// Check each point in the bounding box
	for x := math.Floor(minX); x <= math.Ceil(maxX); x++ {
		for y := math.Floor(minY); y <= math.Ceil(maxY); y++ {
			if isPointInsidePolygon(x, y, trail) {
				enclosedPoints = append(enclosedPoints, models.Point{X: x, Y: y})
			}
		}
	}

	return enclosedPoints
}

// isPointInsidePolygon checks if a point is inside a polygon using the ray-casting method.
func isPointInsidePolygon(x, y float64, polygon []models.Point) bool {
	count := 0
	n := len(polygon)
	for i := 0; i < n; i++ {
		j := (i + 1) % n
		p1 := polygon[i]
		p2 := polygon[j]
		if (p1.Y > y) != (p2.Y > y) && (x < (p2.X-p1.X)*(y-p1.Y)/(p2.Y-p1.Y)+p1.X) {
			count++
		}
	}
	return count%2 != 0
}

// checkPlayerTrailCollision checks if the player has run into their own trail.
func checkPlayerTrailCollision(player *models.Player) {
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
}

func handlePlayerDeath(player *models.Player) {
	playersMutex.Lock()
	defer playersMutex.Unlock()

	delete(players, player.ID)
	log.Printf("Player %s has died and is removed from the game", player.ID)

	// Additional cleanup actions, such as notifying other players
}

// checkPlayerTrailCollisions checks if the player has run into another player's trail.
func checkPlayerTrailCollisions(client *Client) {
	for _, otherPlayer := range clients {
		if otherPlayer.ID != client.ID {
			for _, point := range otherPlayer.Player.PlayerTrail {
				if int(client.Player.X) == int(point.X) && int(client.Player.Y) == int(point.Y) {
					// The player has run into another player's trail
					handlePlayerDeath(client.Player)
					otherPlayer.Player.KillStreak++
					// Transfer the player's territory to the other player
					for i := range client.Player.LandCapture {
						for j := range client.Player.LandCapture[i] {
							if client.Player.LandCapture[i][j] {
								otherPlayer.Player.LandCapture[i][j] = true
							}
						}
					}
					return
				}
			}
		}
	}
}

// broadcastCapture sends the captured territory to all clients
func broadcastCapture(player *models.Player, capturedPoints []models.Point) {
	// Converting capturedPoints into a format suitable for JSON or other client communication
	capturedForBroadcast := make([][]bool, len(player.LandCapture))
	for i := range capturedForBroadcast {
		capturedForBroadcast[i] = make([]bool, len(player.LandCapture[i]))
	}

	for _, point := range capturedPoints {
		x, y := int(point.X), int(point.Y)
		if x >= 0 && x < len(capturedForBroadcast) && y >= 0 && y < len(capturedForBroadcast[x]) {
			capturedForBroadcast[x][y] = true
		}
	}

	captureMessage := models.RenderInstruction{
		Type: "captureTerritory",
		Payload: models.PlayerState{
			ID:          player.ID,
			LandCapture: capturedForBroadcast,
		},
	}

	// Convert the captureMessage to a JSON byte slice
	jsonMessage, err := json.Marshal(captureMessage)
	if err != nil {
		log.Println("error marshalling captureMessage:", err)
		return
	}

	for _, client := range clients {
		// Check if the client's WebSocket connection is not nil before trying to write to it
		if client.Conn != nil {
			client.SendMessage(jsonMessage)

		}
	}
}

func broadcastNewPlayer(player *models.Player) {
	newPlayerMessage := models.RenderInstruction{
		Type: "updatePlayer",
		Payload: models.PlayerState{
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
			Direction:        nil,
		},
	}

	jsonMessage, err := json.Marshal(newPlayerMessage)
	if err != nil {
		log.Println("error marshalling new player message:", err)
		return
	}

	for _, client := range clients {
		if client.Conn != nil {
			client.SendMessage(jsonMessage)
		}
	}
}
