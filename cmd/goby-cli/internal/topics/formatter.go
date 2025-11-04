package topics

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/nfrund/goby/internal/topicmgr"
)

// TopicDisplay represents a topic for display purposes
type TopicDisplay struct {
	Name        string                 `json:"name"`
	Scope       string                 `json:"scope"`
	Module      string                 `json:"module"`
	Description string                 `json:"description"`
	Pattern     string                 `json:"pattern"`
	Example     string                 `json:"example"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// FormatConfig holds configuration for output formatting
type FormatConfig struct {
	OutputFormat string // "table" or "json"
}

// DisplayTopicsTable displays topics in a formatted table
func DisplayTopicsTable(topics []topicmgr.Topic) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	defer w.Flush()

	fmt.Fprintln(w, "NAME\tSCOPE\tMODULE\tDESCRIPTION\tEXAMPLE")
	fmt.Fprintln(w, "----\t-----\t------\t-----------\t-------")

	if len(topics) == 0 {
		fmt.Fprintln(w, "No topics found")
	} else {
		for _, topic := range topics {
			module := topic.Module()
			if module == "" {
				module = "-"
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
				topic.Name(),
				topic.Scope(),
				module,
				truncateString(topic.Description(), 40),
				truncateString(topic.Example(), 30))
		}
	}
}

// DisplayTopicsJSON displays topics in JSON format
func DisplayTopicsJSON(topics []topicmgr.Topic) error {
	topicDisplays := make([]TopicDisplay, len(topics))
	for i, topic := range topics {
		topicDisplays[i] = TopicDisplay{
			Name:        topic.Name(),
			Scope:       string(topic.Scope()),
			Module:      topic.Module(),
			Description: topic.Description(),
			Pattern:     topic.Pattern(),
			Example:     topic.Example(),
			Metadata:    topic.Metadata(),
		}
	}

	output := struct {
		Topics []TopicDisplay `json:"topics"`
		Count  int            `json:"count"`
	}{
		Topics: topicDisplays,
		Count:  len(topicDisplays),
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(output)
}

// DisplayTopicDetails displays detailed information for a specific topic
func DisplayTopicDetails(topic topicmgr.Topic, format string) error {
	if format == "json" {
		topicDisplay := TopicDisplay{
			Name:        topic.Name(),
			Scope:       string(topic.Scope()),
			Module:      topic.Module(),
			Description: topic.Description(),
			Pattern:     topic.Pattern(),
			Example:     topic.Example(),
			Metadata:    topic.Metadata(),
		}

		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		return encoder.Encode(topicDisplay)
	}

	// Table format for detailed view
	fmt.Printf("Name:        %s\n", topic.Name())
	fmt.Printf("Scope:       %s\n", topic.Scope())
	fmt.Printf("Module:      %s\n", topic.Module())
	fmt.Printf("Description: %s\n", topic.Description())
	fmt.Printf("Pattern:     %s\n", topic.Pattern())
	fmt.Printf("Example:     %s\n", topic.Example())

	// Show metadata if available
	metadata := topic.Metadata()
	if len(metadata) > 0 {
		fmt.Printf("Metadata:\n")
		for k, v := range metadata {
			fmt.Printf("  %s: %v\n", k, v)
		}
	}

	return nil
}

// DisplayFilteredTopicsTable displays filtered topics with a header message
func DisplayFilteredTopicsTable(topics []topicmgr.Topic, filterType, filterValue string) {
	fmt.Printf("Topics for %s '%s':\n\n", filterType, filterValue)
	DisplayTopicsTable(topics)
}

// DisplayValidationResult displays topic validation results with appropriate formatting
func DisplayValidationResult(topic topicmgr.Topic, nameErr, defErr error) {
	if nameErr != nil {
		fmt.Printf("❌ Topic name validation failed: %v\n", nameErr)
		return
	}

	if defErr != nil {
		fmt.Printf("❌ Topic validation failed: %v\n", defErr)
		return
	}

	fmt.Printf("✅ Topic '%s' is valid\n", topic.Name())
	fmt.Printf("   Scope: %s\n", topic.Scope())
	fmt.Printf("   Module: %s\n", topic.Module())
}

// truncateString truncates a string to maxLen characters, adding "..." if truncated
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return "..."
	}
	return s[:maxLen-3] + "..."
}
