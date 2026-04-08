package azure

import (
	"encoding/json"
	"log/slog"
	"strings"
	"sync"
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

// FetchResourceCounts fetches VM, RG, and AKS counts concurrently using goroutines.
// Returns partial results on error — count stays 0 for failed resources, errors collected in []string.
func FetchResourceCounts(m *Manager, subName, tenantID string, logger *slog.Logger) (AzureResourceCounts, []string) {
	start := time.Now()
	logger.Debug("azure resource counts start", "subscription", subName)

	var counts AzureResourceCounts
	var mu sync.Mutex
	var errs []string
	var wg sync.WaitGroup

	type countTask struct {
		name     string
		resource string
		args     []string
		target   *int
	}

	tasks := []countTask{
		{"vms", "vm", []string{"vm", "list", "--subscription", subName, "--query", "length(@)"}, &counts.VMs},
		{"rgs", "group", []string{"group", "list", "--subscription", subName, "--query", "length(@)"}, &counts.RGs},
		{"aks", "aks", []string{"aks", "list", "--subscription", subName, "--query", "length(@)"}, &counts.AKS},
	}

	for _, task := range tasks {
		if tenantID != "" {
			task.args = append(task.args, "--tenant", tenantID)
		}
	}

	wg.Add(len(tasks))
	for _, task := range tasks {
		go func(t countTask) {
			defer wg.Done()
			s := time.Now()
			out, err := m.RunCommand(t.args...)
			elapsed := time.Since(s)
			if err != nil {
				logger.Error("azure count failed", "resource", t.name, "subscription", subName, "elapsed", elapsed, "err", err)
				mu.Lock()
				errs = append(errs, t.name+": "+err.Error())
				mu.Unlock()
				return
			}
			count := ParseJSONCount(out)
			logger.Debug("azure count done", "resource", t.name, "subscription", subName, "count", count, "elapsed", elapsed)
			mu.Lock()
			*t.target = count
			mu.Unlock()
		}(task)
	}

	wg.Wait()
	logger.Debug("azure resource counts complete", "subscription", subName, "vms", counts.VMs, "rgs", counts.RGs, "aks", counts.AKS, "total_elapsed", time.Since(start))
	return counts, errs
}
