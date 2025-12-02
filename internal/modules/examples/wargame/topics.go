package wargame

import "github.com/nfrund/goby/internal/modules/examples/wargame/topics"

// Topic references for backward compatibility during migration
var (
	// EventDamage is published when a unit takes damage
	EventDamage = topics.TopicEventDamage

	// StateUpdate is published when game state changes
	StateUpdate = topics.TopicStateUpdate

	// PlayerAction is for player-initiated actions
	PlayerAction = topics.TopicPlayerAction
)
