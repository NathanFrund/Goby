package components

import (
	"fmt"

	"maragu.dev/gomponents"
	hx "maragu.dev/gomponents-htmx"
	"maragu.dev/gomponents/html"
)

func DamageEvent(attackingUnit string, damageAmount int, targetUnit string) gomponents.Node {
	// This is the message content that will be appended to #chat-messages
	messageContent := html.Div(
		html.Class("p-2 text-red-500 font-mono border-b border-red-900"),
		gomponents.Raw(fmt.Sprintf(
			"<strong>HIT</strong> &mdash; %s deals %d damage to %s!",
			attackingUnit,
			damageAmount,
			targetUnit,
		)),
	)

	// Wrap it in a div that tells HTMX to append this to #chat-messages
	return html.Div(
		hx.SwapOOB("beforeend:#chat-messages"),
		messageContent,
	)
}
