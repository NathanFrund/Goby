package topics

import "strings"

// testTopic is a minimal implementation of the Topic interface for testing
type testTopic struct {
	name        string
	description string
	pattern     string
	example     string
}

func (t testTopic) Name() string        { return t.name }
func (t testTopic) Description() string { return t.description }
func (t testTopic) Pattern() string     { return t.pattern }
func (t testTopic) Example() string     { return t.example }
func (t testTopic) Format(vars interface{}) (string, error) {
	params, ok := vars.(map[string]string)
	if !ok {
		return "", nil
	}
	result := t.pattern
	for k, v := range params {
		result = strings.ReplaceAll(result, "{"+k+"}", v)
	}
	return result, nil
}
func (t testTopic) Validate(vars interface{}) error { return nil }

// NewTestTopic creates a new test topic with the given parameters
func NewTestTopic(name, description, pattern, example string) Topic {
	return testTopic{
		name:        name,
		description: description,
		pattern:     pattern,
		example:     example,
	}
}
