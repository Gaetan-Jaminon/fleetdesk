package azure

import (
	"fmt"
	"log/slog"
	"time"
)

// SubscriptionProbeResult is sent when an Azure subscription access check completes.
type SubscriptionProbeResult struct {
	Index int
	Info  SubscriptionProbeInfo
	Err   error
}

// CountVMStates counts VMs by power state.
func CountVMStates(vms []VM) (total, running, stopped, deallocated int) {
	total = len(vms)
	for _, vm := range vms {
		switch vm.PowerState {
		case "running":
			running++
		case "stopped":
			stopped++
		case "deallocated":
			deallocated++
		}
	}
	return
}

// SumAKSNodes returns the cluster count and total node count.
func SumAKSNodes(clusters []AKSCluster) (count, nodes int) {
	count = len(clusters)
	for _, c := range clusters {
		nodes += c.NodeCount
	}
	return
}

// CheckSubscriptionAccess verifies the user has access to a subscription
// by running `az account show --subscription <name>`.
func CheckSubscriptionAccess(m *Manager, subscriptionName, tenantID string, logger *slog.Logger) (SubscriptionProbeInfo, error) {
	start := time.Now()
	logger.Debug("azure access check start", "subscription", subscriptionName)

	args := []string{"account", "show", "--subscription", subscriptionName}
	if tenantID != "" {
		args = append(args, "--tenant", tenantID)
	}
	data, err := m.RunCommand(args...)
	if err != nil {
		return SubscriptionProbeInfo{}, fmt.Errorf("account show: %w", err)
	}

	info, err := ParseAccountShow(data)
	if err != nil {
		return SubscriptionProbeInfo{}, err
	}

	logger.Debug("azure access check complete",
		"subscription", subscriptionName,
		"id", info.ID,
		"state", info.State,
		"tenant", info.Tenant,
		"user", info.User,
		"elapsed", time.Since(start))
	return info, nil
}
