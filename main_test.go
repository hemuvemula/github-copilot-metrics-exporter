package main

import (
	"testing"
	"time"
)

func TestNewCopilotCollector(t *testing.T) {
	scrapeInterval := 60 * time.Second
	collector := NewCopilotCollector("test-token", "test-org", "", "", scrapeInterval)

	if collector == nil {
		t.Fatal("Expected collector to be created")
	}

	if collector.githubToken != "test-token" {
		t.Errorf("Expected githubToken to be 'test-token', got '%s'", collector.githubToken)
	}

	if collector.organization != "test-org" {
		t.Errorf("Expected organization to be 'test-org', got '%s'", collector.organization)
	}

	if collector.scrapeInterval != scrapeInterval {
		t.Errorf("Expected scrapeInterval to be %v, got %v", scrapeInterval, collector.scrapeInterval)
	}
}

func TestNewCopilotCollectorWithTeam(t *testing.T) {
	scrapeInterval := 60 * time.Second
	collector := NewCopilotCollector("test-token", "test-org", "test-team", "", scrapeInterval)

	if collector.team != "test-team" {
		t.Errorf("Expected team to be 'test-team', got '%s'", collector.team)
	}
}

func TestNewCopilotCollectorWithEnterprise(t *testing.T) {
	scrapeInterval := 60 * time.Second
	collector := NewCopilotCollector("test-token", "", "", "test-enterprise", scrapeInterval)

	if collector.enterprise != "test-enterprise" {
		t.Errorf("Expected enterprise to be 'test-enterprise', got '%s'", collector.enterprise)
	}
}
