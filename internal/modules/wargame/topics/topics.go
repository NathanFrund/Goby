package topics

import "github.com/nfrund/goby/internal/topicmgr"

// Module topics for the wargame system
// These topics handle wargame events, state updates, and player actions

var (
	// TopicEventDamage is published when a unit takes damage in the wargame
	TopicEventDamage = topicmgr.DefineModule(topicmgr.TopicConfig{
		Name:        "wargame.event.damage",
		Module:      "wargame",
		Description: "Damage event in wargame when a unit takes damage",
		Pattern:     "wargame.event.damage",
		Example:     `{"targetUnit":"Tank-01","damageAmount":25,"attacker":"Artillery-03","timestamp":"2024-01-01T00:00:00Z"}`,
		Metadata: map[string]interface{}{
			"event_type": "damage",
			"payload_fields": []string{"targetUnit", "damageAmount", "attacker", "timestamp"},
		},
	})

	// TopicStateUpdate is published when the game state changes
	TopicStateUpdate = topicmgr.DefineModule(topicmgr.TopicConfig{
		Name:        "wargame.state.update",
		Module:      "wargame",
		Description: "Game state update containing current game state information",
		Pattern:     "wargame.state.update",
		Example:     `{"gameID":"game123","turn":5,"phase":"combat","units":[...],"timestamp":"2024-01-01T00:00:00Z"}`,
		Metadata: map[string]interface{}{
			"event_type": "state_change",
			"payload_fields": []string{"gameID", "turn", "phase", "units", "timestamp"},
		},
	})

	// TopicPlayerAction represents player-initiated actions
	TopicPlayerAction = topicmgr.DefineModule(topicmgr.TopicConfig{
		Name:        "wargame.action",
		Module:      "wargame",
		Description: "Player action in wargame such as move, attack, or special abilities",
		Pattern:     "wargame.action",
		Example:     `{"playerID":"player123","action":"move","unitID":"tank-01","target":{"x":10,"y":15},"timestamp":"2024-01-01T00:00:00Z"}`,
		Metadata: map[string]interface{}{
			"event_type": "player_action",
			"payload_fields": []string{"playerID", "action", "unitID", "target", "timestamp"},
			"valid_actions": []string{"move", "attack", "defend", "special"},
		},
	})

	// TopicGameStart is published when a new game begins
	TopicGameStart = topicmgr.DefineModule(topicmgr.TopicConfig{
		Name:        "wargame.game.start",
		Module:      "wargame",
		Description: "Published when a new wargame begins",
		Pattern:     "wargame.game.start",
		Example:     `{"gameID":"game123","players":["player1","player2"],"scenario":"desert_storm","timestamp":"2024-01-01T00:00:00Z"}`,
		Metadata: map[string]interface{}{
			"event_type": "lifecycle",
			"payload_fields": []string{"gameID", "players", "scenario", "timestamp"},
		},
	})

	// TopicGameEnd is published when a game ends
	TopicGameEnd = topicmgr.DefineModule(topicmgr.TopicConfig{
		Name:        "wargame.game.end",
		Module:      "wargame",
		Description: "Published when a wargame ends",
		Pattern:     "wargame.game.end",
		Example:     `{"gameID":"game123","winner":"player1","reason":"victory","duration":3600,"timestamp":"2024-01-01T00:00:00Z"}`,
		Metadata: map[string]interface{}{
			"event_type": "lifecycle",
			"payload_fields": []string{"gameID", "winner", "reason", "duration", "timestamp"},
		},
	})

	// TopicTurnChange is published when the turn changes
	TopicTurnChange = topicmgr.DefineModule(topicmgr.TopicConfig{
		Name:        "wargame.turn.change",
		Module:      "wargame",
		Description: "Published when the active turn changes to a different player",
		Pattern:     "wargame.turn.change",
		Example:     `{"gameID":"game123","previousPlayer":"player1","currentPlayer":"player2","turn":6,"timestamp":"2024-01-01T00:00:00Z"}`,
		Metadata: map[string]interface{}{
			"event_type": "turn_management",
			"payload_fields": []string{"gameID", "previousPlayer", "currentPlayer", "turn", "timestamp"},
		},
	})

	// TopicUnitDestroyed is published when a unit is destroyed
	TopicUnitDestroyed = topicmgr.DefineModule(topicmgr.TopicConfig{
		Name:        "wargame.unit.destroyed",
		Module:      "wargame",
		Description: "Published when a unit is destroyed in combat",
		Pattern:     "wargame.unit.destroyed",
		Example:     `{"unitID":"tank-01","unitType":"tank","owner":"player1","destroyedBy":"artillery-03","timestamp":"2024-01-01T00:00:00Z"}`,
		Metadata: map[string]interface{}{
			"event_type": "unit_lifecycle",
			"payload_fields": []string{"unitID", "unitType", "owner", "destroyedBy", "timestamp"},
		},
	})
)

// RegisterTopics registers all wargame module topics with the topic manager
func RegisterTopics() error {
	manager := topicmgr.Default()
	
	topics := []topicmgr.Topic{
		TopicEventDamage,
		TopicStateUpdate,
		TopicPlayerAction,
		TopicGameStart,
		TopicGameEnd,
		TopicTurnChange,
		TopicUnitDestroyed,
	}
	
	for _, topic := range topics {
		if err := manager.Register(topic); err != nil {
			return err
		}
	}
	
	return nil
}

// MustRegisterTopics registers all wargame module topics and panics on error
func MustRegisterTopics() {
	if err := RegisterTopics(); err != nil {
		panic("failed to register wargame module topics: " + err.Error())
	}
}