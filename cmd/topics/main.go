package main

import (
	"fmt"
	"io"
	"log"
	"log/slog"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/joho/godotenv"
	"github.com/nfrund/goby/internal/app"
	"github.com/nfrund/goby/internal/config"
	"github.com/nfrund/goby/internal/registry"
	"github.com/nfrund/goby/internal/topicmgr"
)

func main() {
	// Suppress all logging output to make CLI less chatty
	log.SetOutput(io.Discard)
	slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, nil)))

	// Also suppress watermill logging
	os.Setenv("WATERMILL_LOG_LEVEL", "ERROR")

	// Initialize topics by setting up minimal dependencies
	if err := initializeTopics(); err != nil {
		fmt.Printf("Error initializing topics: %v\n", err)
		os.Exit(1)
	}

	if len(os.Args) > 1 && os.Args[1] == "list" {
		if len(os.Args) > 2 {
			// Support filtering: topics list --module=chat or topics list --scope=framework
			filter := os.Args[2]
			listTopicsWithFilter(filter)
		} else {
			listTopics()
		}
		return
	}

	if len(os.Args) > 2 && os.Args[1] == "get" {
		getTopic(os.Args[2])
		return
	}

	if len(os.Args) > 1 && os.Args[1] == "validate" {
		if len(os.Args) > 2 {
			validateTopic(os.Args[2])
		} else {
			fmt.Println("Usage: topics validate <topic-name>")
		}
		return
	}

	printUsage()
}

// initializeTopics sets up minimal dependencies to register all topics
func initializeTopics() error {
	// Load .env file if it exists
	if err := godotenv.Load(); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("error loading .env file: %w", err)
	}

	// Create minimal configuration
	cfg := config.New()

	// Create a minimal registry
	reg := registry.New(cfg)

	// Create minimal module dependencies
	moduleDeps := app.Dependencies{
		Publisher:       nil,
		Subscriber:      nil,
		Renderer:        nil,
		TopicMgr:        topicmgr.Default(),
		PresenceService: nil,
		// Other fields will be zero values
	}

	// Initialize modules to register their topics
	modules := app.NewModules(moduleDeps)

	// Register module topics by calling their Register methods
	for _, mod := range modules {
		if err := mod.Register(reg); err != nil {
			// Ignore registry-related errors since we only care about topic registration
			if !strings.Contains(err.Error(), "nil pointer") {
				return fmt.Errorf("failed to register module %s: %w", mod.Name(), err)
			}
		}
	}

	// All modules should now register their topics in their Register() method
	// This provides a clean, consistent way for the CLI to discover all topics
	// without needing to know about specific modules or their internal structure

	return nil
}


func printUsage() {
	fmt.Println("Topic Registry CLI")
	fmt.Println("Usage:")
	fmt.Println("  topics list                    - List all registered topics")
	fmt.Println("  topics list --module=<name>    - List topics for specific module")
	fmt.Println("  topics list --scope=<scope>    - List topics for specific scope (framework/module)")
	fmt.Println("  topics get <name>              - Get details about a specific topic")
	fmt.Println("  topics validate <name>         - Validate a topic name")
}

func listTopics() {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tSCOPE\tMODULE\tDESCRIPTION\tEXAMPLE")
	fmt.Fprintln(w, "----\t-----\t------\t-----------\t-------")

	manager := topicmgr.Default()
	topics := manager.List()

	if len(topics) == 0 {
		fmt.Fprintln(w, "No topics registered")
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
	w.Flush()
}

func listTopicsWithFilter(filter string) {
	manager := topicmgr.Default()
	var topics []topicmgr.Topic

	if moduleName, found := strings.CutPrefix(filter, "--module="); found {
		topics = manager.ListByModule(moduleName)
		fmt.Printf("Topics for module '%s':\n\n", moduleName)
	} else if scopeName, found := strings.CutPrefix(filter, "--scope="); found {
		var scope topicmgr.TopicScope
		switch scopeName {
		case "framework":
			scope = topicmgr.ScopeFramework
		case "module":
			scope = topicmgr.ScopeModule
		default:
			fmt.Printf("Invalid scope '%s'. Valid scopes: framework, module\n", scopeName)
			return
		}
		topics = manager.ListByScope(scope)
		fmt.Printf("Topics for scope '%s':\n\n", scopeName)
	} else {
		fmt.Printf("Invalid filter '%s'. Use --module=<name> or --scope=<scope>\n", filter)
		return
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
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
	w.Flush()
}

func getTopic(name string) {
	manager := topicmgr.Default()
	topic, exists := manager.Get(name)
	if !exists {
		fmt.Printf("Topic not found: %s\n", name)
		return
	}

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
}

func validateTopic(name string) {
	manager := topicmgr.Default()

	// Check if topic exists
	topic, exists := manager.Get(name)
	if !exists {
		fmt.Printf("❌ Topic '%s' not found\n", name)
		return
	}

	// Validate the topic name format
	if err := manager.ValidateTopicName(name); err != nil {
		fmt.Printf("❌ Topic name validation failed: %v\n", err)
		return
	}

	// Validate the topic definition
	if err := manager.Validate(topic, "cli-validation"); err != nil {
		fmt.Printf("❌ Topic validation failed: %v\n", err)
		return
	}

	fmt.Printf("✅ Topic '%s' is valid\n", name)
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
