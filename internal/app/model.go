package app

import (
	"fmt"
	"log/slog"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Gaetan-Jaminon/fleetdesk/internal/azure"
	"github.com/Gaetan-Jaminon/fleetdesk/internal/k8s"
	"github.com/Gaetan-Jaminon/fleetdesk/internal/config"
	"github.com/Gaetan-Jaminon/fleetdesk/internal/ssh"
)

// tickMsg triggers a periodic host probe refresh.
type tickMsg time.Time

// azureResourceCountsMsg is sent when Azure resource counts are fetched.
type azureResourceCountsMsg struct {
	counts azure.AzureResourceCounts
	errs   []string
}

type fetchAzureVMsMsg struct {
	vms []azure.VM
	err error
}

type fetchAzureVMDetailMsg struct {
	detail azure.VMDetail
	err    error
}

type azureVMActionMsg struct {
	action string
	vm     string
	err    error
}

// vmTransition tracks an in-flight VM action for optimistic UI.
type vmTransition struct {
	Action      string // "start", "deallocate", "restart"
	Display     string // "starting...", "deallocating...", "restarting...", or real state from poll
	TargetState string // "running", "deallocated", "running"
}

type azureVMPollMsg time.Time

type azureVMPollResultMsg struct {
	states map[string]string // vm name (lowercase) → normalized power state
}

type azureVMTransitionExpireMsg struct {
	vmName string
}

type fetchAzureAKSMsg struct {
	clusters []azure.AKSDetail
	err      error
}

type k8sClusterProbeMsg struct {
	index        int
	contextCount int
	k8sVersion   string
	err          error
}

type fetchK8sContextsMsg struct {
	contexts []k8s.K8sContext
	err      error
}

type k8sResourceCountsMsg struct {
	counts k8s.K8sResourceCounts
	errs   []string
}

type fetchK8sNodesMsg struct {
	nodes []k8s.K8sNode
	err   error
}

type fetchK8sNodeDetailMsg struct {
	detail k8s.K8sNodeDetail
	err    error
}

type fetchK8sNodeUsageMsg struct {
	usage k8s.K8sNodeUsage
	err   error
}

type fetchK8sNodePodsMsg struct {
	pods []k8s.K8sNodePod
	err  error
}

type fetchAzureActivityLogMsg struct {
	entries []azure.ActivityLogEntry
	err     error
}

func (m Model) tickCmd() tea.Cmd {
	interval, err := time.ParseDuration(m.fleets[m.selectedFleet].Defaults.RefreshInterval)
	if err != nil {
		return nil
	}
	return tea.Tick(interval, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

type view int

const (
	viewFleetPicker view = iota
	viewHostList
	viewMetrics
	viewResourcePicker
	viewServiceList
	viewContainerList
	viewCronList
	viewLogLevelPicker
	viewErrorLogList
	viewUpdateList
	viewDiskList
	viewSubscription
	viewAccountList
	viewNetworkPicker
	viewNetworkInterfaces
	viewNetworkPorts
	viewNetworkRoutes
	viewNetworkFirewall
	viewSecurityFailedLogins
	viewSecuritySudo
	viewSecuritySELinux
	viewSecurityAudit
	viewAzureSubList
	viewAzureResourcePicker
	viewAzureVMList
	viewAzureVMDetail
	viewAzureAKSList
	viewAzureAKSDetail
	viewK8sClusterList
	viewK8sContextList
	viewK8sResourcePicker
	viewK8sNodeList
	viewK8sNodeDetail
)

// resourceCount is the number of items in the resource picker (0-indexed).
const resourceCount = 13

// networkSubViewCount is the number of sub-views in the network picker.
const networkSubViewCount = 4

type Model struct {
	view view

	// fleet picker
	fleets      []config.Fleet
	fleetCursor int

	// host list
	selectedFleet int
	hosts         []config.Host
	hostCursor    int

	// azure subscription list
	azureSubs      []azure.AzureSubscriptionItem
	azureSubCursor int

	// azure resource picker
	selectedAzureSub    int
	azureResourceCursor int
	azureResourceCounts azure.AzureResourceCounts
	azureResourceErrors []string
	azureCountsLoaded   bool

	// azure VM list
	azureVMs           []azure.VM
	azureVMCursor      int
	azureVMDetail      azure.VMDetail
	showAzureVMDetail  bool
	azureActivityLog    []azure.ActivityLogEntry
	azureActivityCursor int
	pendingAzureAction  string                    // action name stored for confirm dispatch
	pendingAzureVM      string                    // VM name for confirm dispatch
	pendingAzureRG      string                    // RG name for confirm dispatch
	azureVMTransitions  map[string]vmTransition    // overlay: vm name → in-flight action
	azureVMPolling      bool                       // true if a poll chain is active

	// azure AKS list
	azureAKSClusters []azure.AKSDetail
	azureAKSCursor   int
	azureAKSDetail   azure.AKSDetail

	// kubernetes
	k8s                *k8s.Manager
	k8sClusters        []k8s.K8sClusterItem
	k8sClusterCursor   int
	selectedK8sCluster int
	k8sContexts        []k8s.K8sContext
	k8sContextCursor   int
	selectedK8sContext  string
	k8sResourceCursor  int
	k8sResourceCounts  k8s.K8sResourceCounts
	k8sResourceErrors  []string
	k8sCountsLoaded          bool
	pendingK8sDeleteContext  string
	k8sNodes        []k8s.K8sNode
	k8sNodeCursor   int
	k8sNodeDetail   k8s.K8sNodeDetail
	k8sNodeUsage  *k8s.K8sNodeUsage // nil = loading
	k8sNodePods   []k8s.K8sNodePod // nil = loading
	k8sNodePodCursor int

	// metrics dashboard
	metrics          map[int]config.HostMetrics
	metricsCursor    int
	metricsSortedIdx []int          // sorted host indices for metrics view
	metricErrors     map[int]bool   // hosts where metrics fetch failed

	// resource picker
	selectedHost   int
	resourceCursor int

	// service list
	services           []config.Service
	serviceCursor      int
	showServiceDetail  bool
	serviceDetailUnit  string // unit name being viewed
	serviceStatus      config.ServiceStatus
	serviceLogLines    []string
	serviceLogCursor   int

	// container list
	containers          []config.Container
	containerCursor     int
	showContainerDetail  bool
	containerDetail      config.ContainerDetail
	containerDetailCursor int

	// cron jobs
	cronJobs   []config.CronJob
	cronCursor int

	// log level picker
	logLevels      []config.LogLevelEntry
	logLevelCursor int

	// error logs
	errorLogs        []config.ErrorLog
	errorCursor      int
	selectedLogLevel string

	// updates
	updates            []config.Update
	updateCursor       int
	showUpdateDetail   bool
	updateDetailLines  []string
	updateDetailCursor int

	// disk
	disks            []config.Disk
	diskCursor       int
	showDiskDetail   bool
	diskDetailLines  []string
	diskDetailCursor int

	// subscription
	subscriptions      []config.Subscription
	subscriptionCursor int

	// network
	networkCursor   int
	routeCount      int
	firewallType    string
	firewallCount   int
	interfaces      []config.NetInterface
	interfaceCursor int
	ports           []config.ListeningPort
	portCursor      int
	routes          []config.Route
	routeCursor     int
	dnsNameservers  []string
	dnsSearch       string
	firewallRules   []config.FirewallRule
	firewallCursor  int
	firewallBackend string // "firewalld", "nftables", "iptables", ""

	// security / audit
	failedLogins      []config.FailedLogin
	failedLoginCursor int
	sudoEntries       []config.SudoEntry
	sudoCursor        int
	selinuxDenials    []config.SELinuxDenial
	selinuxCursor     int
	auditEvents       []config.AuditEvent
	auditCursor       int

	// accounts
	accounts           []config.Account
	accountCursor      int
	showAccountDetail    bool
	accountDetailSections []accountDetailSection

	// filter / search
	filterActive bool
	filterText   string

	// column sort
	sortColumn int  // 0 = default sort, 1+ = user-selected column
	sortAsc    bool

	// log detail
	showLogDetail bool

	// confirmation prompt
	showConfirm    bool
	confirmMessage string
	confirmCmd     string
	confirmBanner  string

	// SSH
	ssh   *ssh.Manager
	azure *azure.Manager
	logger *slog.Logger

	// password prompt
	passwordInput      string
	passwordHostIdx    int
	showPasswordPrompt bool

	// flash message
	flash      string
	flashError bool

	// terminal size
	width  int
	height int

	// app info
	version string
}

func NewModel(fleets []config.Fleet, logger *slog.Logger, version string) Model {
	return Model{
		view:    viewFleetPicker,
		fleets:  fleets,
		ssh:   ssh.NewManager(logger),
		azure: azure.NewManager(logger),
		k8s:   k8s.NewManager(logger),
		logger:  logger,
		version: version,
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case ssh.HostProbeResult:
		if msg.Index < len(m.hosts) {
			if msg.Err != nil {
				if ssh.IsAuthError(msg.Err) {
					// mark as needing password
					m.hosts[msg.Index].Status = config.HostUnreachable
					m.hosts[msg.Index].Error = "password required"
					m.hosts[msg.Index].NeedsPassword = true
					// show prompt if not already showing one
					if !m.showPasswordPrompt {
						m.passwordHostIdx = msg.Index
						m.passwordInput = ""
						m.showPasswordPrompt = true
					}
					return m, nil
				}
				m.hosts[msg.Index].Status = config.HostUnreachable
				m.hosts[msg.Index].Error = msg.Err.Error()
			} else {
				m.applyProbeInfo(msg.Index, msg.Info)
			}
		}
		return m, nil

	case azure.SubscriptionProbeResult:
		if msg.Index < len(m.azureSubs) {
			if msg.Err != nil {
				m.azureSubs[msg.Index].Status = azure.SubError
				m.azureSubs[msg.Index].Error = msg.Err.Error()
			} else {
				m.applyAzureProbeInfo(msg.Index, msg.Info)
			}
		}
		return m, nil

	case azureResourceCountsMsg:
		m.azureResourceCounts = msg.counts
		m.azureResourceErrors = msg.errs
		m.azureCountsLoaded = true
		return m, nil

	case fetchAzureVMsMsg:
		if msg.err != nil {
			m.flash = fmt.Sprintf("Failed: %v", msg.err)
			m.flashError = true
		} else {
			m.azureVMs = msg.vms
			m.flash = ""
		}
		return m, nil

	case fetchAzureVMDetailMsg:
		if msg.err != nil {
			m.flash = fmt.Sprintf("Failed: %v", msg.err)
			m.flashError = true
		} else {
			m.azureVMDetail = msg.detail
			// Copy network info from list VM (populated by graph, not az vm show)
			filtered := m.filteredAzureVMs()
			if m.azureVMCursor < len(filtered) {
				listVM := filtered[m.azureVMCursor]
				m.azureVMDetail.VNet = listVM.VNet
				m.azureVMDetail.Subnet = listVM.Subnet
				if m.azureVMDetail.PrivateIP == "" {
					m.azureVMDetail.PrivateIP = listVM.PrivateIP
				}
			}
			m.showAzureVMDetail = true
			m.view = viewAzureVMDetail
			m.flash = ""
		}
		return m, nil

	case azureVMActionMsg:
		if m.azureVMTransitions == nil {
			m.azureVMTransitions = make(map[string]vmTransition)
		}
		if msg.err != nil {
			// Action failed — update overlay to show error, schedule removal
			m.azureVMTransitions[msg.vm] = vmTransition{
				Action:  msg.action,
				Display: msg.action + " failed",
			}
			return m, expireVMTransition(msg.vm)
		}
		// Action succeeded — overlay already set by confirm handler, poll already started
		return m, nil

	case azureVMPollMsg:
		if len(m.azureVMTransitions) == 0 {
			return m, nil // no transitions to track
		}
		return m, m.pollAzureVMStates()

	case azureVMPollResultMsg:
		if m.azureVMTransitions == nil {
			return m, nil
		}
		changed := false
		for name, t := range m.azureVMTransitions {
			if state, ok := msg.states[strings.ToLower(name)]; ok {
				if state == t.TargetState {
					// Target reached — update local list data immediately to avoid flash of old state
					for i := range m.azureVMs {
						if strings.EqualFold(m.azureVMs[i].Name, name) {
							m.azureVMs[i].PowerState = state
							break
						}
					}
					delete(m.azureVMTransitions, name)
					changed = true
				} else if isAzureTransitioningState(state) {
					// Update display with real transitioning state from Azure
					// (starting, stopping, deallocating — but not running, deallocated, stopped)
					t.Display = state
					m.azureVMTransitions[name] = t
				}
			}
		}
		// Refresh list data if any transitions completed
		var cmds []tea.Cmd
		if changed {
			cmds = append(cmds, m.fetchAzureVMs())
		}
		// Continue polling if transitions remain, otherwise stop
		if len(m.azureVMTransitions) > 0 {
			cmds = append(cmds, m.startVMPoll())
		} else {
			m.azureVMPolling = false
		}
		if len(cmds) > 0 {
			return m, tea.Batch(cmds...)
		}
		return m, nil

	case azureVMTransitionExpireMsg:
		if m.azureVMTransitions != nil {
			delete(m.azureVMTransitions, msg.vmName)
			if len(m.azureVMTransitions) == 0 {
				m.azureVMPolling = false
			}
		}
		return m, nil

	case k8sClusterProbeMsg:
		if msg.index < len(m.k8sClusters) {
			if msg.err != nil {
				m.k8sClusters[msg.index].Status = k8s.ClusterError
				m.k8sClusters[msg.index].Error = msg.err.Error()
			} else {
				m.k8sClusters[msg.index].Status = k8s.ClusterOnline
				m.k8sClusters[msg.index].ContextCount = msg.contextCount
				m.k8sClusters[msg.index].K8sVersion = msg.k8sVersion
			}
		}
		return m, nil

	case fetchK8sContextsMsg:
		if msg.err != nil {
			m.flash = fmt.Sprintf("Failed: %v", msg.err)
			m.flashError = true
		} else {
			m.k8sContexts = msg.contexts
			m.flash = ""
		}
		return m, nil

	case k8sResourceCountsMsg:
		m.k8sResourceCounts = msg.counts
		m.k8sResourceErrors = msg.errs
		m.k8sCountsLoaded = true
		return m, nil

	case fetchK8sNodesMsg:
		if msg.err != nil {
			m.flash = fmt.Sprintf("Failed: %v", msg.err)
			m.flashError = true
		} else {
			m.k8sNodes = msg.nodes
			m.flash = ""
		}
		return m, nil

	case fetchK8sNodeDetailMsg:
		if msg.err != nil {
			m.flash = fmt.Sprintf("Failed: %v", msg.err)
			m.flashError = true
		} else {
			m.k8sNodeDetail = msg.detail
			m.view = viewK8sNodeDetail
			m.flash = ""
		}
		return m, nil

	case fetchK8sNodeUsageMsg:
		if msg.err == nil {
			m.k8sNodeUsage = &msg.usage
		}
		return m, nil

	case fetchK8sNodePodsMsg:
		if msg.err == nil {
			m.k8sNodePods = msg.pods
		}
		return m, nil

	case fetchAzureAKSMsg:
		if msg.err != nil {
			m.flash = fmt.Sprintf("Failed: %v", msg.err)
			m.flashError = true
		} else {
			m.azureAKSClusters = msg.clusters
			m.flash = ""
		}
		return m, nil

	case fetchAzureActivityLogMsg:
		if msg.err != nil {
			m.flash = fmt.Sprintf("Activity log: %v", msg.err)
			m.flashError = true
		} else {
			m.azureActivityLog = msg.entries
			m.flash = ""
		}
		return m, nil

	case ssh.PasswordRetryResult:
		if msg.Index < len(m.hosts) {
			if msg.Err != nil {
				m.hosts[msg.Index].Status = config.HostUnreachable
				m.hosts[msg.Index].Error = msg.Err.Error()
			} else {
				m.hosts[msg.Index].NeedsPassword = false
				m.applyProbeInfo(msg.Index, msg.Info)
			}
		}
		// check if more hosts need the same password
		return m, m.retryRemainingPasswordHosts()

	case fetchMetricsMsg:
		if msg.err != nil {
			if m.metricErrors == nil {
				m.metricErrors = make(map[int]bool)
			}
			m.metricErrors[msg.index] = true
		} else {
			if m.metrics == nil {
				m.metrics = make(map[int]config.HostMetrics)
			}
			m.metrics[msg.index] = msg.metrics
		}
		m.flash = ""
		return m, nil

	case fetchServicesMsg:
		if msg.err != nil {
			m.flash = fmt.Sprintf("Failed: %v", msg.err)
			m.flashError = true
		} else {
			if msg.services == nil {
				m.services = []config.Service{}
			} else {
				m.services = msg.services
			}
		}
		return m, nil

	case fetchServiceDetailMsg:
		if msg.err != nil {
			m.flash = fmt.Sprintf("Failed: %v", msg.err)
			m.flashError = true
		} else {
			m.serviceStatus = msg.status
			m.serviceLogLines = msg.logLines
			m.showServiceDetail = true
			m.flash = ""
		}
		return m, nil

	case fetchContainersMsg:
		if msg.err != nil {
			m.flash = fmt.Sprintf("Failed: %v", msg.err)
			m.flashError = true
		} else {
			m.containers = msg.containers
		}
		return m, nil

	case fetchContainerDetailMsg:
		if msg.err != nil {
			m.flash = fmt.Sprintf("Failed: %v", msg.err)
			m.flashError = true
		} else {
			m.containerDetail = msg.detail
			m.showContainerDetail = true
			m.containerDetailCursor = 0
		}
		return m, nil

	case fetchCronMsg:
		if msg.err != nil {
			m.flash = fmt.Sprintf("Failed: %v", msg.err)
			m.flashError = true
		} else {
			m.cronJobs = msg.jobs
		}
		return m, nil

	case fetchLogLevelsMsg:
		if msg.err != nil {
			m.flash = fmt.Sprintf("Failed: %v", msg.err)
			m.flashError = true
		} else {
			m.logLevels = msg.levels
		}
		return m, nil

	case fetchErrorLogsMsg:
		if msg.err != nil {
			m.flash = fmt.Sprintf("Failed: %v", msg.err)
			m.flashError = true
		} else {
			m.errorLogs = msg.logs
		}
		return m, nil

	case fetchUpdatesMsg:
		if msg.err != nil {
			m.flash = fmt.Sprintf("Failed: %v", msg.err)
			m.flashError = true
		} else {
			if msg.updates == nil {
				m.updates = []config.Update{}
			} else {
				m.updates = msg.updates
			}
		}
		return m, nil

	case fetchUpdateDetailMsg:
		if msg.err != nil {
			m.flash = fmt.Sprintf("Failed: %v", msg.err)
			m.flashError = true
		} else {
			m.updateDetailLines = msg.lines
			m.showUpdateDetail = true
			m.updateDetailCursor = 0
		}
		return m, nil

	case fetchSubscriptionMsg:
		if msg.err != nil {
			m.flash = fmt.Sprintf("Failed: %v", msg.err)
			m.flashError = true
		} else {
			m.subscriptions = msg.subs
		}
		return m, nil

	case fetchAccountDetailMsg:
		if msg.err != nil {
			m.flash = fmt.Sprintf("Failed: %v", msg.err)
			m.flashError = true
		} else {
			m.accountDetailSections = msg.sections
			m.showAccountDetail = true
		}
		return m, nil

	case fetchAccountsMsg:
		if msg.err != nil {
			m.flash = fmt.Sprintf("Failed: %v", msg.err)
			m.flashError = true
		} else {
			if msg.accounts == nil {
				m.accounts = []config.Account{}
			} else {
				m.accounts = msg.accounts
			}
		}
		return m, nil

	case fetchPortsMsg:
		if msg.err != nil {
			m.flash = fmt.Sprintf("Failed: %v", msg.err)
			m.flashError = true
		} else {
			if msg.ports == nil {
				m.ports = []config.ListeningPort{}
			} else {
				m.ports = msg.ports
			}
		}
		return m, nil

	case fetchRoutesMsg:
		if msg.err != nil {
			m.flash = fmt.Sprintf("Failed: %v", msg.err)
			m.flashError = true
		} else {
			if msg.routes == nil {
				m.routes = []config.Route{}
			} else {
				m.routes = msg.routes
			}
			m.dnsNameservers = msg.nameservers
			m.dnsSearch = msg.search
		}
		return m, nil

	case fetchInterfacesMsg:
		if msg.err != nil {
			m.flash = fmt.Sprintf("Failed: %v", msg.err)
			m.flashError = true
		} else {
			if msg.interfaces == nil {
				m.interfaces = []config.NetInterface{}
			} else {
				m.interfaces = msg.interfaces
			}
		}
		return m, nil

	case fetchNetworkInfoMsg:
		if msg.err != nil {
			m.flash = fmt.Sprintf("Failed: %v", msg.err)
			m.flashError = true
		} else {
			m.routeCount = msg.routeCount
			m.firewallType = msg.firewallType
			m.firewallCount = msg.firewallCount
		}
		return m, nil

	case fetchFirewallMsg:
		if msg.err != nil {
			m.flash = fmt.Sprintf("Failed: %v", msg.err)
			m.flashError = true
		} else {
			m.firewallRules = msg.rules
			m.firewallBackend = msg.backend
		}
		return m, nil

	case fetchFailedLoginsMsg:
		if msg.err != nil {
			m.flash = fmt.Sprintf("Failed: %v", msg.err)
			m.flashError = true
		} else {
			if msg.logins == nil {
				m.failedLogins = []config.FailedLogin{}
			} else {
				m.failedLogins = msg.logins
			}
		}
		return m, nil

	case fetchSudoActivityMsg:
		if msg.err != nil {
			m.flash = fmt.Sprintf("Failed: %v", msg.err)
			m.flashError = true
		} else {
			if msg.entries == nil {
				m.sudoEntries = []config.SudoEntry{}
			} else {
				m.sudoEntries = msg.entries
			}
		}
		return m, nil

	case fetchSELinuxMsg:
		if msg.err != nil {
			m.flash = fmt.Sprintf("Failed: %v", msg.err)
			m.flashError = true
		} else {
			if msg.denials == nil {
				m.selinuxDenials = []config.SELinuxDenial{}
			} else {
				m.selinuxDenials = msg.denials
			}
		}
		return m, nil

	case fetchAuditSummaryMsg:
		if msg.err != nil {
			m.flash = fmt.Sprintf("Failed: %v", msg.err)
			m.flashError = true
		} else {
			if msg.events == nil {
				m.auditEvents = []config.AuditEvent{}
			} else {
				m.auditEvents = msg.events
			}
		}
		return m, nil

	case fetchDiskMsg:
		if msg.err != nil {
			m.flash = fmt.Sprintf("Failed: %v", msg.err)
			m.flashError = true
		} else {
			m.disks = msg.disks
		}
		return m, nil

	case fetchDiskDetailMsg:
		if msg.err != nil {
			m.flash = fmt.Sprintf("Failed: %v", msg.err)
			m.flashError = true
		} else {
			m.diskDetailLines = msg.lines
			m.showDiskDetail = true
			m.diskDetailCursor = 0
		}
		return m, nil

	case serviceActionMsg:
		if msg.err != nil {
			m.flash = fmt.Sprintf("%s %s failed: %v", msg.action, msg.unit, msg.err)
			m.flashError = true
		} else {
			m.flash = fmt.Sprintf("%s %s: ok", msg.action, msg.unit)
		}
		// refresh service list after action
		return m, m.fetchServices()

	case editFinishedMsg:
		// reload fleets after editor returns
		fleets, err := config.ScanFleets()
		if err != nil {
			m.flash = fmt.Sprintf("Reload failed: %v", err)
			m.flashError = true
		} else {
			m.fleets = fleets
			if m.fleetCursor >= len(m.fleets) {
				m.fleetCursor = max(0, len(m.fleets)-1)
			}
			m.flash = "Reloaded"
		}
		return m, tea.EnterAltScreen

	case sshHandoverFinishedMsg:
		// refresh list after terminal handover returns
		switch m.view {
		case viewServiceList:
			if m.showServiceDetail && m.serviceDetailUnit != "" {
				return m, tea.Batch(tea.EnterAltScreen, m.fetchServiceDetail(m.serviceDetailUnit))
			}
			m.serviceCursor = 0
			m.services = nil
			return m, tea.Batch(tea.EnterAltScreen, m.fetchServices())
		case viewContainerList:
			return m, tea.Batch(tea.EnterAltScreen, m.fetchContainers())
		case viewUpdateList:
			m.updates = nil
			return m, tea.Batch(tea.EnterAltScreen, m.fetchUpdates())
		case viewSubscription:
			m.subscriptions = nil
			return m, tea.Batch(tea.EnterAltScreen, m.fetchSubscription())
		case viewAccountList:
			m.accounts = nil
			return m, tea.Batch(tea.EnterAltScreen, m.fetchAccounts())
		case viewNetworkPicker:
			return m, tea.Batch(tea.EnterAltScreen, m.fetchNetworkInfo())
		}
		return m, tea.EnterAltScreen

	case tickMsg:
		if m.view == viewHostList {
			return m, tea.Batch(m.startProbe(), m.tickCmd())
		}
		if m.view == viewAzureSubList {
			return m, tea.Batch(m.startAzureProbe(), m.tickCmd())
		}
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	return m, nil
}

func (m Model) View() string {
	switch m.view {
	case viewFleetPicker:
		return m.renderFleetPicker()
	case viewHostList:
		return m.renderHostList()
	case viewMetrics:
		return m.renderMetrics()
	case viewResourcePicker:
		return m.renderResourcePicker()
	case viewServiceList:
		return m.renderServiceList()
	case viewContainerList:
		return m.renderContainerList()
	case viewCronList:
		return m.renderCronList()
	case viewLogLevelPicker:
		return m.renderLogLevelPicker()
	case viewErrorLogList:
		return m.renderErrorLogList()
	case viewUpdateList:
		return m.renderUpdateList()
	case viewDiskList:
		return m.renderDiskList()
	case viewAccountList:
		return m.renderAccountList()
	case viewNetworkPicker:
		return m.renderNetworkPicker()
	case viewNetworkInterfaces:
		return m.renderNetworkInterfaces()
	case viewNetworkPorts:
		return m.renderNetworkPorts()
	case viewNetworkRoutes:
		return m.renderNetworkRoutes()
	case viewNetworkFirewall:
		return m.renderNetworkFirewall()
	case viewSubscription:
		return m.renderSubscription()
	case viewSecurityFailedLogins:
		return m.renderFailedLogins()
	case viewSecuritySudo:
		return m.renderSudoActivity()
	case viewSecuritySELinux:
		return m.renderSELinuxDenials()
	case viewSecurityAudit:
		return m.renderAuditSummary()
	case viewAzureSubList:
		return m.renderAzureSubList()
	case viewAzureResourcePicker:
		return m.renderAzureResourcePicker()
	case viewAzureVMList:
		return m.renderAzureVMList()
	case viewAzureVMDetail:
		return m.renderAzureVMDetail()
	case viewAzureAKSList:
		return m.renderAzureAKSList()
	case viewAzureAKSDetail:
		return m.renderAzureAKSDetail()
	case viewK8sClusterList:
		return m.renderK8sClusterList()
	case viewK8sContextList:
		return m.renderK8sContextList()
	case viewK8sResourcePicker:
		return m.renderK8sResourcePicker()
	case viewK8sNodeList:
		return m.renderK8sNodeList()
	case viewK8sNodeDetail:
		return m.renderK8sNodeDetail()
	}
	return ""
}
