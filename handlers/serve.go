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

// Validate direction for movement
func validateDirection(direction string) bool {
	switch direction {
	case "up", "down", "left", "right":
		return true
	default:
		return false
	}
}

// Update player velocity based on direction
func updateVelocity(player *models.Player, direction string) {
	playersMutex.Lock()
	defer playersMutex.Unlock()

	switch direction {
	case "up":
		player.VelocityY = -player.MaxVelocity * player.SpeedMultiplier
		player.VelocityX = 0
	case "down":
		player.VelocityY = player.MaxVelocity * player.SpeedMultiplier
		player.VelocityX = 0
	case "left":
		player.VelocityX = -player.MaxVelocity * player.SpeedMultiplier
		player.VelocityY = 0
	case "right":
		player.VelocityX = player.MaxVelocity * player.SpeedMultiplier
		player.VelocityY = 0
	}
}

// Handle move messages by updating velocity and broadcasting the update
func handleMoveMessage(client *Client, message models.RenderInstruction) {
	// turn direction interface into string
	direction, ok := message.Payload.Direction.(string)
	if !ok {
		log.Printf("Invalid direction %s for client %s", message.Payload.Direction, client.ID)
		return
	}

	if !validateDirection(direction) {
		log.Printf("Invalid direction %s for client %s", message.Payload.Direction, client.ID)
		return
	}

	log.Printf("Handling move direction %s for client %s", message.Payload.Direction, client.ID)
	updateVelocity(client.Player, direction)
	broadcastPlayerState(client.Player)
}

// Function to broadcast the new player state to all clients
func broadcastPlayerState(player *models.Player) {
	state := makePlayerState(player)
	playersMutex.Lock()
	defer playersMutex.Unlock()

	for _, client := range clients {
		client.Send <- state
	}
}

// Example function to create a JSON representation of the player's current state
func makePlayerState(player *models.Player) []byte {
	state, err := json.Marshal(player)
	if err != nil {
		log.Printf("Error marshaling player state: %v", err)
		return nil
	}
	return state
}

func handleClientMessages(client *Client) {
	for {
		event, ok := <-client.EventQueue
		if !ok {
			log.Println("Client disconnected, unregistering")
			unregisterClient(client)
			return
		}

		switch event.Type {
		case EventTypeMessage:
			handleMessageEvent(client, event.Message)
		case EventTypeMove:
			var moveMessage models.RenderInstruction
			if err := json.Unmarshal(event.Message, &moveMessage); err != nil {
				log.Printf("Error decoding move message from client %s: %v", client.ID, err)
				return
			}
			handleMoveMessage(client, moveMessage)
		default:
			log.Printf("Unhandled event type %d for client %s", event.Type, client.ID)
		}
	}
}

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
	log.Printf("Creating new player: %s", clientID)

	messageQueue := NewMessageQueue()
	client := NewClient(conn, clientID, messageQueue)

	// Create a new Player instance and associate it with the Client
	player := &models.Player{
		ID:               clientID,
		StartingPosition: models.Point{X: 0, Y: 0},
		Name:             "Player " + clientID,
		Color:            randomColor(),
		X:                0,
		Y:                0,
		VelocityX:        0,
		VelocityY:        0,
		Acceleration:     0.1,
		MaxVelocity:      5,
		Conn:             conn,
		LandCapture:      make([][]bool, 100, 100),
		PlayerTrail:      make([]models.Point, 0),
		StartingLand:     make([][]bool, 100, 100),
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
	log.Printf("Registering new client: %s", clientID)
	registerClient(client)

	go func() {
		log.Printf("Starting ReadPump for client: %s", clientID)
		client.ReadPump()
	}()

	go func() {
		log.Printf("Starting WritePump for client: %s", clientID)
		client.WritePump()
	}()

	go func() {
		log.Printf("Starting handleClientMessages for client: %s", clientID)
		handleClientMessages(client)
	}()

	log.Printf("Broadcasting new player: %s", clientID)
	broadcastNewPlayer(player)
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

//func handleClientMessages(client *Client) {
//	for {
//		event, ok := <-client.EventQueue
//		if !ok {
//			// The event queue was closed, client has disconnected
//			unregisterClient(client)
//			return
//		}
//
//		switch event.Type {
//		case EventTypeMessage:
//			handleMessageEvent(client, event.Message)
//		case EventTypeError:
//			log.Printf("Error from client %s: %v", client.ID, event.Err)
//		case EventTypeReconnect:
//			// Handle reconnect event
//			log.Printf("Attempting to reconnect client %s", client.ID)
//		case EventTypeMove:
//			// Handle move event
//			log.Printf("Received move signal from client %s: %s", client.ID, string(event.Message))
//			handleMoveEvent(client, event.Message)
//		default:
//			panic("unhandled default case")
//		}
//	}
//}

func handleMoveEvent(client *Client, message []byte) {
	// Decode the message
	var gameMessage models.RenderInstruction
	err := json.Unmarshal(message, &gameMessage)
	if err != nil {
		log.Printf("Error decoding move message from client %s: %v", client.ID, err)
		return
	}

	// Handle the move message
	handleMoveMessage(client, gameMessage)
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

func broadcastPlayerUpdate(player *models.Player) {
	playerState := models.RenderInstruction{
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

	jsonMessage, err := json.Marshal(playerState)
	if err != nil {
		log.Println("error marshalling player update message:", err)
		return
	}

	for _, client := range clients {
		if client.Conn != nil {
			client.SendMessage(jsonMessage)
		}
	}
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
