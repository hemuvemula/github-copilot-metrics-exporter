package main

import (
	"testing"
)

func TestNewCopilotCollector(t *testing.T) {
	collector := NewCopilotCollector("test-token", "test-org", "", "")

	if collector == nil {
		t.Fatal("Expected collector to be created")
	}

	if collector.githubToken != "test-token" {
		t.Errorf("Expected githubToken to be 'test-token', got '%s'", collector.githubToken)
	}

	if collector.organization != "test-org" {
		t.Errorf("Expected organization to be 'test-org', got '%s'", collector.organization)
	}
}

func TestNewCopilotCollectorWithTeam(t *testing.T) {
	collector := NewCopilotCollector("test-token", "test-org", "test-team", "")

	if collector.team != "test-team" {
		t.Errorf("Expected team to be 'test-team', got '%s'", collector.team)
	}
}

func TestNewCopilotCollectorWithEnterprise(t *testing.T) {
	collector := NewCopilotCollector("test-token", "", "", "test-enterprise")

	if collector.enterprise != "test-enterprise" {
		t.Errorf("Expected enterprise to be 'test-enterprise', got '%s'", collector.enterprise)
	}
}
