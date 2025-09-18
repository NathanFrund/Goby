package templates

// TemplateRegistrar defines the signature for a function that can register templates.
type TemplateRegistrar func(r *Renderer)

var registrars []TemplateRegistrar

// Register adds a template registration function to the central registry.
// This is intended to be called from module init() functions.
func Register(registrar TemplateRegistrar) {
	registrars = append(registrars, registrar)
}

// ApplyRegistrars calls all registered template registration functions.
func ApplyRegistrars(r *Renderer) {
	for _, registrar := range registrars {
		registrar(r)
	}
}
