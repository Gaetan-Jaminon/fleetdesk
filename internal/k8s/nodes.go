package k8s

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// FetchNodes fetches all nodes for a kubectl context.
func FetchNodes(m *Manager, contextName string, logger *slog.Logger) ([]K8sNode, error) {
	start := time.Now()
	logger.Debug("k8s fetch nodes start", "context", contextName)

	var nodes []K8sNode
	var topData map[string]TopNodeData
	var nodeErr, topErr error
	var wg sync.WaitGroup

	wg.Add(2)

	// Goroutine 1: get nodes
	go func() {
		defer wg.Done()
		data, err := m.RunCommand("get", "nodes", "--context", contextName, "-o", "json")
		if err != nil {
			nodeErr = fmt.Errorf("get nodes: %w", err)
			return
		}
		nodes, nodeErr = ParseNodes(data)
	}()

	// Goroutine 2: top nodes
	go func() {
		defer wg.Done()
		topData, topErr = FetchTopNodes(m, contextName, logger)
	}()

	wg.Wait()

	if nodeErr != nil {
		return nil, nodeErr
	}

	// Merge top data (best-effort)
	if topErr != nil {
		logger.Error("k8s top nodes failed", "err", topErr)
	} else if topData != nil {
		for i := range nodes {
			if td, ok := topData[nodes[i].Name]; ok {
				nodes[i].CPUUsage = td.CPUUsage
				nodes[i].CPUPct = td.CPUPct
				nodes[i].MemUsage = td.MemUsage
				nodes[i].MemPct = td.MemPct
			}
		}
	}

	// Sort by pool name then node name for consistent grouping
	sort.Slice(nodes, func(i, j int) bool {
		if nodes[i].Pool != nodes[j].Pool {
			return nodes[i].Pool < nodes[j].Pool
		}
		return nodes[i].Name < nodes[j].Name
	})

	logger.Debug("k8s fetch nodes complete", "context", contextName, "count", len(nodes), "elapsed", time.Since(start))
	return nodes, nil
}

// FetchTopNodes fetches CPU/memory usage for all nodes via kubectl top nodes.
func FetchTopNodes(m *Manager, contextName string, logger *slog.Logger) (map[string]TopNodeData, error) {
	start := time.Now()
	logger.Debug("k8s fetch top nodes start", "context", contextName)

	out, err := m.RunCommand("top", "nodes", "--context", contextName, "--no-headers")
	if err != nil {
		return nil, fmt.Errorf("top nodes: %w", err)
	}

	result := ParseTopNodes(string(out))
	logger.Debug("k8s fetch top nodes complete", "context", contextName, "count", len(result), "elapsed", time.Since(start))
	return result, nil
}

// TopNodeData holds usage data from kubectl top nodes.
type TopNodeData struct {
	CPUUsage string
	CPUPct   string
	MemUsage string
	MemPct   string
}

// ParseTopNodes parses `kubectl top nodes --no-headers` output.
// Returns a map of node name -> usage data.
func ParseTopNodes(output string) map[string]TopNodeData {
	result := make(map[string]TopNodeData)
	for _, line := range strings.Split(strings.TrimSpace(output), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 5 {
			continue
		}
		result[fields[0]] = TopNodeData{
			CPUUsage: fields[1],
			CPUPct:   fields[2],
			MemUsage: fields[3],
			MemPct:   fields[4],
		}
	}
	return result
}

// FetchNodeDetail fetches extended properties for a single node.
func FetchNodeDetail(m *Manager, contextName, nodeName string, logger *slog.Logger) (K8sNodeDetail, error) {
	start := time.Now()
	logger.Debug("k8s fetch node detail start", "context", contextName, "node", nodeName)

	data, err := m.RunCommand("get", "node", nodeName, "--context", contextName, "-o", "json")
	if err != nil {
		return K8sNodeDetail{}, fmt.Errorf("get node: %w", err)
	}
	detail, err := ParseNodeDetail(data)
	if err != nil {
		return K8sNodeDetail{}, err
	}

	logger.Debug("k8s fetch node detail complete", "node", nodeName, "elapsed", time.Since(start))
	return detail, nil
}

// ParseNodes parses kubectl get nodes JSON output.
func ParseNodes(data []byte) ([]K8sNode, error) {
	var list struct {
		Items []json.RawMessage `json:"items"`
	}
	if err := json.Unmarshal(data, &list); err != nil {
		return nil, fmt.Errorf("parsing node list: %w", err)
	}

	nodes := make([]K8sNode, 0, len(list.Items))
	for _, item := range list.Items {
		node, err := parseNodeItem(item)
		if err != nil {
			continue
		}
		nodes = append(nodes, node)
	}
	return nodes, nil
}

// ParseNodeDetail parses a single node JSON object.
func ParseNodeDetail(data []byte) (K8sNodeDetail, error) {
	node, err := parseNodeItem(data)
	if err != nil {
		return K8sNodeDetail{}, err
	}

	var raw struct {
		Metadata struct {
			Labels            map[string]string `json:"labels"`
			CreationTimestamp string            `json:"creationTimestamp"`
		} `json:"metadata"`
		Spec struct {
			PodCIDR       string `json:"podCIDR"`
			Unschedulable bool   `json:"unschedulable"`
			Taints        []struct {
				Key    string `json:"key"`
				Value  string `json:"value"`
				Effect string `json:"effect"`
			} `json:"taints"`
		} `json:"spec"`
		Status struct {
			Conditions []struct {
				Type   string `json:"type"`
				Status string `json:"status"`
			} `json:"conditions"`
			Addresses []struct {
				Type    string `json:"type"`
				Address string `json:"address"`
			} `json:"addresses"`
			NodeInfo struct {
				ContainerRuntimeVersion string `json:"containerRuntimeVersion"`
				KernelVersion           string `json:"kernelVersion"`
				OSImage                 string `json:"osImage"`
			} `json:"nodeInfo"`
			Allocatable struct {
				CPU    string `json:"cpu"`
				Memory string `json:"memory"`
				Pods   string `json:"pods"`
			} `json:"allocatable"`
			Images []json.RawMessage `json:"images"`
		} `json:"status"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return K8sNodeDetail{}, fmt.Errorf("parsing node detail: %w", err)
	}

	conditions := make([]K8sCondition, len(raw.Status.Conditions))
	for i, c := range raw.Status.Conditions {
		conditions[i] = K8sCondition{Type: c.Type, Status: c.Status}
	}

	taints := make([]K8sTaint, len(raw.Spec.Taints))
	for i, t := range raw.Spec.Taints {
		taints[i] = K8sTaint{Key: t.Key, Value: t.Value, Effect: t.Effect}
	}

	// Extract internal IP
	internalIP := ""
	for _, addr := range raw.Status.Addresses {
		if addr.Type == "InternalIP" {
			internalIP = addr.Address
			break
		}
	}

	return K8sNodeDetail{
		K8sNode:           node,
		InternalIP:        internalIP,
		PodCIDR:           raw.Spec.PodCIDR,
		Unschedulable:     raw.Spec.Unschedulable,
		ContainerRuntime:  raw.Status.NodeInfo.ContainerRuntimeVersion,
		KernelVersion:     raw.Status.NodeInfo.KernelVersion,
		OSImage:           raw.Status.NodeInfo.OSImage,
		Created:           raw.Metadata.CreationTimestamp,
		AllocatableCPU:    raw.Status.Allocatable.CPU,
		AllocatableMemory: formatMemory(raw.Status.Allocatable.Memory),
		AllocatablePods:   raw.Status.Allocatable.Pods,
		ImageCount:        len(raw.Status.Images),
		Conditions:        conditions,
		Taints:            taints,
		Labels:            raw.Metadata.Labels,
	}, nil
}

func parseNodeItem(data []byte) (K8sNode, error) {
	var raw struct {
		Metadata struct {
			Name              string            `json:"name"`
			Labels            map[string]string `json:"labels"`
			CreationTimestamp string            `json:"creationTimestamp"`
		} `json:"metadata"`
		Spec struct {
			Taints []json.RawMessage `json:"taints"`
		} `json:"spec"`
		Status struct {
			Conditions []struct {
				Type   string `json:"type"`
				Status string `json:"status"`
			} `json:"conditions"`
			NodeInfo struct {
				KubeletVersion string `json:"kubeletVersion"`
			} `json:"nodeInfo"`
			Capacity struct {
				CPU    string `json:"cpu"`
				Memory string `json:"memory"`
				Pods   string `json:"pods"`
			} `json:"capacity"`
			Allocatable struct {
				CPU string `json:"cpu"`
			} `json:"allocatable"`
		} `json:"status"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return K8sNode{}, fmt.Errorf("parsing node: %w", err)
	}

	// Find Ready condition
	status := "Unknown"
	for _, c := range raw.Status.Conditions {
		if c.Type == "Ready" {
			if c.Status == "True" {
				status = "Ready"
			} else {
				status = "NotReady"
			}
			break
		}
	}

	os := raw.Metadata.Labels["kubernetes.io/os"]
	arch := raw.Metadata.Labels["kubernetes.io/arch"]
	osStr := ""
	if os != "" {
		osStr = os + "/" + arch
	}

	return K8sNode{
		Name:    raw.Metadata.Name,
		Pool:    raw.Metadata.Labels["agentpool"],
		Status:  status,
		Version: raw.Status.NodeInfo.KubeletVersion,
		CPU:     raw.Status.Capacity.CPU,
		Memory:  formatMemory(raw.Status.Capacity.Memory),
		Pods:    raw.Status.Capacity.Pods,
		OS:      osStr,
		VMSize:  raw.Metadata.Labels["node.kubernetes.io/instance-type"],
		Taints:  len(raw.Spec.Taints),
		Age:     formatAge(raw.Metadata.CreationTimestamp),
		CPUA:    raw.Status.Allocatable.CPU,
	}, nil
}

// formatMemory converts Ki memory to human-readable format.
func formatMemory(mem string) string {
	mem = strings.TrimSuffix(mem, "Ki")
	ki, err := strconv.ParseInt(mem, 10, 64)
	if err != nil {
		return mem
	}
	gi := ki / (1024 * 1024)
	if gi > 0 {
		return fmt.Sprintf("%dGi", gi)
	}
	mi := ki / 1024
	return fmt.Sprintf("%dMi", mi)
}

// FetchNodeUsage fetches CPU/Memory usage for a node via kubectl top.
func FetchNodeUsage(m *Manager, contextName, nodeName string, logger *slog.Logger) (K8sNodeUsage, error) {
	start := time.Now()
	logger.Debug("k8s fetch node usage start", "node", nodeName)

	out, err := m.RunCommand("top", "node", nodeName, "--context", contextName, "--no-headers")
	if err != nil {
		return K8sNodeUsage{}, fmt.Errorf("top node: %w", err)
	}

	usage := parseTopNode(string(out))
	logger.Debug("k8s fetch node usage complete", "node", nodeName, "elapsed", time.Since(start))
	return usage, nil
}

// parseTopNode parses `kubectl top node <name> --no-headers` output.
// Format: NAME  CPU(cores)  CPU%  MEMORY(bytes)  MEMORY%
func parseTopNode(output string) K8sNodeUsage {
	fields := strings.Fields(strings.TrimSpace(output))
	if len(fields) < 5 {
		return K8sNodeUsage{}
	}
	return K8sNodeUsage{
		CPUUsage:   fields[1],
		CPUPercent: fields[2],
		MemUsage:   fields[3],
		MemPercent: fields[4],
	}
}

// FetchNodePods fetches pods running on a node with their resource requests/limits.
func FetchNodePods(m *Manager, contextName, nodeName string, logger *slog.Logger) ([]K8sNodePod, error) {
	start := time.Now()
	logger.Debug("k8s fetch node pods start", "node", nodeName)

	out, err := m.RunCommand("get", "pods", "--all-namespaces",
		"--field-selector", "spec.nodeName="+nodeName,
		"--context", contextName, "-o", "json")
	if err != nil {
		return nil, fmt.Errorf("get pods: %w", err)
	}

	pods, err := ParseNodePods(out)
	if err != nil {
		return nil, err
	}

	logger.Debug("k8s fetch node pods complete", "node", nodeName, "count", len(pods), "elapsed", time.Since(start))
	return pods, nil
}

// ParseNodePods parses kubectl get pods JSON into K8sNodePod slice.
func ParseNodePods(data []byte) ([]K8sNodePod, error) {
	var list struct {
		Items []struct {
			Metadata struct {
				Namespace         string `json:"namespace"`
				Name              string `json:"name"`
				CreationTimestamp string `json:"creationTimestamp"`
			} `json:"metadata"`
			Spec struct {
				Containers []struct {
					Resources struct {
						Requests map[string]string `json:"requests"`
						Limits   map[string]string `json:"limits"`
					} `json:"resources"`
				} `json:"containers"`
			} `json:"spec"`
			Status struct {
				Phase             string `json:"phase"`
				ContainerStatuses []struct {
					Ready bool `json:"ready"`
				} `json:"containerStatuses"`
			} `json:"status"`
		} `json:"items"`
	}
	if err := json.Unmarshal(data, &list); err != nil {
		return nil, fmt.Errorf("parsing pods: %w", err)
	}

	pods := make([]K8sNodePod, 0, len(list.Items))
	for _, item := range list.Items {
		// Sum resources across all containers
		var cpuReq, cpuLim, memReq, memLim int64
		for _, c := range item.Spec.Containers {
			cpuReq += parseMilliCPU(c.Resources.Requests["cpu"])
			cpuLim += parseMilliCPU(c.Resources.Limits["cpu"])
			memReq += parseMemBytes(c.Resources.Requests["memory"])
			memLim += parseMemBytes(c.Resources.Limits["memory"])
		}

		// Ready count
		readyCount := 0
		for _, cs := range item.Status.ContainerStatuses {
			if cs.Ready {
				readyCount++
			}
		}
		totalContainers := len(item.Spec.Containers)
		ready := fmt.Sprintf("%d/%d", readyCount, totalContainers)

		pods = append(pods, K8sNodePod{
			Namespace: item.Metadata.Namespace,
			Name:      item.Metadata.Name,
			Status:    item.Status.Phase,
			Ready:     ready,
			CPUReq:    formatMilliCPU(cpuReq),
			CPULim:    formatMilliCPU(cpuLim),
			MemReq:    formatMemBytes(memReq),
			MemLim:    formatMemBytes(memLim),
			Age:       formatAge(item.Metadata.CreationTimestamp),
		})
	}
	return pods, nil
}

// parseMilliCPU converts CPU string to millicores (e.g. "100m" → 100, "0.5" → 500).
func parseMilliCPU(s string) int64 {
	if s == "" {
		return 0
	}
	if strings.HasSuffix(s, "m") {
		v, _ := strconv.ParseInt(strings.TrimSuffix(s, "m"), 10, 64)
		return v
	}
	v, _ := strconv.ParseFloat(s, 64)
	return int64(v * 1000)
}

// formatMilliCPU formats millicores for display.
func formatMilliCPU(m int64) string {
	if m == 0 {
		return "0"
	}
	if m >= 1000 && m%1000 == 0 {
		return fmt.Sprintf("%d", m/1000)
	}
	return fmt.Sprintf("%dm", m)
}

// parseMemBytes converts memory string to bytes (e.g. "428Mi" → bytes, "1Gi" → bytes).
func parseMemBytes(s string) int64 {
	if s == "" {
		return 0
	}
	s = strings.TrimSpace(s)
	multipliers := map[string]int64{
		"Ki": 1024,
		"Mi": 1024 * 1024,
		"Gi": 1024 * 1024 * 1024,
		"Ti": 1024 * 1024 * 1024 * 1024,
	}
	for suffix, mult := range multipliers {
		if strings.HasSuffix(s, suffix) {
			v, _ := strconv.ParseInt(strings.TrimSuffix(s, suffix), 10, 64)
			return v * mult
		}
	}
	v, _ := strconv.ParseInt(s, 10, 64)
	return v
}

// formatMemBytes formats bytes for display.
func formatMemBytes(b int64) string {
	if b == 0 {
		return "0"
	}
	gi := b / (1024 * 1024 * 1024)
	if gi > 0 {
		return fmt.Sprintf("%dGi", gi)
	}
	mi := b / (1024 * 1024)
	if mi > 0 {
		return fmt.Sprintf("%dMi", mi)
	}
	ki := b / 1024
	return fmt.Sprintf("%dKi", ki)
}

// formatAge converts a creation timestamp to a human-readable age.
func formatAge(timestamp string) string {
	t, err := time.Parse(time.RFC3339, timestamp)
	if err != nil {
		return "—"
	}
	d := time.Since(t)
	if d.Hours() >= 24 {
		return fmt.Sprintf("%dd", int(d.Hours()/24))
	}
	if d.Hours() >= 1 {
		return fmt.Sprintf("%dh", int(d.Hours()))
	}
	return fmt.Sprintf("%dm", int(d.Minutes()))
}
