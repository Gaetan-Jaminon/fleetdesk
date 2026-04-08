package k8s

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"
)

// MatchContexts returns kubectl contexts whose cluster field matches clusterName.
func MatchContexts(m *Manager, clusterName string, logger *slog.Logger) ([]K8sContext, error) {
	start := time.Now()
	logger.Debug("k8s match contexts start", "cluster", clusterName)

	out, err := m.RunCommand("config", "get-contexts", "--no-headers")
	if err != nil {
		return nil, fmt.Errorf("get-contexts: %w", err)
	}

	var contexts []K8sContext
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if line == "" {
			continue
		}
		ctx := parseContextLine(line)
		if strings.EqualFold(ctx.Cluster, clusterName) {
			contexts = append(contexts, ctx)
		}
	}

	logger.Debug("k8s match contexts complete", "cluster", clusterName, "count", len(contexts), "elapsed", time.Since(start))
	return contexts, nil
}

// CountContexts returns the number of contexts matching a cluster name.
func CountContexts(m *Manager, clusterName string) int {
	out, err := m.RunCommand("config", "get-contexts", "--no-headers")
	if err != nil {
		return 0
	}
	count := 0
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if line == "" {
			continue
		}
		ctx := parseContextLine(line)
		if strings.EqualFold(ctx.Cluster, clusterName) {
			count++
		}
	}
	return count
}

// parseContextLine parses a single line from `kubectl config get-contexts --no-headers`.
// Format: CURRENT  NAME  CLUSTER  AUTHINFO  NAMESPACE
func parseContextLine(line string) K8sContext {
	fields := strings.Fields(line)
	if len(fields) == 0 {
		return K8sContext{}
	}

	current := false
	offset := 0
	if fields[0] == "*" {
		current = true
		offset = 1
	}

	get := func(idx int) string {
		i := idx + offset
		if i < len(fields) {
			return fields[i]
		}
		return ""
	}

	return K8sContext{
		Name:      get(0),
		Cluster:   get(1),
		User:      get(2),
		Namespace: get(3),
		Current:   current,
	}
}

// CheckCluster verifies connectivity and returns the server K8s version.
func CheckCluster(m *Manager, contextName string, logger *slog.Logger) (string, error) {
	start := time.Now()
	logger.Debug("k8s cluster check start", "context", contextName)

	out, err := m.RunCommand("version", "--context", contextName, "-o", "json")
	if err != nil {
		logger.Error("k8s cluster check failed", "context", contextName, "elapsed", time.Since(start), "err", err)
		return "", err
	}

	version := parseServerVersion(out)
	logger.Debug("k8s cluster check complete", "context", contextName, "version", version, "elapsed", time.Since(start))
	return version, nil
}

// parseServerVersion extracts serverVersion.gitVersion from kubectl version JSON.
func parseServerVersion(data []byte) string {
	var v struct {
		ServerVersion struct {
			GitVersion string `json:"gitVersion"`
		} `json:"serverVersion"`
	}
	if err := json.Unmarshal(data, &v); err != nil {
		return ""
	}
	return v.ServerVersion.GitVersion
}
