package analyzer

import (
	"testing"
)

func TestAnalyzer_Basic(t *testing.T) {
	analyzer := New()
	if analyzer == nil {
		t.Fatal("Expected analyzer to be created")
	}

	if analyzer.aggregator == nil {
		t.Fatal("Expected aggregator to be initialized")
	}
}

func TestServiceMetadata_Structure(t *testing.T) {
	metadata := ServiceMetadata{
		Key:         "test.service",
		Type:        "TestService",
		Module:      "test",
		Description: "Test service",
		Category:    "test",
	}

	if metadata.Key != "test.service" {
		t.Errorf("Expected key to be 'test.service', got %s", metadata.Key)
	}
}
