package wargame

import "time"

// BaseMessage contains common fields for all wargame messages
type BaseMessage struct {
	MessageID string    `json:"message_id"`
	Timestamp time.Time `json:"timestamp"`
	Version   string    `json:"version"`
}

// DamageEvent represents a damage event in the game
type DamageEvent struct {
	BaseMessage
	TargetUnit   string `json:"target_unit"`
	DamageAmount int    `json:"damage_amount"`
	Attacker     string `json:"attacker"`
	WeaponType   string `json:"weapon_type,omitempty"`
}

// GameState represents the current game state
type GameState struct {
	BaseMessage
	Units       []UnitState `json:"units"`
	CurrentTurn string      `json:"current_turn"`
	GamePhase   string      `json:"game_phase"`
	LastUpdated time.Time   `json:"last_updated"`
}

// UnitState represents the state of a game unit
type UnitState struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Health    int    `json:"health"`
	MaxHealth int    `json:"max_health"`
	Position  string `json:"position"`
	Status    string `json:"status"` // e.g., "active", "disabled", "destroyed"
}

// Action represents a player action
type Action struct {
	BaseMessage
	PlayerID  string      `json:"player_id"`
	Action    string      `json:"action"` // e.g., "move", "attack", "defend"
	TargetID  string      `json:"target_id,omitempty"`
	Data      interface{} `json:"data,omitempty"` // Action-specific data
	Timestamp time.Time   `json:"timestamp"`
}
