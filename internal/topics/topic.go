package topics

import "strings"

// Topic represents a message topic with metadata
type Topic struct {
	Name        string
	Description string
	Pattern     string
	Example     string
}

// Format formats the topic with the given parameters
func (t Topic) Format(params map[string]string) string {
	result := t.Pattern
	for k, v := range params {
		result = strings.ReplaceAll(result, "{"+k+"}", v)
	}
	return result
}

// String returns the topic's name
func (t Topic) String() string {
	return t.Name
}
