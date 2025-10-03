package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	defaultPort     = "8082"
	metricsEndpoint = "/metrics"
)

// Breakdown represents breakdown of metrics by editor, language, or model
type Breakdown struct {
	Language         string `json:"language,omitempty"`
	Editor           string `json:"editor,omitempty"`
	Model            string `json:"model,omitempty"`
	SuggestionsCount int    `json:"suggestions_count,omitempty"`
	AcceptancesCount int    `json:"acceptances_count,omitempty"`
	LinesSuggested   int    `json:"lines_suggested,omitempty"`
	LinesAccepted    int    `json:"lines_accepted,omitempty"`
	ActiveUsers      int    `json:"active_users,omitempty"`
	ChatAcceptances  int    `json:"chat_acceptances,omitempty"`
	ChatTurns        int    `json:"chat_turns,omitempty"`
	ActiveChatUsers  int    `json:"active_chat_users,omitempty"`
}

// CopilotAPIResponse represents the complete response from GitHub Copilot Metrics API
type CopilotAPIResponse []struct {
	Day                   string `json:"day"`
	TotalSuggestionsCount int    `json:"total_suggestions_count"`
	TotalAcceptancesCount int    `json:"total_acceptances_count"`
	TotalLinesSuggested   int    `json:"total_lines_suggested"`
	TotalLinesAccepted    int    `json:"total_lines_accepted"`
	TotalActiveUsers      int    `json:"total_active_users"`
	TotalChatAcceptances  int    `json:"total_chat_acceptances"`
	TotalChatTurns        int    `json:"total_chat_turns"`
	TotalActiveChatUsers  int    `json:"total_active_chat_users"`

	// Breakdown data
	Breakdown []Breakdown `json:"breakdown,omitempty"`

	// Copilot IDE Code Completions
	CopilotIDECodeCompletions struct {
		TotalEngagedUsers int         `json:"total_engaged_users,omitempty"`
		Languages         []Breakdown `json:"languages,omitempty"`
		Editors           []Breakdown `json:"editors,omitempty"`
		Models            []Breakdown `json:"models,omitempty"`
	} `json:"copilot_ide_code_completions,omitempty"`

	// Copilot IDE Chat
	CopilotIDEChat struct {
		TotalEngagedUsers int         `json:"total_engaged_users,omitempty"`
		Editors           []Breakdown `json:"editors,omitempty"`
		Models            []Breakdown `json:"models,omitempty"`
	} `json:"copilot_ide_chat,omitempty"`

	// Copilot Dotcom Chat
	CopilotDotcomChat struct {
		TotalEngagedUsers int         `json:"total_engaged_users,omitempty"`
		Models            []Breakdown `json:"models,omitempty"`
	} `json:"copilot_dotcom_chat,omitempty"`

	// Copilot Dotcom Pull Requests
	CopilotDotcomPullRequests struct {
		TotalEngagedUsers int `json:"total_engaged_users,omitempty"`
		Repositories      []struct {
			Name              string      `json:"name,omitempty"`
			TotalEngagedUsers int         `json:"total_engaged_users,omitempty"`
			Models            []Breakdown `json:"models,omitempty"`
		} `json:"repositories,omitempty"`
		Models []Breakdown `json:"models,omitempty"`
	} `json:"copilot_dotcom_pull_requests,omitempty"`
}

type CopilotCollector struct {
	githubToken  string
	organization string
	team         string
	enterprise   string

	// Top-level metrics
	totalSuggestions     *prometheus.Desc
	totalAcceptances     *prometheus.Desc
	totalLinesSuggested  *prometheus.Desc
	totalLinesAccepted   *prometheus.Desc
	totalActiveUsers     *prometheus.Desc
	totalChatAcceptances *prometheus.Desc
	totalChatTurns       *prometheus.Desc
	totalActiveChatUsers *prometheus.Desc
	acceptanceRate       *prometheus.Desc

	// Breakdown metrics (by language, editor, model)
	breakdownSuggestions     *prometheus.Desc
	breakdownAcceptances     *prometheus.Desc
	breakdownLinesSuggested  *prometheus.Desc
	breakdownLinesAccepted   *prometheus.Desc
	breakdownActiveUsers     *prometheus.Desc
	breakdownChatAcceptances *prometheus.Desc
	breakdownChatTurns       *prometheus.Desc
	breakdownActiveChatUsers *prometheus.Desc

	// IDE Code Completions
	ideCodeCompletionsEngagedUsers *prometheus.Desc

	// IDE Chat
	ideChatEngagedUsers *prometheus.Desc

	// Dotcom Chat
	dotcomChatEngagedUsers *prometheus.Desc

	// Dotcom Pull Requests
	dotcomPREngagedUsers     *prometheus.Desc
	dotcomPRRepoEngagedUsers *prometheus.Desc
}

func NewCopilotCollector(githubToken, organization, team, enterprise string) *CopilotCollector {
	return &CopilotCollector{
		githubToken:  githubToken,
		organization: organization,
		team:         team,
		enterprise:   enterprise,
		totalSuggestions: prometheus.NewDesc(
			"github_copilot_suggestions_total",
			"Total number of Copilot suggestions",
			[]string{"day", "org"},
			nil,
		),
		totalAcceptances: prometheus.NewDesc(
			"github_copilot_acceptances_total",
			"Total number of Copilot acceptances",
			[]string{"day", "org"},
			nil,
		),
		totalLinesSuggested: prometheus.NewDesc(
			"github_copilot_lines_suggested_total",
			"Total number of lines suggested by Copilot",
			[]string{"day", "org"},
			nil,
		),
		totalLinesAccepted: prometheus.NewDesc(
			"github_copilot_lines_accepted_total",
			"Total number of lines accepted from Copilot",
			[]string{"day", "org"},
			nil,
		),
		totalActiveUsers: prometheus.NewDesc(
			"github_copilot_active_users_total",
			"Total number of active Copilot users",
			[]string{"day", "org"},
			nil,
		),
		totalChatAcceptances: prometheus.NewDesc(
			"github_copilot_chat_acceptances_total",
			"Total number of Copilot chat acceptances",
			[]string{"day", "org"},
			nil,
		),
		totalChatTurns: prometheus.NewDesc(
			"github_copilot_chat_turns_total",
			"Total number of Copilot chat turns",
			[]string{"day", "org"},
			nil,
		),
		totalActiveChatUsers: prometheus.NewDesc(
			"github_copilot_active_chat_users_total",
			"Total number of active Copilot chat users",
			[]string{"day", "org"},
			nil,
		),
		acceptanceRate: prometheus.NewDesc(
			"github_copilot_acceptance_rate",
			"Copilot acceptance rate (acceptances/suggestions)",
			[]string{"day", "org"},
			nil,
		),
		// Breakdown metrics with language, editor, and model labels
		breakdownSuggestions: prometheus.NewDesc(
			"github_copilot_breakdown_suggestions_total",
			"Copilot suggestions by language, editor, or model",
			[]string{"day", "org", "language", "editor", "model"},
			nil,
		),
		breakdownAcceptances: prometheus.NewDesc(
			"github_copilot_breakdown_acceptances_total",
			"Copilot acceptances by language, editor, or model",
			[]string{"day", "org", "language", "editor", "model"},
			nil,
		),
		breakdownLinesSuggested: prometheus.NewDesc(
			"github_copilot_breakdown_lines_suggested_total",
			"Lines suggested by language, editor, or model",
			[]string{"day", "org", "language", "editor", "model"},
			nil,
		),
		breakdownLinesAccepted: prometheus.NewDesc(
			"github_copilot_breakdown_lines_accepted_total",
			"Lines accepted by language, editor, or model",
			[]string{"day", "org", "language", "editor", "model"},
			nil,
		),
		breakdownActiveUsers: prometheus.NewDesc(
			"github_copilot_breakdown_active_users",
			"Active users by language, editor, or model",
			[]string{"day", "org", "language", "editor", "model"},
			nil,
		),
		breakdownChatAcceptances: prometheus.NewDesc(
			"github_copilot_breakdown_chat_acceptances_total",
			"Chat acceptances by language, editor, or model",
			[]string{"day", "org", "language", "editor", "model"},
			nil,
		),
		breakdownChatTurns: prometheus.NewDesc(
			"github_copilot_breakdown_chat_turns_total",
			"Chat turns by language, editor, or model",
			[]string{"day", "org", "language", "editor", "model"},
			nil,
		),
		breakdownActiveChatUsers: prometheus.NewDesc(
			"github_copilot_breakdown_active_chat_users",
			"Active chat users by language, editor, or model",
			[]string{"day", "org", "language", "editor", "model"},
			nil,
		),
		// IDE Code Completions
		ideCodeCompletionsEngagedUsers: prometheus.NewDesc(
			"github_copilot_ide_code_completions_engaged_users",
			"Total engaged users for IDE code completions",
			[]string{"day", "org"},
			nil,
		),
		// IDE Chat
		ideChatEngagedUsers: prometheus.NewDesc(
			"github_copilot_ide_chat_engaged_users",
			"Total engaged users for IDE chat",
			[]string{"day", "org"},
			nil,
		),
		// Dotcom Chat
		dotcomChatEngagedUsers: prometheus.NewDesc(
			"github_copilot_dotcom_chat_engaged_users",
			"Total engaged users for Dotcom chat",
			[]string{"day", "org"},
			nil,
		),
		// Dotcom Pull Requests
		dotcomPREngagedUsers: prometheus.NewDesc(
			"github_copilot_dotcom_pr_engaged_users",
			"Total engaged users for Dotcom pull requests",
			[]string{"day", "org"},
			nil,
		),
		dotcomPRRepoEngagedUsers: prometheus.NewDesc(
			"github_copilot_dotcom_pr_repo_engaged_users",
			"Engaged users for Dotcom pull requests by repository",
			[]string{"day", "org", "repository"},
			nil,
		),
	}
}

func (c *CopilotCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.totalSuggestions
	ch <- c.totalAcceptances
	ch <- c.totalLinesSuggested
	ch <- c.totalLinesAccepted
	ch <- c.totalActiveUsers
	ch <- c.totalChatAcceptances
	ch <- c.totalChatTurns
	ch <- c.totalActiveChatUsers
	ch <- c.acceptanceRate
	ch <- c.breakdownSuggestions
	ch <- c.breakdownAcceptances
	ch <- c.breakdownLinesSuggested
	ch <- c.breakdownLinesAccepted
	ch <- c.breakdownActiveUsers
	ch <- c.breakdownChatAcceptances
	ch <- c.breakdownChatTurns
	ch <- c.breakdownActiveChatUsers
	ch <- c.ideCodeCompletionsEngagedUsers
	ch <- c.ideChatEngagedUsers
	ch <- c.dotcomChatEngagedUsers
	ch <- c.dotcomPREngagedUsers
	ch <- c.dotcomPRRepoEngagedUsers
}

func (c *CopilotCollector) Collect(ch chan<- prometheus.Metric) {
	// Fetch fresh metrics on every scrape - no caching
	metrics, err := c.fetchMetrics()
	if err != nil {
		log.Printf("Error fetching metrics: %v", err)
		return
	}

	for _, metric := range metrics {
		day := metric.Day
		org := c.organization
		if c.enterprise != "" {
			org = c.enterprise
		}

		// Top-level aggregate metrics
		ch <- prometheus.MustNewConstMetric(
			c.totalSuggestions,
			prometheus.GaugeValue,
			float64(metric.TotalSuggestionsCount),
			day, org,
		)
		ch <- prometheus.MustNewConstMetric(
			c.totalAcceptances,
			prometheus.GaugeValue,
			float64(metric.TotalAcceptancesCount),
			day, org,
		)
		ch <- prometheus.MustNewConstMetric(
			c.totalLinesSuggested,
			prometheus.GaugeValue,
			float64(metric.TotalLinesSuggested),
			day, org,
		)
		ch <- prometheus.MustNewConstMetric(
			c.totalLinesAccepted,
			prometheus.GaugeValue,
			float64(metric.TotalLinesAccepted),
			day, org,
		)
		ch <- prometheus.MustNewConstMetric(
			c.totalActiveUsers,
			prometheus.GaugeValue,
			float64(metric.TotalActiveUsers),
			day, org,
		)
		ch <- prometheus.MustNewConstMetric(
			c.totalChatAcceptances,
			prometheus.GaugeValue,
			float64(metric.TotalChatAcceptances),
			day, org,
		)
		ch <- prometheus.MustNewConstMetric(
			c.totalChatTurns,
			prometheus.GaugeValue,
			float64(metric.TotalChatTurns),
			day, org,
		)
		ch <- prometheus.MustNewConstMetric(
			c.totalActiveChatUsers,
			prometheus.GaugeValue,
			float64(metric.TotalActiveChatUsers),
			day, org,
		)

		// Calculate acceptance rate
		acceptanceRate := 0.0
		if metric.TotalSuggestionsCount > 0 {
			acceptanceRate = float64(metric.TotalAcceptancesCount) / float64(metric.TotalSuggestionsCount)
		}
		ch <- prometheus.MustNewConstMetric(
			c.acceptanceRate,
			prometheus.GaugeValue,
			acceptanceRate,
			day, org,
		)

		// Breakdown metrics (generic breakdown array)
		for _, breakdown := range metric.Breakdown {
			language := breakdown.Language
			editor := breakdown.Editor
			model := breakdown.Model

			if breakdown.SuggestionsCount > 0 {
				ch <- prometheus.MustNewConstMetric(
					c.breakdownSuggestions,
					prometheus.GaugeValue,
					float64(breakdown.SuggestionsCount),
					day, org, language, editor, model,
				)
			}
			if breakdown.AcceptancesCount > 0 {
				ch <- prometheus.MustNewConstMetric(
					c.breakdownAcceptances,
					prometheus.GaugeValue,
					float64(breakdown.AcceptancesCount),
					day, org, language, editor, model,
				)
			}
			if breakdown.LinesSuggested > 0 {
				ch <- prometheus.MustNewConstMetric(
					c.breakdownLinesSuggested,
					prometheus.GaugeValue,
					float64(breakdown.LinesSuggested),
					day, org, language, editor, model,
				)
			}
			if breakdown.LinesAccepted > 0 {
				ch <- prometheus.MustNewConstMetric(
					c.breakdownLinesAccepted,
					prometheus.GaugeValue,
					float64(breakdown.LinesAccepted),
					day, org, language, editor, model,
				)
			}
			if breakdown.ActiveUsers > 0 {
				ch <- prometheus.MustNewConstMetric(
					c.breakdownActiveUsers,
					prometheus.GaugeValue,
					float64(breakdown.ActiveUsers),
					day, org, language, editor, model,
				)
			}
			if breakdown.ChatAcceptances > 0 {
				ch <- prometheus.MustNewConstMetric(
					c.breakdownChatAcceptances,
					prometheus.GaugeValue,
					float64(breakdown.ChatAcceptances),
					day, org, language, editor, model,
				)
			}
			if breakdown.ChatTurns > 0 {
				ch <- prometheus.MustNewConstMetric(
					c.breakdownChatTurns,
					prometheus.GaugeValue,
					float64(breakdown.ChatTurns),
					day, org, language, editor, model,
				)
			}
			if breakdown.ActiveChatUsers > 0 {
				ch <- prometheus.MustNewConstMetric(
					c.breakdownActiveChatUsers,
					prometheus.GaugeValue,
					float64(breakdown.ActiveChatUsers),
					day, org, language, editor, model,
				)
			}
		}

		// IDE Code Completions
		if metric.CopilotIDECodeCompletions.TotalEngagedUsers > 0 {
			ch <- prometheus.MustNewConstMetric(
				c.ideCodeCompletionsEngagedUsers,
				prometheus.GaugeValue,
				float64(metric.CopilotIDECodeCompletions.TotalEngagedUsers),
				day, org,
			)
		}

		// IDE Code Completions - Languages breakdown
		for _, lang := range metric.CopilotIDECodeCompletions.Languages {
			c.exportBreakdown(ch, day, org, lang, "language")
		}

		// IDE Code Completions - Editors breakdown
		for _, editor := range metric.CopilotIDECodeCompletions.Editors {
			c.exportBreakdown(ch, day, org, editor, "editor")
		}

		// IDE Code Completions - Models breakdown
		for _, model := range metric.CopilotIDECodeCompletions.Models {
			c.exportBreakdown(ch, day, org, model, "model")
		}

		// IDE Chat
		if metric.CopilotIDEChat.TotalEngagedUsers > 0 {
			ch <- prometheus.MustNewConstMetric(
				c.ideChatEngagedUsers,
				prometheus.GaugeValue,
				float64(metric.CopilotIDEChat.TotalEngagedUsers),
				day, org,
			)
		}

		// IDE Chat - Editors breakdown
		for _, editor := range metric.CopilotIDEChat.Editors {
			c.exportBreakdown(ch, day, org, editor, "editor")
		}

		// IDE Chat - Models breakdown
		for _, model := range metric.CopilotIDEChat.Models {
			c.exportBreakdown(ch, day, org, model, "model")
		}

		// Dotcom Chat
		if metric.CopilotDotcomChat.TotalEngagedUsers > 0 {
			ch <- prometheus.MustNewConstMetric(
				c.dotcomChatEngagedUsers,
				prometheus.GaugeValue,
				float64(metric.CopilotDotcomChat.TotalEngagedUsers),
				day, org,
			)
		}

		// Dotcom Chat - Models breakdown
		for _, model := range metric.CopilotDotcomChat.Models {
			c.exportBreakdown(ch, day, org, model, "model")
		}

		// Dotcom Pull Requests
		if metric.CopilotDotcomPullRequests.TotalEngagedUsers > 0 {
			ch <- prometheus.MustNewConstMetric(
				c.dotcomPREngagedUsers,
				prometheus.GaugeValue,
				float64(metric.CopilotDotcomPullRequests.TotalEngagedUsers),
				day, org,
			)
		}

		// Dotcom Pull Requests - Repositories
		for _, repo := range metric.CopilotDotcomPullRequests.Repositories {
			if repo.TotalEngagedUsers > 0 {
				ch <- prometheus.MustNewConstMetric(
					c.dotcomPRRepoEngagedUsers,
					prometheus.GaugeValue,
					float64(repo.TotalEngagedUsers),
					day, org, repo.Name,
				)
			}

			// Repository models breakdown
			for _, model := range repo.Models {
				c.exportBreakdown(ch, day, org, model, "model")
			}
		}

		// Dotcom Pull Requests - Models breakdown
		for _, model := range metric.CopilotDotcomPullRequests.Models {
			c.exportBreakdown(ch, day, org, model, "model")
		}
	}
}

// Helper function to export breakdown metrics
func (c *CopilotCollector) exportBreakdown(ch chan<- prometheus.Metric, day, org string, breakdown Breakdown, breakdownType string) {
	language := breakdown.Language
	editor := breakdown.Editor
	model := breakdown.Model

	// Set the breakdown dimension based on type
	if breakdownType == "language" && language == "" {
		language = "unknown"
	} else if breakdownType == "editor" && editor == "" {
		editor = "unknown"
	} else if breakdownType == "model" && model == "" {
		model = "unknown"
	}

	if breakdown.SuggestionsCount > 0 {
		ch <- prometheus.MustNewConstMetric(
			c.breakdownSuggestions,
			prometheus.GaugeValue,
			float64(breakdown.SuggestionsCount),
			day, org, language, editor, model,
		)
	}
	if breakdown.AcceptancesCount > 0 {
		ch <- prometheus.MustNewConstMetric(
			c.breakdownAcceptances,
			prometheus.GaugeValue,
			float64(breakdown.AcceptancesCount),
			day, org, language, editor, model,
		)
	}
	if breakdown.LinesSuggested > 0 {
		ch <- prometheus.MustNewConstMetric(
			c.breakdownLinesSuggested,
			prometheus.GaugeValue,
			float64(breakdown.LinesSuggested),
			day, org, language, editor, model,
		)
	}
	if breakdown.LinesAccepted > 0 {
		ch <- prometheus.MustNewConstMetric(
			c.breakdownLinesAccepted,
			prometheus.GaugeValue,
			float64(breakdown.LinesAccepted),
			day, org, language, editor, model,
		)
	}
	if breakdown.ActiveUsers > 0 {
		ch <- prometheus.MustNewConstMetric(
			c.breakdownActiveUsers,
			prometheus.GaugeValue,
			float64(breakdown.ActiveUsers),
			day, org, language, editor, model,
		)
	}
	if breakdown.ChatAcceptances > 0 {
		ch <- prometheus.MustNewConstMetric(
			c.breakdownChatAcceptances,
			prometheus.GaugeValue,
			float64(breakdown.ChatAcceptances),
			day, org, language, editor, model,
		)
	}
	if breakdown.ChatTurns > 0 {
		ch <- prometheus.MustNewConstMetric(
			c.breakdownChatTurns,
			prometheus.GaugeValue,
			float64(breakdown.ChatTurns),
			day, org, language, editor, model,
		)
	}
	if breakdown.ActiveChatUsers > 0 {
		ch <- prometheus.MustNewConstMetric(
			c.breakdownActiveChatUsers,
			prometheus.GaugeValue,
			float64(breakdown.ActiveChatUsers),
			day, org, language, editor, model,
		)
	}
}

func (c *CopilotCollector) fetchMetrics() (CopilotAPIResponse, error) {
	var apiURL string

	if c.enterprise != "" {
		apiURL = fmt.Sprintf("https://api.github.com/enterprises/%s/copilot/metrics", c.enterprise)
	} else if c.team != "" {
		apiURL = fmt.Sprintf("https://api.github.com/orgs/%s/team/%s/copilot/metrics", c.organization, c.team)
	} else {
		apiURL = fmt.Sprintf("https://api.github.com/orgs/%s/copilot/metrics", c.organization)
	}

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.githubToken)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error making request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %w", err)
	}

	var metrics CopilotAPIResponse
	if err := json.Unmarshal(body, &metrics); err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %w", err)
	}

	return metrics, nil
}

func main() {
	githubToken := os.Getenv("GITHUB_TOKEN")
	if githubToken == "" {
		log.Fatal("GITHUB_TOKEN environment variable is required")
	}

	organization := os.Getenv("GITHUB_ORG")
	team := os.Getenv("GITHUB_TEAM")
	enterprise := os.Getenv("GITHUB_ENTERPRISE")

	if organization == "" && enterprise == "" {
		log.Fatal("Either GITHUB_ORG or GITHUB_ENTERPRISE environment variable is required")
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = defaultPort
	}

	collector := NewCopilotCollector(githubToken, organization, team, enterprise)
	prometheus.MustRegister(collector)

	http.Handle(metricsEndpoint, promhttp.Handler())
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintf(w, `<html>
<head><title>GitHub Copilot Metrics Exporter</title></head>
<body>
<h1>GitHub Copilot Metrics Exporter</h1>
<p><a href="%s">Metrics</a></p>
</body>
</html>`, metricsEndpoint)
	})
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "OK")
	})

	log.Printf("Starting GitHub Copilot Metrics Exporter on port %s", port)
	log.Printf("Metrics will be fetched fresh from GitHub API on each scrape")
	log.Printf("Metrics available at http://localhost:%s%s", port, metricsEndpoint)

	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}
