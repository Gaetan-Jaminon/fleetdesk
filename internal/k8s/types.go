package k8s

// ClusterStatus represents the connectivity state of a K8s cluster.
type ClusterStatus int

const (
	ClusterChecking ClusterStatus = iota
	ClusterOnline
	ClusterError
)

// K8sClusterItem is the runtime representation of a cluster from config.
type K8sClusterItem struct {
	Name         string
	Status       ClusterStatus
	K8sVersion   string
	Error        string
	ContextCount int // number of matching kubectl contexts
}

// K8sContext represents a kubectl context matching a cluster.
type K8sContext struct {
	Name      string // context name
	Cluster   string // cluster name
	User      string // auth info
	Namespace string // default namespace
	Current   bool   // true if this is the active context
}

// K8sResourceCounts holds resource counts for a cluster context.
type K8sResourceCounts struct {
	Namespaces int
	Nodes      int
	ArgoApps   int
}

// ClusterProbeResult is sent when a cluster connectivity check completes.
type ClusterProbeResult struct {
	Index        int
	ContextCount int
	Err          error
}
