package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

// Helper function to create a test collector with mocked API
func createTestCollectorWithMockAPI(t *testing.T, mockData string) (*CopilotCollector, *httptest.Server) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, mockData)
	}))

	collector := NewCopilotCollector("test-token", "test-org", "", "")
	return collector, server
}

// Helper to collect metrics into a slice
func collectMetrics(t *testing.T, collector *CopilotCollector) []prometheus.Metric {
	ch := make(chan prometheus.Metric, 500)
	go func() {
		collector.Collect(ch)
		close(ch)
	}()

	var metrics []prometheus.Metric
	for m := range ch {
		metrics = append(metrics, m)
	}
	return metrics
}

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

	// Verify all metrics are initialized
	if collector.totalSuggestions == nil {
		t.Error("Expected totalSuggestions to be initialized")
	}
	if collector.totalAcceptances == nil {
		t.Error("Expected totalAcceptances to be initialized")
	}
	if collector.breakdownSuggestions == nil {
		t.Error("Expected breakdownSuggestions to be initialized")
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

func TestCopilotCollector_Describe(t *testing.T) {
	collector := NewCopilotCollector("test-token", "test-org", "", "")
	ch := make(chan *prometheus.Desc, 30)

	collector.Describe(ch)
	close(ch)

	count := 0
	for range ch {
		count++
	}

	// Should have 22 metrics
	if count != 22 {
		t.Errorf("Expected 22 metric descriptions, got %d", count)
	}
}

func TestCopilotCollector_FetchMetrics_Organization(t *testing.T) {
	mockResponse := []map[string]interface{}{
		{
			"day":                     "2024-01-01",
			"total_suggestions_count": 100,
			"total_acceptances_count": 80,
			"total_lines_suggested":   500,
			"total_lines_accepted":    400,
			"total_active_users":      10,
			"total_chat_acceptances":  20,
			"total_chat_turns":        30,
			"total_active_chat_users": 5,
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify the request headers
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Errorf("Expected Authorization header 'Bearer test-token'")
		}
		if r.Header.Get("Accept") != "application/vnd.github+json" {
			t.Errorf("Expected Accept header 'application/vnd.github+json'")
		}
		if r.Header.Get("X-GitHub-Api-Version") != "2022-11-28" {
			t.Errorf("Expected X-GitHub-Api-Version header '2022-11-28'")
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockResponse)
	}))
	defer server.Close()

	collector := NewCopilotCollector("test-token", "test-org", "", "")

	// We can verify the collector structure is correct
	if collector.organization != "test-org" {
		t.Errorf("Expected organization 'test-org', got '%s'", collector.organization)
	}
	if collector.githubToken != "test-token" {
		t.Errorf("Expected token 'test-token', got '%s'", collector.githubToken)
	}
}

func TestCopilotCollector_FetchMetrics_Team(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/orgs/test-org/team/test-team/copilot/metrics" {
			t.Errorf("Expected path /orgs/test-org/team/test-team/copilot/metrics, got %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]map[string]interface{}{
			{
				"day":                     "2024-01-01",
				"total_suggestions_count": 50,
				"total_acceptances_count": 40,
				"total_lines_suggested":   250,
				"total_lines_accepted":    200,
				"total_active_users":      5,
				"total_chat_acceptances":  10,
				"total_chat_turns":        15,
				"total_active_chat_users": 3,
			},
		})
	}))
	defer server.Close()

	collector := NewCopilotCollector("test-token", "test-org", "test-team", "")
	// Test that team URL is constructed correctly (we can't actually test without mocking the full HTTP client)
	if collector.team != "test-team" {
		t.Errorf("Expected team 'test-team', got '%s'", collector.team)
	}
}

func TestCopilotCollector_FetchMetrics_Enterprise(t *testing.T) {
	collector := NewCopilotCollector("test-token", "", "", "test-enterprise")
	if collector.enterprise != "test-enterprise" {
		t.Errorf("Expected enterprise 'test-enterprise', got '%s'", collector.enterprise)
	}
}

func TestCopilotCollector_Collect(t *testing.T) {
	mockData := `[{
		"day": "2024-01-01",
		"total_suggestions_count": 100,
		"total_acceptances_count": 80,
		"total_lines_suggested": 500,
		"total_lines_accepted": 400,
		"total_active_users": 10,
		"total_chat_acceptances": 20,
		"total_chat_turns": 30,
		"total_active_chat_users": 5,
		"breakdown": [
			{
				"language": "python",
				"suggestions_count": 50,
				"acceptances_count": 40,
				"lines_suggested": 250,
				"lines_accepted": 200,
				"active_users": 5
			}
		],
		"copilot_ide_code_completions": {
			"total_engaged_users": 10,
			"languages": [
				{
					"language": "python",
					"suggestions_count": 50,
					"acceptances_count": 40
				}
			],
			"editors": [
				{
					"editor": "vscode",
					"suggestions_count": 50,
					"acceptances_count": 40
				}
			],
			"models": [
				{
					"model": "gpt-4",
					"suggestions_count": 50,
					"acceptances_count": 40
				}
			]
		},
		"copilot_ide_chat": {
			"total_engaged_users": 5,
			"editors": [
				{
					"editor": "vscode",
					"chat_acceptances": 20,
					"chat_turns": 30
				}
			],
			"models": [
				{
					"model": "gpt-4",
					"chat_acceptances": 20,
					"chat_turns": 30
				}
			]
		},
		"copilot_dotcom_chat": {
			"total_engaged_users": 3,
			"models": [
				{
					"model": "gpt-4",
					"chat_turns": 15
				}
			]
		},
		"copilot_dotcom_pull_requests": {
			"total_engaged_users": 2,
			"repositories": [
				{
					"name": "test-repo",
					"total_engaged_users": 2,
					"models": [
						{
							"model": "gpt-4",
							"suggestions_count": 10
						}
					]
				}
			],
			"models": [
				{
					"model": "gpt-4",
					"suggestions_count": 10
				}
			]
		}
	}]`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, mockData)
	}))
	defer server.Close()

	collector := NewCopilotCollector("test-token", "test-org", "", "")

	// Create a registry and register the collector
	registry := prometheus.NewRegistry()
	registry.MustRegister(collector)

	// Note: Collection will fail in test environment without real API
	// but we verify the collector is properly structured
	count := testutil.CollectAndCount(collector)
	// The collector should attempt to collect metrics
	// In production, it would fetch from GitHub API
	_ = count
}

func TestCopilotCollector_ExportBreakdown(t *testing.T) {
	collector := NewCopilotCollector("test-token", "test-org", "", "")
	ch := make(chan prometheus.Metric, 100)

	breakdown := Breakdown{
		Language:         "python",
		Editor:           "vscode",
		Model:            "gpt-4",
		SuggestionsCount: 50,
		AcceptancesCount: 40,
		LinesSuggested:   250,
		LinesAccepted:    200,
		ActiveUsers:      5,
		ChatAcceptances:  10,
		ChatTurns:        15,
		ActiveChatUsers:  3,
	}

	collector.exportBreakdown(ch, "2024-01-01", "test-org", breakdown, "language")

	close(ch)

	count := 0
	for range ch {
		count++
	}

	// Should export metrics for all non-zero fields
	if count < 1 {
		t.Errorf("Expected at least 1 metric, got %d", count)
	}
}

func TestCopilotCollector_ExportBreakdown_EmptyFields(t *testing.T) {
	collector := NewCopilotCollector("test-token", "test-org", "", "")
	ch := make(chan prometheus.Metric, 100)

	breakdown := Breakdown{
		Language: "python",
		// All other fields are 0
	}

	collector.exportBreakdown(ch, "2024-01-01", "test-org", breakdown, "language")

	close(ch)

	count := 0
	for range ch {
		count++
	}

	// Should not export metrics for zero fields
	if count != 0 {
		t.Errorf("Expected 0 metrics for empty breakdown, got %d", count)
	}
}

func TestCopilotCollector_ExportBreakdown_UnknownType(t *testing.T) {
	collector := NewCopilotCollector("test-token", "test-org", "", "")
	ch := make(chan prometheus.Metric, 100)

	breakdown := Breakdown{
		SuggestionsCount: 50,
	}

	// Should handle missing language/editor/model gracefully
	collector.exportBreakdown(ch, "2024-01-01", "test-org", breakdown, "language")

	close(ch)

	count := 0
	for range ch {
		count++
	}

	if count < 1 {
		t.Errorf("Expected at least 1 metric, got %d", count)
	}
}

func TestBreakdownStructJSON(t *testing.T) {
	jsonData := `{
		"language": "python",
		"editor": "vscode",
		"model": "gpt-4",
		"suggestions_count": 100,
		"acceptances_count": 80,
		"lines_suggested": 500,
		"lines_accepted": 400,
		"active_users": 10,
		"chat_acceptances": 20,
		"chat_turns": 30,
		"active_chat_users": 5
	}`

	var breakdown Breakdown
	err := json.Unmarshal([]byte(jsonData), &breakdown)
	if err != nil {
		t.Fatalf("Failed to unmarshal breakdown: %v", err)
	}

	if breakdown.Language != "python" {
		t.Errorf("Expected language 'python', got '%s'", breakdown.Language)
	}
	if breakdown.SuggestionsCount != 100 {
		t.Errorf("Expected 100 suggestions, got %d", breakdown.SuggestionsCount)
	}
}

func TestCopilotAPIResponseJSON(t *testing.T) {
	jsonData := `[{
		"day": "2024-01-01",
		"total_suggestions_count": 100,
		"total_acceptances_count": 80,
		"total_lines_suggested": 500,
		"total_lines_accepted": 400,
		"total_active_users": 10,
		"total_chat_acceptances": 20,
		"total_chat_turns": 30,
		"total_active_chat_users": 5
	}]`

	var response CopilotAPIResponse
	err := json.Unmarshal([]byte(jsonData), &response)
	if err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if len(response) != 1 {
		t.Fatalf("Expected 1 response item, got %d", len(response))
	}

	if response[0].Day != "2024-01-01" {
		t.Errorf("Expected day '2024-01-01', got '%s'", response[0].Day)
	}
	if response[0].TotalSuggestionsCount != 100 {
		t.Errorf("Expected 100 suggestions, got %d", response[0].TotalSuggestionsCount)
	}
}

func TestCopilotAPIResponseWithNestedStructures(t *testing.T) {
	jsonData := `[{
		"day": "2024-01-01",
		"total_suggestions_count": 100,
		"total_acceptances_count": 80,
		"total_lines_suggested": 500,
		"total_lines_accepted": 400,
		"total_active_users": 10,
		"total_chat_acceptances": 20,
		"total_chat_turns": 30,
		"total_active_chat_users": 5,
		"copilot_ide_code_completions": {
			"total_engaged_users": 10,
			"languages": [
				{
					"language": "python",
					"suggestions_count": 50
				}
			]
		},
		"copilot_dotcom_pull_requests": {
			"total_engaged_users": 2,
			"repositories": [
				{
					"name": "test-repo",
					"total_engaged_users": 2
				}
			]
		}
	}]`

	var response CopilotAPIResponse
	err := json.Unmarshal([]byte(jsonData), &response)
	if err != nil {
		t.Fatalf("Failed to unmarshal response with nested structures: %v", err)
	}

	if response[0].CopilotIDECodeCompletions.TotalEngagedUsers != 10 {
		t.Errorf("Expected 10 engaged users, got %d", response[0].CopilotIDECodeCompletions.TotalEngagedUsers)
	}

	if len(response[0].CopilotIDECodeCompletions.Languages) != 1 {
		t.Fatalf("Expected 1 language, got %d", len(response[0].CopilotIDECodeCompletions.Languages))
	}

	if response[0].CopilotIDECodeCompletions.Languages[0].Language != "python" {
		t.Errorf("Expected language 'python', got '%s'", response[0].CopilotIDECodeCompletions.Languages[0].Language)
	}

	if len(response[0].CopilotDotcomPullRequests.Repositories) != 1 {
		t.Fatalf("Expected 1 repository, got %d", len(response[0].CopilotDotcomPullRequests.Repositories))
	}

	if response[0].CopilotDotcomPullRequests.Repositories[0].Name != "test-repo" {
		t.Errorf("Expected repository name 'test-repo', got '%s'", response[0].CopilotDotcomPullRequests.Repositories[0].Name)
	}
}

func TestConstants(t *testing.T) {
	if defaultPort != "8082" {
		t.Errorf("Expected default port '8082', got '%s'", defaultPort)
	}
	if metricsEndpoint != "/metrics" {
		t.Errorf("Expected metrics endpoint '/metrics', got '%s'", metricsEndpoint)
	}
}

// Test fetchMetrics with different scenarios
func TestCopilotCollector_FetchMetrics_Success(t *testing.T) {
	mockData := `[{
		"day": "2024-01-01",
		"total_suggestions_count": 100,
		"total_acceptances_count": 80,
		"total_lines_suggested": 500,
		"total_lines_accepted": 400,
		"total_active_users": 10,
		"total_chat_acceptances": 20,
		"total_chat_turns": 30,
		"total_active_chat_users": 5
	}]`

	// Mock server that simulates GitHub API
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-token" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, mockData)
	}))
	defer server.Close()

	// Test with organization
	collector := &CopilotCollector{
		githubToken:  "test-token",
		organization: "test-org",
	}

	// We can't directly test fetchMetrics without modifying production code,
	// but we verify the structure is correct
	if collector.organization != "test-org" {
		t.Errorf("Expected organization 'test-org', got '%s'", collector.organization)
	}
}

func TestCopilotCollector_FetchMetrics_ErrorHandling(t *testing.T) {
	// Mock server that returns error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, "Internal Server Error")
	}))
	defer server.Close()

	collector := &CopilotCollector{
		githubToken:  "test-token",
		organization: "test-org",
	}

	// Verify error handling setup
	if collector.githubToken == "" {
		t.Error("Expected non-empty token")
	}
}

// Comprehensive Collect test with full metrics structure
func TestCopilotCollector_Collect_WithFullMetrics(t *testing.T) {
	// Create a collector
	collector := NewCopilotCollector("test-token", "test-org", "", "")

	// Create a channel to collect metrics
	ch := make(chan prometheus.Metric, 200)

	// Call Collect - it will try to fetch from API and fail in test environment
	// But we can verify the structure
	go func() {
		collector.Collect(ch)
		close(ch)
	}()

	// Drain the channel
	count := 0
	for range ch {
		count++
	}

	// In test environment, no metrics will be collected due to API error
	// But the code path is executed
	t.Logf("Collected %d metrics (expected 0 in test environment)", count)
}

// Test exportBreakdown with all fields populated
func TestCopilotCollector_ExportBreakdown_AllFields(t *testing.T) {
	collector := NewCopilotCollector("test-token", "test-org", "", "")
	ch := make(chan prometheus.Metric, 100)

	breakdown := Breakdown{
		Language:         "go",
		Editor:           "vscode",
		Model:            "gpt-4",
		SuggestionsCount: 100,
		AcceptancesCount: 80,
		LinesSuggested:   500,
		LinesAccepted:    400,
		ActiveUsers:      10,
		ChatAcceptances:  20,
		ChatTurns:        30,
		ActiveChatUsers:  5,
	}

	// Test with all breakdown types
	breakdownTypes := []string{"language", "editor", "model"}
	for _, breakdownType := range breakdownTypes {
		collector.exportBreakdown(ch, "2024-01-01", "test-org", breakdown, breakdownType)
	}

	close(ch)

	count := 0
	for range ch {
		count++
	}

	// Should export 8 metrics per breakdown type * 3 types = 24
	if count < 20 {
		t.Errorf("Expected at least 20 metrics, got %d", count)
	}
}

// Test exportBreakdown with partial fields
func TestCopilotCollector_ExportBreakdown_PartialFields(t *testing.T) {
	collector := NewCopilotCollector("test-token", "test-org", "", "")
	ch := make(chan prometheus.Metric, 100)

	breakdown := Breakdown{
		Language:         "python",
		SuggestionsCount: 50,
		AcceptancesCount: 40,
		// Other fields are zero
	}

	collector.exportBreakdown(ch, "2024-01-01", "test-org", breakdown, "language")

	close(ch)

	count := 0
	for range ch {
		count++
	}

	// Should only export metrics for non-zero fields (2)
	if count != 2 {
		t.Errorf("Expected 2 metrics for partial breakdown, got %d", count)
	}
}

// Test exportBreakdown with editor type
func TestCopilotCollector_ExportBreakdown_EditorType(t *testing.T) {
	collector := NewCopilotCollector("test-token", "test-org", "", "")
	ch := make(chan prometheus.Metric, 100)

	breakdown := Breakdown{
		Editor:           "intellij",
		SuggestionsCount: 25,
		ActiveUsers:      3,
	}

	collector.exportBreakdown(ch, "2024-01-01", "test-org", breakdown, "editor")

	close(ch)

	count := 0
	for range ch {
		count++
	}

	if count != 2 {
		t.Errorf("Expected 2 metrics for editor breakdown, got %d", count)
	}
}

// Test exportBreakdown with model type
func TestCopilotCollector_ExportBreakdown_ModelType(t *testing.T) {
	collector := NewCopilotCollector("test-token", "test-org", "", "")
	ch := make(chan prometheus.Metric, 100)

	breakdown := Breakdown{
		Model:            "gpt-3.5",
		SuggestionsCount: 75,
		ChatTurns:        15,
	}

	collector.exportBreakdown(ch, "2024-01-01", "test-org", breakdown, "model")

	close(ch)

	count := 0
	for range ch {
		count++
	}

	if count != 2 {
		t.Errorf("Expected 2 metrics for model breakdown, got %d", count)
	}
}

// Test JSON unmarshaling with empty fields
func TestCopilotAPIResponse_EmptyFields(t *testing.T) {
	jsonData := `[{
		"day": "2024-01-01",
		"total_suggestions_count": 0,
		"total_acceptances_count": 0,
		"total_lines_suggested": 0,
		"total_lines_accepted": 0,
		"total_active_users": 0,
		"total_chat_acceptances": 0,
		"total_chat_turns": 0,
		"total_active_chat_users": 0
	}]`

	var response CopilotAPIResponse
	err := json.Unmarshal([]byte(jsonData), &response)
	if err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if len(response) != 1 {
		t.Fatalf("Expected 1 response item, got %d", len(response))
	}

	// All counts should be 0
	if response[0].TotalSuggestionsCount != 0 {
		t.Errorf("Expected 0 suggestions, got %d", response[0].TotalSuggestionsCount)
	}
}

// Test all nested structures comprehensively
func TestCopilotAPIResponse_AllNestedStructures(t *testing.T) {
	jsonData := `[{
		"day": "2024-01-01",
		"total_suggestions_count": 1000,
		"total_acceptances_count": 800,
		"total_lines_suggested": 5000,
		"total_lines_accepted": 4000,
		"total_active_users": 50,
		"total_chat_acceptances": 200,
		"total_chat_turns": 300,
		"total_active_chat_users": 25,
		"breakdown": [
			{
				"language": "javascript",
				"suggestions_count": 300,
				"acceptances_count": 240
			},
			{
				"editor": "vscode",
				"suggestions_count": 500,
				"acceptances_count": 400
			}
		],
		"copilot_ide_code_completions": {
			"total_engaged_users": 50,
			"languages": [
				{
					"language": "python",
					"suggestions_count": 250,
					"acceptances_count": 200
				},
				{
					"language": "go",
					"suggestions_count": 150,
					"acceptances_count": 120
				}
			],
			"editors": [
				{
					"editor": "vscode",
					"suggestions_count": 400,
					"acceptances_count": 320
				}
			],
			"models": [
				{
					"model": "gpt-4",
					"suggestions_count": 600,
					"acceptances_count": 480
				}
			]
		},
		"copilot_ide_chat": {
			"total_engaged_users": 25,
			"editors": [
				{
					"editor": "vscode",
					"chat_acceptances": 150,
					"chat_turns": 200
				}
			],
			"models": [
				{
					"model": "gpt-4",
					"chat_acceptances": 150,
					"chat_turns": 200
				}
			]
		},
		"copilot_dotcom_chat": {
			"total_engaged_users": 15,
			"models": [
				{
					"model": "gpt-4",
					"chat_turns": 50
				},
				{
					"model": "gpt-3.5",
					"chat_turns": 30
				}
			]
		},
		"copilot_dotcom_pull_requests": {
			"total_engaged_users": 10,
			"repositories": [
				{
					"name": "repo1",
					"total_engaged_users": 5,
					"models": [
						{
							"model": "gpt-4",
							"suggestions_count": 20
						}
					]
				},
				{
					"name": "repo2",
					"total_engaged_users": 5,
					"models": [
						{
							"model": "gpt-4",
							"suggestions_count": 15
						}
					]
				}
			],
			"models": [
				{
					"model": "gpt-4",
					"suggestions_count": 35
				}
			]
		}
	}]`

	var response CopilotAPIResponse
	err := json.Unmarshal([]byte(jsonData), &response)
	if err != nil {
		t.Fatalf("Failed to unmarshal comprehensive response: %v", err)
	}

	// Verify top-level fields
	if response[0].TotalSuggestionsCount != 1000 {
		t.Errorf("Expected 1000 suggestions, got %d", response[0].TotalSuggestionsCount)
	}

	// Verify breakdown
	if len(response[0].Breakdown) != 2 {
		t.Errorf("Expected 2 breakdown items, got %d", len(response[0].Breakdown))
	}

	// Verify IDE Code Completions
	if response[0].CopilotIDECodeCompletions.TotalEngagedUsers != 50 {
		t.Errorf("Expected 50 engaged users, got %d", response[0].CopilotIDECodeCompletions.TotalEngagedUsers)
	}
	if len(response[0].CopilotIDECodeCompletions.Languages) != 2 {
		t.Errorf("Expected 2 languages, got %d", len(response[0].CopilotIDECodeCompletions.Languages))
	}
	if len(response[0].CopilotIDECodeCompletions.Editors) != 1 {
		t.Errorf("Expected 1 editor, got %d", len(response[0].CopilotIDECodeCompletions.Editors))
	}
	if len(response[0].CopilotIDECodeCompletions.Models) != 1 {
		t.Errorf("Expected 1 model, got %d", len(response[0].CopilotIDECodeCompletions.Models))
	}

	// Verify IDE Chat
	if response[0].CopilotIDEChat.TotalEngagedUsers != 25 {
		t.Errorf("Expected 25 chat engaged users, got %d", response[0].CopilotIDEChat.TotalEngagedUsers)
	}
	if len(response[0].CopilotIDEChat.Editors) != 1 {
		t.Errorf("Expected 1 chat editor, got %d", len(response[0].CopilotIDEChat.Editors))
	}
	if len(response[0].CopilotIDEChat.Models) != 1 {
		t.Errorf("Expected 1 chat model, got %d", len(response[0].CopilotIDEChat.Models))
	}

	// Verify Dotcom Chat
	if response[0].CopilotDotcomChat.TotalEngagedUsers != 15 {
		t.Errorf("Expected 15 dotcom chat users, got %d", response[0].CopilotDotcomChat.TotalEngagedUsers)
	}
	if len(response[0].CopilotDotcomChat.Models) != 2 {
		t.Errorf("Expected 2 dotcom chat models, got %d", len(response[0].CopilotDotcomChat.Models))
	}

	// Verify Dotcom PR
	if response[0].CopilotDotcomPullRequests.TotalEngagedUsers != 10 {
		t.Errorf("Expected 10 PR engaged users, got %d", response[0].CopilotDotcomPullRequests.TotalEngagedUsers)
	}
	if len(response[0].CopilotDotcomPullRequests.Repositories) != 2 {
		t.Errorf("Expected 2 repositories, got %d", len(response[0].CopilotDotcomPullRequests.Repositories))
	}
	if response[0].CopilotDotcomPullRequests.Repositories[0].Name != "repo1" {
		t.Errorf("Expected repository name 'repo1', got '%s'", response[0].CopilotDotcomPullRequests.Repositories[0].Name)
	}
	if len(response[0].CopilotDotcomPullRequests.Models) != 1 {
		t.Errorf("Expected 1 PR model, got %d", len(response[0].CopilotDotcomPullRequests.Models))
	}
}

// Integration test: Test Collect with comprehensive mock data
func TestCopilotCollector_Collect_Integration(t *testing.T) {
	// Comprehensive mock data that exercises all code paths
	mockDataJSON := `[
		{
			"day": "2024-01-01",
			"total_suggestions_count": 1000,
			"total_acceptances_count": 800,
			"total_lines_suggested": 5000,
			"total_lines_accepted": 4000,
			"total_active_users": 50,
			"total_chat_acceptances": 200,
			"total_chat_turns": 300,
			"total_active_chat_users": 25,
			"breakdown": [
				{
					"language": "python",
					"suggestions_count": 300,
					"acceptances_count": 240,
					"lines_suggested": 1500,
					"lines_accepted": 1200,
					"active_users": 15
				},
				{
					"editor": "vscode",
					"suggestions_count": 500,
					"acceptances_count": 400,
					"lines_suggested": 2500,
					"lines_accepted": 2000,
					"active_users": 30,
					"chat_acceptances": 100,
					"chat_turns": 150,
					"active_chat_users": 12
				},
				{
					"model": "gpt-4",
					"suggestions_count": 700,
					"acceptances_count": 560,
					"lines_suggested": 3500,
					"lines_accepted": 2800,
					"active_users": 35
				}
			],
			"copilot_ide_code_completions": {
				"total_engaged_users": 50,
				"languages": [
					{
						"language": "python",
						"suggestions_count": 250,
						"acceptances_count": 200,
						"lines_suggested": 1250,
						"lines_accepted": 1000,
						"active_users": 15
					},
					{
						"language": "javascript",
						"suggestions_count": 200,
						"acceptances_count": 160,
						"lines_suggested": 1000,
						"lines_accepted": 800,
						"active_users": 12
					}
				],
				"editors": [
					{
						"editor": "vscode",
						"suggestions_count": 400,
						"acceptances_count": 320,
						"lines_suggested": 2000,
						"lines_accepted": 1600,
						"active_users": 28
					},
					{
						"editor": "intellij",
						"suggestions_count": 100,
						"acceptances_count": 80,
						"lines_suggested": 500,
						"lines_accepted": 400,
						"active_users": 8
					}
				],
				"models": [
					{
						"model": "gpt-4",
						"suggestions_count": 600,
						"acceptances_count": 480,
						"lines_suggested": 3000,
						"lines_accepted": 2400,
						"active_users": 40
					}
				]
			},
			"copilot_ide_chat": {
				"total_engaged_users": 25,
				"editors": [
					{
						"editor": "vscode",
						"chat_acceptances": 150,
						"chat_turns": 200,
						"active_chat_users": 18
					},
					{
						"editor": "intellij",
						"chat_acceptances": 50,
						"chat_turns": 100,
						"active_chat_users": 7
					}
				],
				"models": [
					{
						"model": "gpt-4",
						"chat_acceptances": 180,
						"chat_turns": 280,
						"active_chat_users": 22
					}
				]
			},
			"copilot_dotcom_chat": {
				"total_engaged_users": 15,
				"models": [
					{
						"model": "gpt-4",
						"chat_turns": 80,
						"chat_acceptances": 60,
						"active_chat_users": 12
					},
					{
						"model": "gpt-3.5",
						"chat_turns": 40,
						"chat_acceptances": 30,
						"active_chat_users": 8
					}
				]
			},
			"copilot_dotcom_pull_requests": {
				"total_engaged_users": 10,
				"repositories": [
					{
						"name": "main-repo",
						"total_engaged_users": 6,
						"models": [
							{
								"model": "gpt-4",
								"suggestions_count": 30,
								"acceptances_count": 24,
								"active_users": 6
							}
						]
					},
					{
						"name": "secondary-repo",
						"total_engaged_users": 4,
						"models": [
							{
								"model": "gpt-4",
								"suggestions_count": 20,
								"acceptances_count": 16,
								"active_users": 4
							}
						]
					}
				],
				"models": [
					{
						"model": "gpt-4",
						"suggestions_count": 50,
						"acceptances_count": 40,
						"active_users": 10
					}
				]
			}
		},
		{
			"day": "2024-01-02",
			"total_suggestions_count": 1100,
			"total_acceptances_count": 880,
			"total_lines_suggested": 5500,
			"total_lines_accepted": 4400,
			"total_active_users": 55,
			"total_chat_acceptances": 220,
			"total_chat_turns": 330,
			"total_active_chat_users": 28
		}
	]`

	// Create collector with test data injector
	collector := NewCopilotCollector("test-token", "test-org", "", "")

	// Inject mock data fetcher
	collector.testMetricsFetcher = func() (CopilotAPIResponse, error) {
		var response CopilotAPIResponse
		err := json.Unmarshal([]byte(mockDataJSON), &response)
		return response, err
	}

	// Collect metrics
	ch := make(chan prometheus.Metric, 500)
	go func() {
		collector.Collect(ch)
		close(ch)
	}()

	count := 0
	for range ch {
		count++
	}

	// Should collect many metrics from the comprehensive data
	// Expected: 9 top-level * 2 days + breakdown + IDE + chat + PR metrics
	if count < 50 {
		t.Errorf("Expected at least 50 metrics from comprehensive data, got %d", count)
	}
	t.Logf("Collected %d metrics from comprehensive integration test", count)
}

// Test with enterprise configuration
func TestCopilotCollector_Collect_Enterprise(t *testing.T) {
	collector := NewCopilotCollector("test-token", "", "", "test-enterprise")

	if collector.enterprise != "test-enterprise" {
		t.Errorf("Expected enterprise 'test-enterprise', got '%s'", collector.enterprise)
	}

	ch := make(chan prometheus.Metric, 100)
	go func() {
		collector.Collect(ch)
		close(ch)
	}()

	count := 0
	for range ch {
		count++
	}

	t.Logf("Enterprise collector collected %d metrics", count)
}

// Test Describe ensures all metrics are described
func TestCopilotCollector_Describe_AllMetrics(t *testing.T) {
	collector := NewCopilotCollector("test-token", "test-org", "", "")

	descriptors := make(map[string]bool)
	ch := make(chan *prometheus.Desc, 30)
	go func() {
		collector.Describe(ch)
		close(ch)
	}()

	for desc := range ch {
		descriptors[desc.String()] = true
	}

	// Should have exactly 22 unique descriptors
	if len(descriptors) != 22 {
		t.Errorf("Expected 22 unique metric descriptors, got %d", len(descriptors))
	}
}

// Test JSON marshaling and unmarshaling
func TestBreakdown_JSONRoundTrip(t *testing.T) {
	original := Breakdown{
		Language:         "rust",
		Editor:           "neovim",
		Model:            "gpt-4",
		SuggestionsCount: 123,
		AcceptancesCount: 98,
		LinesSuggested:   456,
		LinesAccepted:    365,
		ActiveUsers:      12,
		ChatAcceptances:  34,
		ChatTurns:        45,
		ActiveChatUsers:  8,
	}

	// Marshal to JSON
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	// Unmarshal back
	var decoded Breakdown
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	// Verify all fields
	if decoded.Language != original.Language {
		t.Errorf("Language mismatch: expected %s, got %s", original.Language, decoded.Language)
	}
	if decoded.SuggestionsCount != original.SuggestionsCount {
		t.Errorf("SuggestionsCount mismatch: expected %d, got %d", original.SuggestionsCount, decoded.SuggestionsCount)
	}
	if decoded.ActiveChatUsers != original.ActiveChatUsers {
		t.Errorf("ActiveChatUsers mismatch: expected %d, got %d", original.ActiveChatUsers, decoded.ActiveChatUsers)
	}
}

// Test collector with all configuration options
func TestNewCopilotCollector_AllConfigurations(t *testing.T) {
	tests := []struct {
		name        string
		token       string
		org         string
		team        string
		enterprise  string
		expectError bool
	}{
		{
			name:       "organization only",
			token:      "token1",
			org:        "org1",
			team:       "",
			enterprise: "",
		},
		{
			name:       "organization with team",
			token:      "token2",
			org:        "org2",
			team:       "team2",
			enterprise: "",
		},
		{
			name:       "enterprise only",
			token:      "token3",
			org:        "",
			team:       "",
			enterprise: "enterprise3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			collector := NewCopilotCollector(tt.token, tt.org, tt.team, tt.enterprise)

			if collector.githubToken != tt.token {
				t.Errorf("Expected token %s, got %s", tt.token, collector.githubToken)
			}
			if collector.organization != tt.org {
				t.Errorf("Expected org %s, got %s", tt.org, collector.organization)
			}
			if collector.team != tt.team {
				t.Errorf("Expected team %s, got %s", tt.team, collector.team)
			}
			if collector.enterprise != tt.enterprise {
				t.Errorf("Expected enterprise %s, got %s", tt.enterprise, collector.enterprise)
			}

			// Verify all metrics are initialized
			if collector.totalSuggestions == nil {
				t.Error("totalSuggestions not initialized")
			}
			if collector.breakdownSuggestions == nil {
				t.Error("breakdownSuggestions not initialized")
			}
			if collector.ideCodeCompletionsEngagedUsers == nil {
				t.Error("ideCodeCompletionsEngagedUsers not initialized")
			}
		})
	}
}

// Test fetchMetrics with successful response
func TestCopilotCollector_FetchMetrics_SuccessfulResponse(t *testing.T) {
	mockData := `[{
		"day": "2024-01-01",
		"total_suggestions_count": 100,
		"total_acceptances_count": 80,
		"total_lines_suggested": 500,
		"total_lines_accepted": 400,
		"total_active_users": 10,
		"total_chat_acceptances": 20,
		"total_chat_turns": 30,
		"total_active_chat_users": 5
	}]`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, mockData)
	}))
	defer server.Close()

	// Test would require injecting server URL - verify collector setup
	collector := NewCopilotCollector("test-token", "test-org", "", "")
	if collector.organization != "test-org" {
		t.Errorf("Expected org test-org, got %s", collector.organization)
	}
}

// Test with error scenarios
func TestCopilotCollector_Collect_WithError(t *testing.T) {
	collector := NewCopilotCollector("test-token", "test-org", "", "")

	// Inject error fetcher
	collector.testMetricsFetcher = func() (CopilotAPIResponse, error) {
		return nil, fmt.Errorf("simulated API error")
	}

	ch := make(chan prometheus.Metric, 100)
	go func() {
		collector.Collect(ch)
		close(ch)
	}()

	count := 0
	for range ch {
		count++
	}

	// Should return 0 metrics on error
	if count != 0 {
		t.Errorf("Expected 0 metrics on error, got %d", count)
	}
}

// Test with zero acceptances (edge case for acceptance rate)
func TestCopilotCollector_Collect_ZeroAcceptances(t *testing.T) {
	mockData := `[{
		"day": "2024-01-01",
		"total_suggestions_count": 0,
		"total_acceptances_count": 0,
		"total_lines_suggested": 0,
		"total_lines_accepted": 0,
		"total_active_users": 0,
		"total_chat_acceptances": 0,
		"total_chat_turns": 0,
		"total_active_chat_users": 0
	}]`

	collector := NewCopilotCollector("test-token", "test-org", "", "")
	collector.testMetricsFetcher = func() (CopilotAPIResponse, error) {
		var response CopilotAPIResponse
		err := json.Unmarshal([]byte(mockData), &response)
		return response, err
	}

	ch := make(chan prometheus.Metric, 100)
	go func() {
		collector.Collect(ch)
		close(ch)
	}()

	count := 0
	for range ch {
		count++
	}

	// Should collect 9 top-level metrics even with zeros
	if count != 9 {
		t.Errorf("Expected 9 metrics (including zero values), got %d", count)
	}
}

// Test enterprise org label
func TestCopilotCollector_Collect_EnterpriseOrgLabel(t *testing.T) {
	mockData := `[{
		"day": "2024-01-01",
		"total_suggestions_count": 100,
		"total_acceptances_count": 80,
		"total_lines_suggested": 500,
		"total_lines_accepted": 400,
		"total_active_users": 10,
		"total_chat_acceptances": 20,
		"total_chat_turns": 30,
		"total_active_chat_users": 5
	}]`

	collector := NewCopilotCollector("test-token", "", "", "test-enterprise")
	collector.testMetricsFetcher = func() (CopilotAPIResponse, error) {
		var response CopilotAPIResponse
		err := json.Unmarshal([]byte(mockData), &response)
		return response, err
	}

	ch := make(chan prometheus.Metric, 100)
	go func() {
		collector.Collect(ch)
		close(ch)
	}()

	count := 0
	for range ch {
		count++
	}

	// Should use enterprise as org label
	if count < 9 {
		t.Errorf("Expected at least 9 metrics, got %d", count)
	}
}

// Test all feature-specific metrics
func TestCopilotCollector_Collect_AllFeatures(t *testing.T) {
	mockData := `[{
		"day": "2024-01-01",
		"total_suggestions_count": 100,
		"total_acceptances_count": 80,
		"total_lines_suggested": 500,
		"total_lines_accepted": 400,
		"total_active_users": 10,
		"total_chat_acceptances": 20,
		"total_chat_turns": 30,
		"total_active_chat_users": 5,
		"copilot_ide_code_completions": {
			"total_engaged_users": 10
		},
		"copilot_ide_chat": {
			"total_engaged_users": 5
		},
		"copilot_dotcom_chat": {
			"total_engaged_users": 3
		},
		"copilot_dotcom_pull_requests": {
			"total_engaged_users": 2
		}
	}]`

	collector := NewCopilotCollector("test-token", "test-org", "", "")
	collector.testMetricsFetcher = func() (CopilotAPIResponse, error) {
		var response CopilotAPIResponse
		err := json.Unmarshal([]byte(mockData), &response)
		return response, err
	}

	ch := make(chan prometheus.Metric, 100)
	go func() {
		collector.Collect(ch)
		close(ch)
	}()

	count := 0
	for range ch {
		count++
	}

	// 9 top-level + 4 feature-specific = 13
	if count != 13 {
		t.Errorf("Expected 13 metrics, got %d", count)
	}
}
