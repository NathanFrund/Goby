package wargame

import "github.com/nfrund/goby/internal/topics"

var (
	// EventDamage is published when a unit takes damage
	EventDamage = topics.NewBaseTopic(
		"wargame.event.damage",
		"Damage event in wargame",
		"wargame.event.damage",
		"wargame.event.damage",
	)

	// StateUpdate is published when game state changes
	StateUpdate = topics.NewBaseTopic(
		"wargame.state.update",
		"Game state update",
		"wargame.state.update",
		"wargame.state.update",
	)

	// PlayerAction is for player-initiated actions
	PlayerAction = topics.NewBaseTopic(
		"wargame.action",
		"Player action in wargame",
		"wargame.action",
		"wargame.action",
	)
)

// RegisterTopics registers all wargame topics with the provided registry
func RegisterTopics(reg *topics.TopicRegistry) error {
	topicsToRegister := []topics.Topic{
		EventDamage,
		StateUpdate,
		PlayerAction,
	}

	for _, topic := range topicsToRegister {
		if err := reg.Register(topic); err != nil {
			return err
		}
	}
	return nil
}
