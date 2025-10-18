package main

import (
	"fmt"
	"os"
	"text/tabwriter"
	
	"github.com/nfrund/goby/internal/topics"
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
	fmt.Fprintln(w, "NAME\tDESCRIPTION\tEXAMPLE")
	fmt.Fprintln(w, "----\t-----------\t-------")
	
	for _, topic := range topics.List() {
		fmt.Fprintf(w, "%s\t%s\t%s\n", 
			topic.Name, 
			topic.Description, 
			topic.Example)
	}
	w.Flush()
}

func getTopic(name string) {
	topic, exists := topics.Get(name)
	if !exists {
		fmt.Printf("Topic not found: %s\n", name)
		return
	}
	
	fmt.Printf("Name:        %s\n", topic.Name)
	fmt.Printf("Description: %s\n", topic.Description)
	fmt.Printf("Pattern:     %s\n", topic.Pattern)
	fmt.Printf("Example:     %s\n", topic.Example)
}
