package components

import (
	"fmt"

	"maragu.dev/gomponents"
	hx "maragu.dev/gomponents-htmx"
	. "maragu.dev/gomponents/html"
)

// DamageEvent creates a damage event message component for the chat.
func DamageEvent(targetUnit string, damageAmount int, attackingUnit string) gomponents.Node {
	// Outer div with HTMX OOB swap
	return Div(
		hx.SwapOOB("beforeend:#chat-messages"),
		// Inner div with styling
		Div(
			Class("p-2 text-red-500 font-mono border-b border-red-900"),
			gomponents.Text(fmt.Sprintf(
				"**HIT** %s deals %d damage to %s!",
				attackingUnit,
				damageAmount,
				targetUnit,
			)),
		),
	)
}
