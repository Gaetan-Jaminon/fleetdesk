package azure

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"
)

// ParseJSONCount parses a bare JSON number (e.g. from --query "length(@)").
// Returns 0 for empty, nil, or invalid input.
func ParseJSONCount(data []byte) int {
	s := strings.TrimSpace(string(data))
	if s == "" || s == "null" {
		return 0
	}
	var n int
	if err := json.Unmarshal([]byte(s), &n); err != nil {
		return 0
	}
	return n
}

// FetchResourceCounts fetches VM, RG, and AKS counts using a single Resource Graph query.
// Falls back to 3 separate az CLI calls if subID is empty or graph fails.
func FetchResourceCounts(m *Manager, subName, subID, tenantID string, logger *slog.Logger) (AzureResourceCounts, []string) {
	start := time.Now()
	logger.Debug("azure resource counts start", "subscription", subName)

	if subID != "" {
		counts, err := fetchCountsGraph(m, subName, subID, logger, start)
		if err == nil {
			return counts, nil
		}
		logger.Error("azure graph count failed, falling back", "err", err)
	}

	return fetchCountsLegacy(m, subName, tenantID, logger, start)
}

// fetchCountsGraph uses a single Resource Graph query to count VMs, RGs, and AKS clusters.
func fetchCountsGraph(m *Manager, subName, subID string, logger *slog.Logger, start time.Time) (AzureResourceCounts, error) {
	query := "Resources " +
		"| where type in~ ('microsoft.compute/virtualmachines', 'microsoft.containerservice/managedclusters') " +
		"| union (ResourceContainers | where type =~ 'microsoft.resources/subscriptions/resourcegroups') " +
		"| summarize count() by type"

	args := []string{"graph", "query", "-q", query, "--subscriptions", subID, "--first", "1000"}
	data, err := m.RunCommand(args...)
	if err != nil {
		return AzureResourceCounts{}, fmt.Errorf("graph count query: %w", err)
	}

	var result struct {
		Data []struct {
			Type  string `json:"type"`
			Count int    `json:"count_"`
		} `json:"data"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return AzureResourceCounts{}, fmt.Errorf("parsing graph count result: %w", err)
	}

	var counts AzureResourceCounts
	for _, r := range result.Data {
		switch strings.ToLower(r.Type) {
		case "microsoft.compute/virtualmachines":
			counts.VMs = r.Count
		case "microsoft.resources/subscriptions/resourcegroups":
			counts.RGs = r.Count
		case "microsoft.containerservice/managedclusters":
			counts.AKS = r.Count
		}
	}

	logger.Debug("azure resource counts complete (graph)", "subscription", subName,
		"vms", counts.VMs, "rgs", counts.RGs, "aks", counts.AKS,
		"total_elapsed", time.Since(start))
	return counts, nil
}

// fetchCountsLegacy uses 3 separate az CLI calls with goroutines (slow fallback).
func fetchCountsLegacy(m *Manager, subName, tenantID string, logger *slog.Logger, start time.Time) (AzureResourceCounts, []string) {
	logger.Debug("azure resource counts legacy (3 az calls)", "subscription", subName)

	type result struct {
		name  string
		count int
		err   error
	}

	ch := make(chan result, 3)

	commands := []struct {
		name string
		args []string
	}{
		{"vms", []string{"vm", "list", "--subscription", subName, "--query", "length(@)"}},
		{"rgs", []string{"group", "list", "--subscription", subName, "--query", "length(@)"}},
		{"aks", []string{"aks", "list", "--subscription", subName, "--query", "length(@)"}},
	}

	for _, cmd := range commands {
		args := cmd.args
		if tenantID != "" {
			args = append(args, "--tenant", tenantID)
		}
		name := cmd.name
		go func() {
			s := time.Now()
			out, err := m.RunCommand(args...)
			elapsed := time.Since(s)
			if err != nil {
				logger.Error("azure count failed", "resource", name, "subscription", subName, "elapsed", elapsed, "err", err)
				ch <- result{name: name, err: err}
				return
			}
			count := ParseJSONCount(out)
			logger.Debug("azure count done", "resource", name, "subscription", subName, "count", count, "elapsed", elapsed)
			ch <- result{name: name, count: count}
		}()
	}

	var counts AzureResourceCounts
	var errs []string
	for range 3 {
		r := <-ch
		if r.err != nil {
			errs = append(errs, r.name+": "+r.err.Error())
			continue
		}
		switch r.name {
		case "vms":
			counts.VMs = r.count
		case "rgs":
			counts.RGs = r.count
		case "aks":
			counts.AKS = r.count
		}
	}

	logger.Debug("azure resource counts complete (legacy)", "subscription", subName,
		"vms", counts.VMs, "rgs", counts.RGs, "aks", counts.AKS,
		"total_elapsed", time.Since(start))
	return counts, errs
}
