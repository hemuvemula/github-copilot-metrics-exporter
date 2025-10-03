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
	defaultPort     = "9101"
	metricsEndpoint = "/metrics"
)

type CopilotMetrics struct {
	TotalSuggestions     int `json:"total_suggestions_count"`
	TotalAcceptances     int `json:"total_acceptances_count"`
	TotalLinesSuggested  int `json:"total_lines_suggested"`
	TotalLinesAccepted   int `json:"total_lines_accepted"`
	TotalActiveUsers     int `json:"total_active_users"`
	TotalChatAcceptances int `json:"total_chat_acceptances"`
	TotalChatTurns       int `json:"total_chat_turns"`
	TotalActiveChat      int `json:"total_active_chat_users"`
}

type CopilotAPIResponse []struct {
	Day                  string `json:"day"`
	TotalSuggestions     int    `json:"total_suggestions_count"`
	TotalAcceptances     int    `json:"total_acceptances_count"`
	TotalLinesSuggested  int    `json:"total_lines_suggested"`
	TotalLinesAccepted   int    `json:"total_lines_accepted"`
	TotalActiveUsers     int    `json:"total_active_users"`
	TotalChatAcceptances int    `json:"total_chat_acceptances"`
	TotalChatTurns       int    `json:"total_chat_turns"`
	TotalActiveChat      int    `json:"total_active_chat_users"`
}

type CopilotCollector struct {
	githubToken          string
	organization         string
	team                 string
	enterprise           string
	totalSuggestions     *prometheus.Desc
	totalAcceptances     *prometheus.Desc
	totalLinesSuggested  *prometheus.Desc
	totalLinesAccepted   *prometheus.Desc
	totalActiveUsers     *prometheus.Desc
	totalChatAcceptances *prometheus.Desc
	totalChatTurns       *prometheus.Desc
	totalActiveChatUsers *prometheus.Desc
	acceptanceRate       *prometheus.Desc
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
}

func (c *CopilotCollector) Collect(ch chan<- prometheus.Metric) {
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

		ch <- prometheus.MustNewConstMetric(
			c.totalSuggestions,
			prometheus.GaugeValue,
			float64(metric.TotalSuggestions),
			day, org,
		)
		ch <- prometheus.MustNewConstMetric(
			c.totalAcceptances,
			prometheus.GaugeValue,
			float64(metric.TotalAcceptances),
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
			float64(metric.TotalActiveChat),
			day, org,
		)

		// Calculate acceptance rate
		acceptanceRate := 0.0
		if metric.TotalSuggestions > 0 {
			acceptanceRate = float64(metric.TotalAcceptances) / float64(metric.TotalSuggestions)
		}
		ch <- prometheus.MustNewConstMetric(
			c.acceptanceRate,
			prometheus.GaugeValue,
			acceptanceRate,
			day, org,
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
	log.Printf("Metrics available at http://localhost:%s%s", port, metricsEndpoint)

	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}
