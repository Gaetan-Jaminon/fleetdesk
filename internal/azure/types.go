package azure

// VM represents an Azure virtual machine from `az vm list -d`.
type VM struct {
	Name          string
	ResourceGroup string
	Location      string
	VMSize        string
	OSType        string // Linux, Windows
	OSDisk        string // OS distro info (e.g. "RHEL 9", "WindowsServer 2022")
	PrivateIP     string
	PublicIP      string
	PowerState    string // running, deallocated, stopped
	ID            string // full Azure resource ID
}

// ResourceGroup represents an Azure resource group from `az group list`.
type ResourceGroup struct {
	Name     string
	Location string
	State    string // Succeeded, etc.
	ID       string
}

// AKSCluster represents an Azure Kubernetes cluster from `az aks list`.
type AKSCluster struct {
	Name              string
	ResourceGroup     string
	Location          string
	KubernetesVersion string
	NodeCount         int
	PowerState        string // Running, Stopped
	ID                string
}

// AzureSubscription represents an Azure subscription from `az account list`.
// Named AzureSubscription to avoid collision with config.Subscription (RHEL subscription-manager).
type AzureSubscription struct {
	Name      string
	ID        string
	State     string // Enabled, Disabled
	IsDefault bool
}

// SubscriptionStatus represents the probe state of an Azure subscription.
type SubscriptionStatus int

const (
	SubConnecting SubscriptionStatus = iota
	SubOnline
	SubError
)

// SubscriptionProbeInfo holds the result of checking access to an Azure subscription.
type SubscriptionProbeInfo struct {
	ID     string // subscription UUID
	State  string // Enabled, Disabled
	Tenant string // tenantDisplayName
	User   string // user.name
}

// AzureSubscriptionItem is the runtime representation of a subscription with access state.
type AzureSubscriptionItem struct {
	Name     string
	TenantID string
	Status   SubscriptionStatus
	Error    string
	ID       string // subscription UUID
	State    string // Enabled, Disabled
	Tenant   string // tenantDisplayName
	User     string // user.name
}
