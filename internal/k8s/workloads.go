package k8s

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"time"
)

// FetchNamespaces fetches namespaces with resource counts (parallel).
// FetchNamespaces fetches namespace list (fast, ~200ms).
func FetchNamespaces(m *Manager, contextName string, logger *slog.Logger) ([]K8sNamespace, error) {
	start := time.Now()
	logger.Debug("k8s fetch namespaces start", "context", contextName)

	data, err := m.RunCommand("get", "namespaces", "--context", contextName, "-o", "json")
	if err != nil {
		return nil, fmt.Errorf("get namespaces: %w", err)
	}
	namespaces, err := ParseNamespaces(data)
	if err != nil {
		return nil, err
	}

	sort.Slice(namespaces, func(i, j int) bool {
		return namespaces[i].Name < namespaces[j].Name
	})

	logger.Debug("k8s fetch namespaces complete", "context", contextName, "count", len(namespaces), "elapsed", time.Since(start))
	return namespaces, nil
}

// FetchNamespaceCounts fetches resource counts per namespace (slow, ~4-6s).
// Returns map[namespace][4]int = [pods, deployments, statefulsets, daemonsets].
func FetchNamespaceCounts(m *Manager, contextName string, logger *slog.Logger) (map[string][4]int, error) {
	start := time.Now()
	logger.Debug("k8s fetch namespace counts start", "context", contextName)

	data, err := m.RunCommand("get", "pods,deployments,statefulsets,daemonsets", "-A", "--context", contextName, "-o", "json")
	if err != nil {
		return nil, fmt.Errorf("get resource counts: %w", err)
	}
	counts, err := parseResourceCounts(data)
	if err != nil {
		return nil, err
	}

	logger.Debug("k8s fetch namespace counts complete", "context", contextName, "elapsed", time.Since(start))
	return counts, nil
}

// ParseNamespaces parses kubectl get namespaces JSON.
func ParseNamespaces(data []byte) ([]K8sNamespace, error) {
	var list struct {
		Items []struct {
			Metadata struct {
				Name              string `json:"name"`
				CreationTimestamp string `json:"creationTimestamp"`
			} `json:"metadata"`
			Status struct {
				Phase string `json:"phase"`
			} `json:"status"`
		} `json:"items"`
	}
	if err := json.Unmarshal(data, &list); err != nil {
		return nil, fmt.Errorf("parsing namespaces: %w", err)
	}

	ns := make([]K8sNamespace, len(list.Items))
	for i, item := range list.Items {
		ns[i] = K8sNamespace{
			Name:   item.Metadata.Name,
			Status: item.Status.Phase,
			Age:    formatAge(item.Metadata.CreationTimestamp),
		}
	}
	return ns, nil
}

// parseResourceCounts parses a multi-type kubectl get output and counts per namespace.
// Returns map[namespace][4]int = [pods, deployments, statefulsets, daemonsets].
func parseResourceCounts(data []byte) (map[string][4]int, error) {
	var list struct {
		Items []struct {
			Kind     string `json:"kind"`
			Metadata struct {
				Namespace string `json:"namespace"`
			} `json:"metadata"`
		} `json:"items"`
	}
	if err := json.Unmarshal(data, &list); err != nil {
		return nil, fmt.Errorf("parsing resource counts: %w", err)
	}

	counts := make(map[string][4]int)
	for _, item := range list.Items {
		ns := item.Metadata.Namespace
		c := counts[ns]
		switch item.Kind {
		case "Pod":
			c[0]++
		case "Deployment":
			c[1]++
		case "StatefulSet":
			c[2]++
		case "DaemonSet":
			c[3]++
		}
		counts[ns] = c
	}
	return counts, nil
}

// FetchWorkloads fetches deployments, statefulsets, and daemonsets in a namespace.
func FetchWorkloads(m *Manager, contextName, namespace string, logger *slog.Logger) ([]K8sWorkload, error) {
	start := time.Now()
	logger.Debug("k8s fetch workloads start", "context", contextName, "namespace", namespace)

	data, err := m.RunCommand("get", "deployments,statefulsets,daemonsets", "-n", namespace, "--context", contextName, "-o", "json")
	if err != nil {
		return nil, fmt.Errorf("get workloads: %w", err)
	}

	workloads, err := ParseWorkloads(data)
	if err != nil {
		return nil, err
	}

	// Sort by kind then name
	kindOrder := map[string]int{"Deployment": 0, "StatefulSet": 1, "DaemonSet": 2}
	sort.Slice(workloads, func(i, j int) bool {
		oi, oj := kindOrder[workloads[i].Kind], kindOrder[workloads[j].Kind]
		if oi != oj {
			return oi < oj
		}
		return workloads[i].Name < workloads[j].Name
	})

	logger.Debug("k8s fetch workloads complete", "context", contextName, "namespace", namespace, "count", len(workloads), "elapsed", time.Since(start))
	return workloads, nil
}

// ParseWorkloads parses a multi-type kubectl get output for workloads.
func ParseWorkloads(data []byte) ([]K8sWorkload, error) {
	var list struct {
		Items []json.RawMessage `json:"items"`
	}
	if err := json.Unmarshal(data, &list); err != nil {
		return nil, fmt.Errorf("parsing workloads: %w", err)
	}

	var workloads []K8sWorkload
	for _, raw := range list.Items {
		var meta struct {
			Kind     string `json:"kind"`
			Metadata struct {
				Name              string `json:"name"`
				CreationTimestamp string `json:"creationTimestamp"`
			} `json:"metadata"`
			Spec struct {
				Replicas               *int `json:"replicas"`
				DesiredNumberScheduled int  `json:"desiredNumberScheduled"`
			} `json:"spec"`
			Status struct {
				ReadyReplicas     int `json:"readyReplicas"`
				UpdatedReplicas   int `json:"updatedReplicas"`
				AvailableReplicas int `json:"availableReplicas"`
				NumberReady       int `json:"numberReady"`
			} `json:"status"`
		}
		if err := json.Unmarshal(raw, &meta); err != nil {
			continue
		}

		w := K8sWorkload{
			Kind: meta.Kind,
			Name: meta.Metadata.Name,
			Age:  formatAge(meta.Metadata.CreationTimestamp),
		}

		switch meta.Kind {
		case "Deployment":
			replicas := 0
			if meta.Spec.Replicas != nil {
				replicas = *meta.Spec.Replicas
			}
			w.Ready = fmt.Sprintf("%d/%d", meta.Status.ReadyReplicas, replicas)
			w.UpToDate = meta.Status.UpdatedReplicas
			w.Available = meta.Status.AvailableReplicas
		case "StatefulSet":
			replicas := 0
			if meta.Spec.Replicas != nil {
				replicas = *meta.Spec.Replicas
			}
			w.Ready = fmt.Sprintf("%d/%d", meta.Status.ReadyReplicas, replicas)
		case "DaemonSet":
			w.Ready = fmt.Sprintf("%d/%d", meta.Status.NumberReady, meta.Spec.DesiredNumberScheduled)
		}

		workloads = append(workloads, w)
	}
	return workloads, nil
}

// FetchPods fetches pods in a namespace, optionally filtered by workload name prefix.
func FetchPods(m *Manager, contextName, namespace, workloadPrefix string, logger *slog.Logger) ([]K8sPod, error) {
	start := time.Now()
	logger.Debug("k8s fetch pods start", "context", contextName, "namespace", namespace, "workload", workloadPrefix)

	data, err := m.RunCommand("get", "pods", "-n", namespace, "--context", contextName, "-o", "json")
	if err != nil {
		return nil, fmt.Errorf("get pods: %w", err)
	}

	allPods, err := ParsePods(data)
	if err != nil {
		return nil, err
	}

	// Filter by workload name prefix if specified
	var pods []K8sPod
	if workloadPrefix == "" {
		pods = allPods
	} else {
		prefix := workloadPrefix + "-"
		for _, p := range allPods {
			if strings.HasPrefix(p.Name, prefix) {
				pods = append(pods, p)
			}
		}
	}

	logger.Debug("k8s fetch pods complete", "context", contextName, "namespace", namespace, "total", len(allPods), "filtered", len(pods), "elapsed", time.Since(start))
	return pods, nil
}

// ParsePods parses kubectl get pods JSON.
func ParsePods(data []byte) ([]K8sPod, error) {
	var list struct {
		Items []struct {
			Metadata struct {
				Name              string `json:"name"`
				Namespace         string `json:"namespace"`
				CreationTimestamp string `json:"creationTimestamp"`
			} `json:"metadata"`
			Spec struct {
				NodeName   string   `json:"nodeName"`
				Containers []struct{} `json:"containers"`
			} `json:"spec"`
			Status struct {
				Phase             string `json:"phase"`
				PodIP             string `json:"podIP"`
				ContainerStatuses []struct {
					Ready        bool `json:"ready"`
					RestartCount int  `json:"restartCount"`
				} `json:"containerStatuses"`
			} `json:"status"`
		} `json:"items"`
	}
	if err := json.Unmarshal(data, &list); err != nil {
		return nil, fmt.Errorf("parsing pods: %w", err)
	}

	pods := make([]K8sPod, 0, len(list.Items))
	for _, item := range list.Items {
		readyCount := 0
		restarts := 0
		for _, cs := range item.Status.ContainerStatuses {
			if cs.Ready {
				readyCount++
			}
			restarts += cs.RestartCount
		}
		totalContainers := len(item.Spec.Containers)

		pods = append(pods, K8sPod{
			Name:      item.Metadata.Name,
			Namespace: item.Metadata.Namespace,
			Status:    item.Status.Phase,
			Ready:     fmt.Sprintf("%d/%d", readyCount, totalContainers),
			Restarts:  restarts,
			Node:      item.Spec.NodeName,
			IP:        item.Status.PodIP,
			Age:       formatAge(item.Metadata.CreationTimestamp),
		})
	}
	return pods, nil
}

// FetchPodDetail fetches extended pod properties.
func FetchPodDetail(m *Manager, contextName, namespace, podName string, logger *slog.Logger) (K8sPodDetail, error) {
	start := time.Now()
	logger.Debug("k8s fetch pod detail start", "pod", podName, "namespace", namespace)

	data, err := m.RunCommand("get", "pod", podName, "-n", namespace, "--context", contextName, "-o", "json")
	if err != nil {
		return K8sPodDetail{}, fmt.Errorf("get pod: %w", err)
	}

	detail, err := ParsePodDetail(data)
	if err != nil {
		return K8sPodDetail{}, err
	}

	logger.Debug("k8s fetch pod detail complete", "pod", podName, "elapsed", time.Since(start))
	return detail, nil
}

// ParsePodDetail parses a single pod JSON into K8sPodDetail.
func ParsePodDetail(data []byte) (K8sPodDetail, error) {
	var raw struct {
		Metadata struct {
			Name              string `json:"name"`
			Namespace         string `json:"namespace"`
			CreationTimestamp string `json:"creationTimestamp"`
		} `json:"metadata"`
		Spec struct {
			NodeName   string `json:"nodeName"`
			Containers []struct {
				Name      string `json:"name"`
				Image     string `json:"image"`
				Resources struct {
					Requests map[string]string `json:"requests"`
					Limits   map[string]string `json:"limits"`
				} `json:"resources"`
			} `json:"containers"`
		} `json:"spec"`
		Status struct {
			Phase             string `json:"phase"`
			PodIP             string `json:"podIP"`
			ContainerStatuses []struct {
				Name         string                 `json:"name"`
				Ready        bool                   `json:"ready"`
				RestartCount int                    `json:"restartCount"`
				State        map[string]interface{} `json:"state"`
			} `json:"containerStatuses"`
			Conditions []struct {
				Type   string `json:"type"`
				Status string `json:"status"`
			} `json:"conditions"`
		} `json:"status"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return K8sPodDetail{}, fmt.Errorf("parsing pod detail: %w", err)
	}

	// Build pod base
	readyCount := 0
	totalRestarts := 0
	for _, cs := range raw.Status.ContainerStatuses {
		if cs.Ready {
			readyCount++
		}
		totalRestarts += cs.RestartCount
	}

	pod := K8sPod{
		Name:      raw.Metadata.Name,
		Namespace: raw.Metadata.Namespace,
		Status:    raw.Status.Phase,
		Ready:     fmt.Sprintf("%d/%d", readyCount, len(raw.Spec.Containers)),
		Restarts:  totalRestarts,
		Node:      raw.Spec.NodeName,
		IP:        raw.Status.PodIP,
		Age:       formatAge(raw.Metadata.CreationTimestamp),
	}

	// Build containers
	containers := make([]K8sContainer, len(raw.Spec.Containers))
	statusMap := make(map[string]struct {
		ready    bool
		restarts int
		state    string
	})
	for _, cs := range raw.Status.ContainerStatuses {
		state := "unknown"
		for k := range cs.State {
			state = k
			break
		}
		statusMap[cs.Name] = struct {
			ready    bool
			restarts int
			state    string
		}{cs.Ready, cs.RestartCount, state}
	}

	for i, c := range raw.Spec.Containers {
		containers[i] = K8sContainer{
			Name:   c.Name,
			Image:  c.Image,
			CPUReq: c.Resources.Requests["cpu"],
			CPULim: c.Resources.Limits["cpu"],
			MemReq: c.Resources.Requests["memory"],
			MemLim: c.Resources.Limits["memory"],
		}
		if s, ok := statusMap[c.Name]; ok {
			containers[i].Ready = s.ready
			containers[i].Restarts = s.restarts
			containers[i].State = s.state
		}
	}

	// Build conditions
	conditions := make([]K8sCondition, len(raw.Status.Conditions))
	for i, c := range raw.Status.Conditions {
		conditions[i] = K8sCondition{Type: c.Type, Status: c.Status}
	}

	return K8sPodDetail{
		K8sPod:     pod,
		Containers: containers,
		Conditions: conditions,
	}, nil
}
