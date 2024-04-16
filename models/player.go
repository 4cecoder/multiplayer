// Package models player.go
package models

import (
	"github.com/gorilla/websocket"
)

type Player struct {
	ID               string          `json:"id"`
	StartingPosition Point           `json:"startingPosition"`
	Name             string          `json:"name"`
	Color            string          `json:"color"`
	X                float64         `json:"x"`
	Y                float64         `json:"y"`
	VelocityX        float64         `json:"velocityX"`
	VelocityY        float64         `json:"velocityY"`
	Acceleration     float64         `json:"-"`
	MaxVelocity      float64         `json:"-"`
	Conn             *websocket.Conn `json:"-"`
	LandCapture      [][]bool        `json:"landCapture"`
	PlayerTrail      []Point         `json:"playerTrail"`
	StartingLand     [][]bool        `json:"startingLand"`
	IsAlive          bool            `json:"isAlive"`
	KillStreak       int             `json:"-"`
	SpeedMultiplier  float64         `json:"-"`
	WriteChan        chan RenderInstruction
}

type Point struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

type RenderInstruction struct {
	Type    string      `json:"type"`
	Payload PlayerState `json:"payload"`
}

type PlayerState struct {
	ID               string   `json:"id"`
	StartingPosition Point    `json:"startingPosition"`
	Name             string   `json:"name"`
	Color            string   `json:"color"`
	X                float64  `json:"x"`
	Y                float64  `json:"y"`
	VelocityX        float64  `json:"velocityX"`
	VelocityY        float64  `json:"velocityY"`
	LandCapture      [][]bool `json:"landCapture"`
	PlayerTrail      []Point  `json:"playerTrail"`
	StartingLand     [][]bool `json:"startingLand"`
	IsAlive          bool     `json:"isAlive"`
	Direction        interface{}
}
