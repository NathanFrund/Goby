package main

import (
	"fmt"
	"os"
	"text/tabwriter"
	
	"github.com/nfrund/goby/internal/topicmgr"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "list" {
		listTopics()
		return
	}
	
	if len(os.Args) > 2 && os.Args[1] == "get" {
		getTopic(os.Args[2])
		return
	}
	
	printUsage()
}

func printUsage() {
	fmt.Println("Topic Registry CLI")
	fmt.Println("Usage:")
	fmt.Println("  topics list          - List all registered topics")
	fmt.Println("  topics get <name>    - Get details about a specific topic")
}

func listTopics() {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tSCOPE\tMODULE\tDESCRIPTION\tEXAMPLE")
	fmt.Fprintln(w, "----\t-----\t------\t-----------\t-------")
	
	manager := topicmgr.Default()
	for _, topic := range manager.List() {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", 
			topic.Name(), 
			topic.Scope(),
			topic.Module(),
			topic.Description(), 
			topic.Example())
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
