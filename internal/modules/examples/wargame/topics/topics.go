package topics

import (
	"github.com/nfrund/goby/internal/modules/examples/wargame/events"
	"github.com/nfrund/goby/internal/pubsub"
)

// Module topics for the wargame system
// These topics handle wargame events, state updates, and player actions

var (
	// TopicEventDamage is published when a unit takes damage in the wargame
	TopicEventDamage = pubsub.NewEvent[events.Damage]("wargame.event.damage", "Damage event in wargame when a unit takes damage")

	// TopicStateUpdate is published when the game state changes
	TopicStateUpdate = pubsub.NewEvent[events.StateUpdate]("wargame.state.update", "Game state update containing current game state information")

	// TopicPlayerAction represents player-initiated actions
	TopicPlayerAction = pubsub.NewEvent[events.PlayerAction]("wargame.action", "Player action in wargame such as move, attack, or special abilities")

	// TopicGameStart is published when a new game begins
	TopicGameStart = pubsub.NewEvent[events.GameStart]("wargame.game.start", "Published when a new wargame begins")

	// TopicGameEnd is published when a game ends
	TopicGameEnd = pubsub.NewEvent[events.GameEnd]("wargame.game.end", "Published when a wargame ends")

	// TopicTurnChange is published when the turn changes
	TopicTurnChange = pubsub.NewEvent[events.TurnChange]("wargame.turn.change", "Published when the active turn changes to a different player")

	// TopicUnitDestroyed is published when a unit is destroyed
	TopicUnitDestroyed = pubsub.NewEvent[events.UnitDestroyed]("wargame.unit.destroyed", "Published when a unit is destroyed in combat")
)

// RegisterTopics is now a no-op because NewEvent auto-registers topics.
// Kept for backward compatibility with module interface.
func RegisterTopics() error {
	return nil
}

// MustRegisterTopics is now a no-op.
func MustRegisterTopics() {
}
