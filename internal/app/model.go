package app

import (
	"fmt"
	"log/slog"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Gaetan-Jaminon/fleetdesk/internal/config"
	"github.com/Gaetan-Jaminon/fleetdesk/internal/ssh"
)

// tickMsg triggers a periodic host probe refresh.
type tickMsg time.Time

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
	ssh    *ssh.Manager
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
}

func NewModel(fleets []config.Fleet, logger *slog.Logger) Model {
	return Model{
		view:   viewFleetPicker,
		fleets: fleets,
		ssh:    ssh.NewManager(logger),
		logger: logger,
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
	}
	return ""
}
