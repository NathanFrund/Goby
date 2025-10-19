package components

import (
	"maragu.dev/gomponents"
	hx "maragu.dev/gomponents-htmx"
	"maragu.dev/gomponents/html"
)

// DamageEvent creates a damage event component using Gomponents
// In damage_event.go
func DamageEvent(targetUnit string, damageAmount int, attacker string, messageID string) gomponents.Node {
	return html.Div(
		html.Class("p-4 mb-4 bg-red-900/10 rounded-lg border-l-4 border-red-500"),
		html.ID(messageID),
		hx.SwapOOB("beforeend:#chat-messages"), // Moved here from subscriber
		html.Div(html.Class("flex items-center justify-between"),
			html.Div(
				html.Span(html.Class("font-bold text-red-400"),
					gomponents.Text("💥 BATTLE: "+targetUnit+" hit! "),
				),
				html.Span(html.Class("text-red-300 ml-2"),
					gomponents.Textf("-%d HP", damageAmount),
				),
			),
			html.Span(html.Class("text-yellow-300 text-sm"),
				gomponents.Text("By "+attacker),
			),
		),
	)
}
