package k8s

import "testing"

func TestParseNamespaces(t *testing.T) {
	input := []byte(`{"items":[
		{"metadata":{"name":"default","creationTimestamp":"2026-01-01T00:00:00Z"},"status":{"phase":"Active"}},
		{"metadata":{"name":"kube-system","creationTimestamp":"2026-01-01T00:00:00Z"},"status":{"phase":"Active"}},
		{"metadata":{"name":"terminating-ns","creationTimestamp":"2026-01-01T00:00:00Z"},"status":{"phase":"Terminating"}}
	]}`)
	ns, err := ParseNamespaces(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ns) != 3 {
		t.Fatalf("got %d namespaces, want 3", len(ns))
	}
	if ns[0].Name != "default" {
		t.Errorf("ns[0].Name = %q", ns[0].Name)
	}
	if ns[2].Status != "Terminating" {
		t.Errorf("ns[2].Status = %q", ns[2].Status)
	}
}

func TestParseWorkloads(t *testing.T) {
	input := []byte(`{"items":[
		{"kind":"Deployment","metadata":{"name":"web","creationTimestamp":"2026-01-01T00:00:00Z"},"spec":{"replicas":3},"status":{"readyReplicas":3,"updatedReplicas":3,"availableReplicas":3}},
		{"kind":"StatefulSet","metadata":{"name":"redis","creationTimestamp":"2026-01-01T00:00:00Z"},"spec":{"replicas":1},"status":{"readyReplicas":1}},
		{"kind":"DaemonSet","metadata":{"name":"node-exporter","creationTimestamp":"2026-01-01T00:00:00Z"},"spec":{"desiredNumberScheduled":5},"status":{"numberReady":5}}
	]}`)
	workloads, err := ParseWorkloads(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(workloads) != 3 {
		t.Fatalf("got %d workloads, want 3", len(workloads))
	}
	if workloads[0].Ready != "3/3" {
		t.Errorf("deployment Ready = %q, want 3/3", workloads[0].Ready)
	}
	if workloads[0].UpToDate != 3 {
		t.Errorf("deployment UpToDate = %d, want 3", workloads[0].UpToDate)
	}
	if workloads[1].Ready != "1/1" {
		t.Errorf("statefulset Ready = %q, want 1/1", workloads[1].Ready)
	}
	if workloads[2].Ready != "5/5" {
		t.Errorf("daemonset Ready = %q, want 5/5", workloads[2].Ready)
	}
}

func TestParsePods(t *testing.T) {
	input := []byte(`{"items":[
		{"metadata":{"name":"web-abc123","namespace":"default","creationTimestamp":"2026-04-08T00:00:00Z"},"spec":{"nodeName":"node-1","containers":[{},{}]},"status":{"phase":"Running","podIP":"10.0.0.1","containerStatuses":[{"ready":true,"restartCount":0},{"ready":true,"restartCount":1}]}},
		{"metadata":{"name":"web-def456","namespace":"default","creationTimestamp":"2026-04-08T00:00:00Z"},"spec":{"nodeName":"node-2","containers":[{}]},"status":{"phase":"Pending","podIP":"","containerStatuses":[]}}
	]}`)
	pods, err := ParsePods(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(pods) != 2 {
		t.Fatalf("got %d pods, want 2", len(pods))
	}
	if pods[0].Ready != "2/2" {
		t.Errorf("pods[0].Ready = %q, want 2/2", pods[0].Ready)
	}
	if pods[0].Restarts != 1 {
		t.Errorf("pods[0].Restarts = %d, want 1", pods[0].Restarts)
	}
	if pods[1].Status != "Pending" {
		t.Errorf("pods[1].Status = %q", pods[1].Status)
	}
}

func TestParsePodDetail(t *testing.T) {
	input := []byte(`{
		"metadata":{"name":"web-abc123","namespace":"default","creationTimestamp":"2026-04-08T00:00:00Z"},
		"spec":{"nodeName":"node-1","containers":[
			{"name":"web","image":"nginx:1.25","resources":{"requests":{"cpu":"100m","memory":"128Mi"},"limits":{"cpu":"200m","memory":"256Mi"}}}
		]},
		"status":{
			"phase":"Running","podIP":"10.0.0.1",
			"containerStatuses":[{"name":"web","ready":true,"restartCount":2,"state":{"running":{}}}],
			"conditions":[{"type":"Ready","status":"True"},{"type":"PodScheduled","status":"True"}]
		}
	}`)
	detail, err := ParsePodDetail(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if detail.Name != "web-abc123" {
		t.Errorf("Name = %q", detail.Name)
	}
	if len(detail.Containers) != 1 {
		t.Fatalf("Containers = %d, want 1", len(detail.Containers))
	}
	if detail.Containers[0].Image != "nginx:1.25" {
		t.Errorf("Container.Image = %q", detail.Containers[0].Image)
	}
	if detail.Containers[0].State != "running" {
		t.Errorf("Container.State = %q", detail.Containers[0].State)
	}
	if detail.Containers[0].CPUReq != "100m" {
		t.Errorf("Container.CPUReq = %q", detail.Containers[0].CPUReq)
	}
	if len(detail.Conditions) != 2 {
		t.Fatalf("Conditions = %d, want 2", len(detail.Conditions))
	}
}

func TestParseResourceCounts(t *testing.T) {
	input := []byte(`{"items":[
		{"kind":"Pod","metadata":{"namespace":"default"}},
		{"kind":"Pod","metadata":{"namespace":"default"}},
		{"kind":"Deployment","metadata":{"namespace":"default"}},
		{"kind":"Pod","metadata":{"namespace":"kube-system"}},
		{"kind":"DaemonSet","metadata":{"namespace":"kube-system"}}
	]}`)
	counts, err := parseResourceCounts(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if counts["default"][0] != 2 {
		t.Errorf("default pods = %d, want 2", counts["default"][0])
	}
	if counts["default"][1] != 1 {
		t.Errorf("default deploys = %d, want 1", counts["default"][1])
	}
	if counts["kube-system"][3] != 1 {
		t.Errorf("kube-system ds = %d, want 1", counts["kube-system"][3])
	}
}
