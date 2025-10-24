package scripts

import _ "embed"

// Embedded script files for the wargame module
// These scripts provide customizable behavior for damage calculation,
// event processing, and hit simulation

//go:embed damage_calculator.tengo
var DamageCalculatorScript string

//go:embed event_processor.tengo
var EventProcessorScript string

//go:embed hit_simulator.tengo
var HitSimulatorScript string

// GetEmbeddedScripts returns all embedded scripts for the wargame module
func GetEmbeddedScripts() map[string]string {
	return map[string]string{
		"damage_calculator": DamageCalculatorScript,
		"event_processor":   EventProcessorScript,
		"hit_simulator":     HitSimulatorScript,
	}
}

// GetModuleName returns the module name for these scripts
func GetModuleName() string {
	return "wargame"
}