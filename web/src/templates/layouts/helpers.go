package layouts

// CalculateTitle handles the conditional logic for the page title.
// It is exported (capital C) so it can be called from the templ component.
func CalculateTitle(title string) string {
	if title != "" {
		return title + " - Goby"
	}
	return "Goby"
}
