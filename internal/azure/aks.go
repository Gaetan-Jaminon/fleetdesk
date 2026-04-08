package azure

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"time"
)

// FetchAKSClusters fetches AKS clusters for a subscription using Resource Graph.
func FetchAKSClusters(m *Manager, subName, subID, tenantID string, logger *slog.Logger) ([]AKSDetail, error) {
	start := time.Now()
	logger.Debug("azure fetch aks start", "subscription", subName)

	if subID == "" {
		return nil, fmt.Errorf("subscription ID required for graph query")
	}

	query := "Resources | where type =~ 'microsoft.containerservice/managedclusters' " +
		"| project name, resourceGroup, location, id, " +
		"k8sVersion = properties.kubernetesVersion, " +
		"powerState = properties.powerState.code, " +
		"networkPlugin = properties.networkProfile.networkPlugin, " +
		"pools = properties.agentPoolProfiles"

	args := []string{"graph", "query", "-q", query, "--subscriptions", subID, "--first", "1000"}
	data, err := m.RunCommand(args...)
	if err != nil {
		return nil, fmt.Errorf("graph aks query: %w", err)
	}

	clusters, err := ParseGraphAKS(data)
	if err != nil {
		return nil, err
	}

	logger.Debug("azure fetch aks complete", "subscription", subName, "count", len(clusters), "elapsed", time.Since(start))
	return clusters, nil
}

// ParseGraphAKS parses Resource Graph output for AKS clusters with embedded node pools.
func ParseGraphAKS(data []byte) ([]AKSDetail, error) {
	var result struct {
		Data []struct {
			Name          string `json:"name"`
			ResourceGroup string `json:"resourceGroup"`
			Location      string `json:"location"`
			ID            string `json:"id"`
			K8sVersion    string `json:"k8sVersion"`
			PowerState    string `json:"powerState"`
			NetworkPlugin string `json:"networkPlugin"`
			Pools         []struct {
				Name                       string `json:"name"`
				Mode                       string `json:"mode"`
				VMSize                     string `json:"vmSize"`
				Count                      int    `json:"count"`
				MinCount                   int    `json:"minCount"`
				MaxCount                   int    `json:"maxCount"`
				CurrentOrchestratorVersion string `json:"currentOrchestratorVersion"`
				EnableAutoScaling          bool   `json:"enableAutoScaling"`
			} `json:"pools"`
		} `json:"data"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("parsing graph aks result: %w", err)
	}

	clusters := make([]AKSDetail, len(result.Data))
	for i, r := range result.Data {
		totalNodes := 0
		pools := make([]AKSNodePool, len(r.Pools))
		for j, p := range r.Pools {
			totalNodes += p.Count
			pools[j] = AKSNodePool{
				Name:      p.Name,
				Mode:      p.Mode,
				VMSize:    p.VMSize,
				Count:     p.Count,
				MinCount:  p.MinCount,
				MaxCount:  p.MaxCount,
				Version:   p.CurrentOrchestratorVersion,
				AutoScale: p.EnableAutoScaling,
			}
		}
		clusters[i] = AKSDetail{
			AKSCluster: AKSCluster{
				Name:              r.Name,
				ResourceGroup:     r.ResourceGroup,
				Location:          r.Location,
				KubernetesVersion: r.K8sVersion,
				NodeCount:         totalNodes,
				PowerState:        r.PowerState,
				ID:                r.ID,
			},
			NetworkPlugin: r.NetworkPlugin,
			Pools:         pools,
		}
	}
	return clusters, nil
}
