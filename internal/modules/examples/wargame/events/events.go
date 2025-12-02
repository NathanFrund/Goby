package events

// Damage represents a unit taking damage.
type Damage struct {
	TargetUnit   string `json:"targetUnit"`
	DamageAmount int    `json:"damageAmount"`
	Attacker     string `json:"attacker"`
	Timestamp    string `json:"timestamp"`
}

// StateUpdate represents a full game state update.
type StateUpdate struct {
	GameID    string      `json:"gameID"`
	Turn      int         `json:"turn"`
	Phase     string      `json:"phase"`
	Units     interface{} `json:"units"` // Using interface{} for now as unit structure is complex
	Timestamp string      `json:"timestamp"`
}

// PlayerAction represents an action taken by a player.
type PlayerAction struct {
	PlayerID  string      `json:"playerID"`
	Action    string      `json:"action"`
	UnitID    string      `json:"unitID"`
	Target    interface{} `json:"target"`
	Timestamp string      `json:"timestamp"`
}

// GameStart represents the start of a new game.
type GameStart struct {
	GameID    string   `json:"gameID"`
	Players   []string `json:"players"`
	Scenario  string   `json:"scenario"`
	Timestamp string   `json:"timestamp"`
}

// GameEnd represents the end of a game.
type GameEnd struct {
	GameID    string `json:"gameID"`
	Winner    string `json:"winner"`
	Reason    string `json:"reason"`
	Duration  int    `json:"duration"`
	Timestamp string `json:"timestamp"`
}

// TurnChange represents a turn change event.
type TurnChange struct {
	GameID         string `json:"gameID"`
	PreviousPlayer string `json:"previousPlayer"`
	CurrentPlayer  string `json:"currentPlayer"`
	Turn           int    `json:"turn"`
	Timestamp      string `json:"timestamp"`
}

// UnitDestroyed represents a unit being destroyed.
type UnitDestroyed struct {
	UnitID      string `json:"unitID"`
	UnitType    string `json:"unitType"`
	Owner       string `json:"owner"`
	DestroyedBy string `json:"destroyedBy"`
	Timestamp   string `json:"timestamp"`
}
