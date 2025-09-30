package pages

import (
	// FIX: Replaced dot import with 'cmp' prefix to resolve ST1001 warning.
	cmp "maragu.dev/gomponents"
	// Imports helper components (e.g., Table, Ul) with prefix c.

	// Imports all HTML elements (Div, H1, Class, etc.) with prefix g.
	g "maragu.dev/gomponents/html"
)

// AboutContent is a Gomponents Node representing the main content of the About page.
// FIX: Changed return type from 'Node' to 'cmp.Node'.
func AboutContent() cmp.Node {
	return g.Div(
		g.Class("container mx-auto p-8"),
		g.Div(
			g.Class("bg-white shadow-2xl rounded-xl p-10"),
			g.H1(
				g.Class("text-4xl font-extrabold text-indigo-700 mb-4 border-b pb-2"),
				// FIX: Added 'cmp.' prefix to Text() calls.
				cmp.Text("About Goby: Modular Go Architecture"),
			),
			g.P(
				g.Class("text-gray-700 mb-6 leading-relaxed"),
				// FIX: Added 'cmp.' prefix to Text() calls.
				cmp.Text("Goby is designed to showcase a modern, modular, and hybrid approach to web development in Go. It allows different modules to choose the best rendering tool—Gomponents for speed and structure, or Templ for complex, type-safe components."),
			),
			g.Div(
				g.Class("space-y-4"),
				g.Div(
					g.Class("p-6 bg-gray-50 rounded-lg shadow"),
					g.Div(g.Class("font-bold text-xl mb-2"), cmp.Text("Friction-Free Development")),
					g.P(g.Class("text-gray-700 text-base"), cmp.Text("This page is built entirely with Gomponents, demonstrating how easily structural pages can be added without running any code generation tool.")),
				),
				g.Div(
					g.Class("p-6 bg-gray-50 rounded-lg shadow"),
					g.Div(g.Class("font-bold text-xl mb-2"), cmp.Text("Future-Proof Hybrid")),
					g.P(g.Class("text-gray-700 text-base"), cmp.Text("We use the adapter pattern to wrap any Templ component into this Gomponents-first layout, ensuring total flexibility for future modules.")),
				),
			),
			g.Div(
				g.Class("mt-8 pt-4 border-t text-sm text-gray-500"),
				// FIX: Added 'cmp.' prefix to Text() calls.
				cmp.Text("Thank you for exploring Goby!"),
			),
		),
	)
}
