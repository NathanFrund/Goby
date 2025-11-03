package cmd

import (
	"fmt"
	"os"

	"github.com/nfrund/goby/cmd/goby-cli/internal/topics"
	"github.com/nfrund/goby/internal/topicmgr"
	"github.com/spf13/cobra"
)

// topicsValidateCmd represents the topics validate command
var topicsValidateCmd = &cobra.Command{
	Use:   "validate <topic-name>",
	Short: "Validate a topic definition",
	Long: `Validate a topic definition to ensure it follows proper naming conventions
and has complete configuration. This command checks both the topic name format
and the topic definition completeness.

The validation process includes:
- Topic name format validation (lowercase, alphanumeric, dots only)
- Topic definition completeness (description, pattern, example)
- Scope-specific validation rules (framework vs module topics)
- Reserved prefix checking

Examples:
  # Basic validation
  goby-cli topics validate user.created          # Validate user.created topic
  goby-cli topics validate chat.message.sent     # Validate chat message topic
  
  # Error cases
  goby-cli topics validate Invalid.Topic         # Shows name format error
  goby-cli topics validate nonexistent.topic     # Shows "topic not found" error

Output:
  ✅ Success - Shows topic is valid with details
  ❌ Error   - Shows specific validation failure with explanation`,
	Args: cobra.ExactArgs(1),
	Run:  topicsValidateHandler,
}

func topicsValidateHandler(cmd *cobra.Command, args []string) {
	topicName := args[0]

	// Initialize topics system
	if err := topics.Initialize(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to initialize topics: %v\n", err)
		os.Exit(1)
	}

	manager := topicmgr.Default()

	// First validate the topic name format
	nameErr := manager.ValidateTopicName(topicName)

	// Look up the topic to validate its definition
	topic, found := manager.Get(topicName)
	var defErr error
	if found {
		// Validate the topic definition
		defErr = manager.Validate(topic, "cli-validation")
	} else {
		// Topic not found - this is an error for validation
		defErr = fmt.Errorf("topic '%s' not found", topicName)
	}

	// Display validation results with appropriate formatting
	if nameErr != nil {
		fmt.Printf("❌ Topic name validation failed: %v\n", nameErr)
		fmt.Fprintf(os.Stderr, "\nTopic names must follow the pattern: scope.module.action\n")
		fmt.Fprintf(os.Stderr, "Examples: user.created, chat.message.sent, presence.user.online\n")
		os.Exit(1)
	}

	if defErr != nil {
		fmt.Printf("❌ Topic validation failed: %v\n", defErr)
		if !found {
			fmt.Fprintf(os.Stderr, "\nUse 'goby-cli topics list' to see all available topics.\n")
		}
		os.Exit(1)
	}

	// Success case - display topic details
	fmt.Printf("✅ Topic '%s' is valid\n", topic.Name())
	fmt.Printf("   Scope: %s\n", topic.Scope())
	if topic.Module() != "" {
		fmt.Printf("   Module: %s\n", topic.Module())
	} else {
		fmt.Printf("   Module: (framework)\n")
	}
	fmt.Printf("   Description: %s\n", topic.Description())
	fmt.Printf("   Pattern: %s\n", topic.Pattern())
	if topic.Example() != "" {
		fmt.Printf("   Example: %s\n", topic.Example())
	}
}

func init() {
	topicsCmd.AddCommand(topicsValidateCmd)
}
