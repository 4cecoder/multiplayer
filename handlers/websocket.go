// Package handlers websocket.go contains the implementation of a WebSocket handler that manages player connections and game state.
package handlers

//
//import (
//	"encoding/json"
//	"fmt"
//	"github.com/4cecoder/multiplayer/models"
//	"github.com/google/uuid"
//	"github.com/gorilla/websocket"
//	"log"
//	"math"
//	"math/rand"
//	"net/http"
//	"sync"
//)
//
//var players = make(map[string]*models.Player)
//var gameStateMutex sync.Mutex
//
//const (
//	FieldWidth  = 800
//	FieldHeight = 600
//)
//
//func HandleWebSocket(w http.ResponseWriter, r *http.Request) {
//	conn, err := upgrader.Upgrade(w, r, nil)
//	if err != nil {
//		log.Println("Failed to upgrade WebSocket:", err)
//		return
//	}
//	defer func() {
//		if r := recover(); r != nil {
//			log.Println("Recovered from panic in WebSocket handler:", r)
//		}
//		if conn != nil {
//			err := conn.Close()
//			if err != nil {
//				log.Println("Error closing WebSocket connection:", err)
//			}
//		}
//	}()
//
//	// Get the client's IP address
//	clientIP := r.RemoteAddr
//
//	// Check if a player with the same IP already exists
//	existingPlayer := getPlayerByIP(clientIP)
//	if existingPlayer != nil {
//		// Remove the old player
//		handlePlayerRemoval(existingPlayer)
//	}
//
//	playerID := uuid.New().String()
//	playerColor := fmt.Sprintf("#%06x", rand.Intn(0xFFFFFF))
//
//	landCapture := make([][]bool, FieldHeight/20)
//	for i := range landCapture {
//		landCapture[i] = make([]bool, FieldWidth/20)
//	}
//	startingLand := make([][]bool, 3)
//	for i := range startingLand {
//		startingLand[i] = make([]bool, 3)
//		for j := range startingLand[i] {
//			startingLand[i][j] = true
//			landCapture[FieldHeight/40+i][FieldWidth/40+j] = true
//		}
//	}
//	player := &models.Player{
//		ID:               playerID,
//		StartingPosition: models.Point{X: FieldWidth / 2, Y: FieldHeight / 2},
//		Name:             "Player",
//		Color:            playerColor,
//		X:                FieldWidth / 2,
//		Y:                FieldHeight / 2,
//		Acceleration:     0.13,
//		MaxVelocity:      5,
//		Conn:             conn,
//		LandCapture:      landCapture,
//		StartingLand:     startingLand,
//		IsAlive:          true,
//		WriteChan:        make(chan models.RenderInstruction, 10), // Buffer size can be adjusted
//	}
//	players[playerID] = player
//
//	// Send the initial player state to the new client
//	initialPlayerState := models.PlayerState{
//		ID:               player.ID,
//		StartingPosition: player.StartingPosition,
//		Name:             player.Name,
//		Color:            player.Color,
//		X:                player.X,
//		Y:                player.Y,
//		VelocityX:        player.VelocityX,
//		VelocityY:        player.VelocityY,
//		LandCapture:      player.LandCapture,
//		PlayerTrail:      player.PlayerTrail,
//		StartingLand:     player.StartingLand,
//		IsAlive:          player.IsAlive,
//	}
//
//	initialInstruction := models.RenderInstruction{
//		Type:    "updatePlayer",
//		Payload: initialPlayerState,
//	}
//
//	err = sendRenderInstructionToClient(player, initialInstruction)
//	if err != nil {
//		log.Println("Error sending initial player state:", err)
//		return
//	}
//
//	go safeGoRoutine(func() {
//		for instruction := range player.WriteChan {
//			err := sendRenderInstructionToClient(player, instruction)
//			if err != nil {
//				log.Println("Error sending render instruction:", err)
//				return
//			}
//		}
//	})
//
//	go safeGoRoutine(func() {
//		connectionListener(player)
//	})
//}
//
//func sendRenderInstructionToClient(player *models.Player, instruction models.RenderInstruction) error {
//	if player.Conn == nil {
//		log.Printf("Player %s WebSocket connection is nil, skipping render instruction", player.ID)
//		return nil
//	}
//
//	jsonData, err := json.Marshal(instruction)
//	if err != nil {
//		return err
//	}
//
//	err = player.Conn.WriteMessage(websocket.TextMessage, jsonData)
//	if err != nil {
//		return err
//	}
//
//	return nil
//}
//
//func getPlayerByIP(clientIP string) *models.Player {
//	gameStateMutex.Lock()
//	defer gameStateMutex.Unlock()
//
//	for _, p := range players {
//		if p.Conn != nil && p.Conn.RemoteAddr().String() == clientIP {
//			return p
//		}
//	}
//	return nil
//}
//
//func connectionListener(player *models.Player) {
//	defer func() {
//		if r := recover(); r != nil {
//			log.Printf("Recovered from panic in connection listener: %v", r)
//		}
//		cleanupPlayer(player)
//	}()
//
//	for {
//		_, message, err := player.Conn.ReadMessage()
//		if err != nil {
//			log.Printf("Error reading message: %v", err)
//			return
//		}
//		handlePlayerMovement(player, string(message))
//		err = sendRenderInstruction(player)
//		if err != nil {
//			log.Println("error sending render instruction:", err)
//			return
//		}
//	}
//}
//
//func cleanupPlayer(player *models.Player) {
//	if player != nil {
//		gameStateMutex.Lock()
//		defer gameStateMutex.Unlock()
//
//		if player.Conn != nil {
//			err := player.Conn.Close()
//			if err != nil {
//				log.Println("Error closing WebSocket connection:", err)
//			}
//		}
//		delete(players, player.ID)
//	}
//}
//
//func safeGoRoutine(f func()) {
//	go func() {
//		defer func() {
//			if r := recover(); r != nil {
//				log.Println("Recovered in f", r)
//			}
//		}()
//		f()
//	}()
//}
//
//func updatePlayerPosition(player *models.Player) {
//	newX, newY := player.X+player.VelocityX, player.Y+player.VelocityY
//	if newX < 0 {
//		newX = 0
//	} else if newX > FieldWidth-20 {
//		newX = FieldWidth - 20
//	}
//	if newY < 0 {
//		newY = 0
//	} else if newY > FieldHeight-20 {
//		newY = FieldHeight - 20
//	}
//	player.X, player.Y = newX, newY
//}
//
//func sendRenderInstruction(player *models.Player) error {
//	gameStateMutex.Lock()
//	defer gameStateMutex.Unlock()
//
//	if player.Conn == nil {
//		log.Printf("Player %s WebSocket connection is nil, skipping render instruction", player.ID)
//		return nil
//	}
//
//	playerState := models.PlayerState{
//		ID:               player.ID,
//		StartingPosition: player.StartingPosition,
//		Name:             player.Name,
//		Color:            player.Color,
//		X:                player.X,
//		Y:                player.Y,
//		VelocityX:        player.VelocityX,
//		VelocityY:        player.VelocityY,
//		LandCapture:      player.LandCapture,
//		PlayerTrail:      player.PlayerTrail,
//		StartingLand:     player.StartingLand,
//		IsAlive:          player.IsAlive,
//	}
//
//	instruction := models.RenderInstruction{
//		Type:    "updatePlayer",
//		Payload: playerState,
//	}
//
//	player.WriteChan <- instruction
//	return nil
//}
//
//func handlePlayerMovement(player *models.Player, message string) {
//	// Calculate speed multiplier based on kill streak
//	player.SpeedMultiplier = 1 + float64(player.KillStreak)*0.01
//	if player.SpeedMultiplier > 1.09 { // Cap the speed multiplier at 1.09
//		player.SpeedMultiplier = 1.09
//	}
//
//	if message != "" {
//		switch message {
//		case "up":
//			if player.VelocityY >= 0 {
//				player.VelocityY = -player.MaxVelocity * player.SpeedMultiplier
//				player.VelocityX = 0
//			}
//		case "down":
//			if player.VelocityY <= 0 {
//				player.VelocityY = player.MaxVelocity * player.SpeedMultiplier
//				player.VelocityX = 0
//			}
//		case "left":
//			if player.VelocityX >= 0 {
//				player.VelocityX = -player.MaxVelocity * player.SpeedMultiplier
//				player.VelocityY = 0
//			}
//		case "right":
//			if player.VelocityX <= 0 {
//				player.VelocityX = player.MaxVelocity * player.SpeedMultiplier
//				player.VelocityY = 0
//			}
//		case "stop":
//			player.VelocityX = 0
//			player.VelocityY = 0
//		}
//	}
//	updatePlayerPosition(player)
//	// New logic to check and capture territory
//	newPoint := models.Point{X: player.X, Y: player.Y}
//	player.PlayerTrail = append(player.PlayerTrail, newPoint) // Update trail with new position
//	checkAndCaptureTerritory(player.ID, newPoint)             // Check for and handle territory encapsulation
//
//	// Add the current position to the player's trail
//	player.PlayerTrail = append(player.PlayerTrail, models.Point{X: player.X, Y: player.Y})
//
//	// Check if the player has run into their own trail (at least 40px away from the back)
//	if len(player.PlayerTrail) > 2 {
//		for i := 0; i < len(player.PlayerTrail)-2; i++ {
//			if math.Sqrt(math.Pow(player.PlayerTrail[i].X-player.X, 2)+math.Pow(player.PlayerTrail[i].Y-player.Y, 2)) >= 40 {
//				if int(player.X) == int(player.PlayerTrail[i].X) && int(player.Y) == int(player.PlayerTrail[i].Y) {
//					// The player has run into their own trail
//					handlePlayerDeath(player)
//					return
//				}
//			}
//		}
//	}
//
//	// Check if the player has run into another player's trail
//	for _, otherPlayer := range players {
//		if otherPlayer.ID != player.ID {
//			for _, point := range otherPlayer.PlayerTrail {
//				if int(player.X) == int(point.X) && int(player.Y) == int(point.Y) {
//					// The player has run into another player's trail
//					handlePlayerDeath(player)
//					otherPlayer.KillStreak++
//					// Transfer the player's territory to the other player
//					for i := range player.LandCapture {
//						for j := range player.LandCapture[i] {
//							if player.LandCapture[i][j] {
//								otherPlayer.LandCapture[i][j] = true
//							}
//						}
//					}
//					return
//				}
//			}
//		}
//	}
//}
//
//func handlePlayerDeath(player *models.Player) {
//	player.IsAlive = false
//
//	// Convert the player's territory to neutral
//	for i := range player.LandCapture {
//		for j := range player.LandCapture[i] {
//			player.LandCapture[i][j] = false
//		}
//	}
//
//	// Send a "removePlayer" instruction to the player's WriteChan
//	playerState := models.PlayerState{
//		ID:               player.ID,
//		StartingPosition: player.StartingPosition,
//		Name:             player.Name,
//		Color:            player.Color,
//		X:                player.X,
//		Y:                player.Y,
//		VelocityX:        player.VelocityX,
//		VelocityY:        player.VelocityY,
//		LandCapture:      player.LandCapture,
//		PlayerTrail:      player.PlayerTrail,
//		StartingLand:     player.StartingLand,
//		IsAlive:          player.IsAlive,
//	}
//
//	instruction := models.RenderInstruction{
//		Type:    "removePlayer",
//		Payload: playerState,
//	}
//
//	if player.Conn != nil {
//		player.WriteChan <- instruction
//	} else {
//		log.Printf("Player %s WebSocket connection is nil, skipping remove player instruction", player.ID)
//	}
//
//	// Disconnect the player from the game
//	cleanupPlayer(player)
//}
//
//func isEncapsulated(player *models.Player, x, y int) bool {
//	trail := player.PlayerTrail
//	crossings := 0
//	for i := 0; i < len(trail); i++ {
//		p1 := trail[i]
//		p2 := trail[(i+1)%len(trail)]
//		if (p1.Y > float64(y)) != (p2.Y > float64(y)) &&
//			float64(x) < (p2.X-p1.X)*(float64(y)-p1.Y)/(p2.Y-p1.Y)+p1.X {
//			crossings++
//		}
//	}
//	return crossings%2 != 0
//}
//
//func checkEncapsulation(player *models.Player) {
//	// Check if the player's current position connects with their old trail
//	for i := 0; i < len(player.PlayerTrail)-1; i++ {
//		if int(player.X) == int(player.PlayerTrail[i].X) && int(player.Y) == int(player.PlayerTrail[i].Y) {
//			// The player's current position connects with their old trail
//
//			// Update the LandCapture array to mark the encapsulated area as captured
//			for y := 0; y < len(player.LandCapture); y++ {
//				for x := 0; x < len(player.LandCapture[y]); x++ {
//					if isEncapsulated(player, x, y) {
//						player.LandCapture[y][x] = true
//					}
//				}
//			}
//
//			// Clear the player's trail
//			player.PlayerTrail = []models.Point{}
//
//			// Send a render instruction to update the client-side rendering
//			err := sendRenderInstruction(player)
//			if err != nil {
//				log.Println("error sending render instruction:", err)
//				return
//			}
//
//			break
//		}
//	}
//}
//
//func handlePlayerRemoval(player *models.Player) {
//	// Send a "removePlayer" instruction to all other connected players
//	for _, p := range players {
//		if p.ID != player.ID && p.Conn != nil {
//			instruction := models.RenderInstruction{
//				Type:    "removePlayer",
//				Payload: models.PlayerState{ID: player.ID},
//			}
//			err := p.Conn.WriteJSON(instruction)
//			if err != nil {
//				log.Printf("Error broadcasting remove player to player %s: %v", p.ID, err)
//				if websocket.IsCloseError(err) {
//					// If the connection is closed, remove the player from the players map
//					cleanupPlayer(p)
//				}
//			}
//		}
//	}
//
//	// Remove the player from the game state
//	cleanupPlayer(player)
//}
//
//// checkAndCaptureTerritory checks for loops in the player's path and captures territory.
//func checkAndCaptureTerritory(playerID string, newPos models.Point) {
//	player := players[playerID]
//	if player == nil {
//		log.Printf("Player %s not found in checkAndCaptureTerritory", playerID)
//		return
//	}
//
//	player.PlayerTrail = append(player.PlayerTrail, newPos) // Append new position to trail
//
//	// Detect if the trail closes a loop
//	if loopClosed(player.PlayerTrail) {
//		capturedPoints := calculateEnclosedArea(player.PlayerTrail)
//		updatePlayerLand(player, capturedPoints)
//		broadcastCapture(player, capturedPoints)
//	}
//}
//
//// loopClosed checks if the last point in the trail closes a loop with any previous part of the trail
//func loopClosed(trail []models.Point) bool {
//	// Simplistic check: loop is closed if the last point equals the starting point
//	lastPoint := trail[len(trail)-1]
//	return lastPoint == trail[0]
//}
//
//// calculateEnclosedArea calculates the points inside the loop defined by the player's trail.
//func calculateEnclosedArea(trail []models.Point) []models.Point {
//	if len(trail) < 3 {
//		return nil // A valid closed area requires at least three points
//	}
//
//	var minX, maxX, minY, maxY float64
//	minX, maxX = trail[0].X, trail[0].X
//	minY, maxY = trail[0].Y, trail[0].Y
//
//	// Determine the bounding box of the trail
//	for _, point := range trail {
//		if point.X < minX {
//			minX = point.X
//		}
//		if point.X > maxX {
//			maxX = point.X
//		}
//		if point.Y < minY {
//			minY = point.Y
//		}
//		if point.Y > maxY {
//			maxY = point.Y
//		}
//	}
//
//	// Prepare to collect enclosed points
//	var enclosedPoints []models.Point
//
//	// Check each point in the bounding box
//	for x := math.Floor(minX); x <= math.Ceil(maxX); x++ {
//		for y := math.Floor(minY); y <= math.Ceil(maxY); y++ {
//			if isPointInsidePolygon(x, y, trail) {
//				enclosedPoints = append(enclosedPoints, models.Point{X: x, Y: y})
//			}
//		}
//	}
//
//	return enclosedPoints
//}
//
//// isPointInsidePolygon checks if a point is inside a polygon using the ray-casting method.
//func isPointInsidePolygon(x, y float64, polygon []models.Point) bool {
//	count := 0
//	n := len(polygon)
//	for i := 0; i < n; i++ {
//		j := (i + 1) % n
//		p1 := polygon[i]
//		p2 := polygon[j]
//		if (p1.Y > y) != (p2.Y > y) && (x < (p2.X-p1.X)*(y-p1.Y)/(p2.Y-p1.Y)+p1.X) {
//			count++
//		}
//	}
//	return count%2 != 0
//}
//
//// updatePlayerLand updates the player's land based on captured territory
//func updatePlayerLand(player *models.Player, capturedPoints []models.Point) {
//	// Merge captured land with player's existing land
//	for _, point := range capturedPoints {
//		// Assuming LandCapture is adequately sized and coordinates are valid
//		x, y := int(point.X), int(point.Y)
//		if x >= 0 && x < len(player.LandCapture) && y >= 0 && y < len(player.LandCapture[x]) {
//			player.LandCapture[x][y] = true
//		}
//	}
//}
//
//// broadcastCapture sends the captured territory to all clients
//func broadcastCapture(player *models.Player, capturedPoints []models.Point) {
//	// Converting capturedPoints into a format suitable for JSON or other client communication
//	if len(player.LandCapture) == 0 {
//		log.Println("LandCapture is empty or uninitialized")
//		return
//	}
//
//	capturedForBroadcast := make([][]bool, len(player.LandCapture))
//	for i := range capturedForBroadcast {
//		if len(player.LandCapture[i]) == 0 {
//			log.Println("Sub-slice of LandCapture is empty or uninitialized")
//			continue
//		}
//		capturedForBroadcast[i] = make([]bool, len(player.LandCapture[i]))
//	}
//
//	for _, point := range capturedPoints {
//		x, y := int(point.X), int(point.Y)
//		if x >= 0 && x < len(capturedForBroadcast) && y >= 0 && y < len(capturedForBroadcast[x]) {
//			capturedForBroadcast[x][y] = true
//		} else {
//			log.Printf("Point out of bounds: %d, %d", x, y)
//		}
//	}
//
//	captureMessage := models.RenderInstruction{
//		Type: "captureTerritory",
//		Payload: models.PlayerState{
//			ID:          player.ID,
//			LandCapture: capturedForBroadcast,
//		},
//	}
//
//	for id, conn := range players {
//		if conn.Conn == nil {
//			log.Printf("WebSocket connection for player %s is nil", id)
//			continue
//		}
//
//		err := conn.Conn.WriteJSON(captureMessage)
//		if err != nil {
//			log.Printf("Error broadcasting capture to player %s: %v", id, err)
//			if websocket.IsCloseError(err) {
//				// If the connection is closed, remove the player from the players map
//				cleanupPlayer(conn)
//			}
//		}
//	}
//}
