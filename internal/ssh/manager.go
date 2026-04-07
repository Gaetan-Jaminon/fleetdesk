package ssh

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/kevinburke/ssh_config"
	gossh "golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"

	"github.com/Gaetan-Jaminon/fleetdesk/internal/config"
)

// Manager holds persistent SSH connections for all hosts in a fleet.
type Manager struct {
	mu             sync.Mutex
	conns          map[int]*gossh.Client
	cachedPassword string
}

// NewManager creates a new SSH manager.
func NewManager() *Manager {
	return &Manager{
		conns: make(map[int]*gossh.Client),
	}
}

// HostProbeResult is sent when an SSH probe completes for a host.
type HostProbeResult struct {
	Index int
	Info  ProbeInfo
	Err   error
}

// PasswordRetryResult is sent after retrying connection with a password.
type PasswordRetryResult struct {
	Index int
	Info  ProbeInfo
	Err   error
}

// ConnectAndProbe connects to a single host and runs the probe command.
func (sm *Manager) ConnectAndProbe(idx int, h config.Host) HostProbeResult {
	client, err := sm.dial(h)
	if err != nil {
		return HostProbeResult{Index: idx, Err: err}
	}

	sm.mu.Lock()
	sm.conns[idx] = client
	sm.mu.Unlock()

	info, err := Probe(client, h.Entry.SystemdMode, h.ErrorLogSince)
	if err != nil {
		return HostProbeResult{Index: idx, Err: fmt.Errorf("probe: %w", err)}
	}

	return HostProbeResult{Index: idx, Info: info}
}

// RunCommand executes a command on the given host index and returns stdout.
func (sm *Manager) RunCommand(idx int, cmd string) (string, error) {
	sm.mu.Lock()
	client, ok := sm.conns[idx]
	sm.mu.Unlock()

	if !ok || client == nil {
		return "", fmt.Errorf("no connection for host")
	}

	session, err := client.NewSession()
	if err != nil {
		return "", fmt.Errorf("new session: %w", err)
	}
	defer session.Close()

	out, err := session.CombinedOutput(cmd)
	if err != nil {
		return string(out), err
	}
	return string(out), nil
}

// SetCachedPassword stores a password temporarily for batch retries.
func (sm *Manager) SetCachedPassword(password string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.cachedPassword = password
}

// ClearPassword zeroes out the cached password.
func (sm *Manager) ClearPassword() {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.cachedPassword = ""
}

// RetryWithCachedPassword uses the temporarily cached password to connect a host.
func (sm *Manager) RetryWithCachedPassword(idx int, h config.Host) PasswordRetryResult {
	sm.mu.Lock()
	pw := sm.cachedPassword
	sm.mu.Unlock()

	if pw == "" {
		return PasswordRetryResult{Index: idx, Err: fmt.Errorf("no cached password")}
	}

	return sm.ConnectWithPassword(idx, h, pw)
}

// ConnectWithPassword connects to a host using password auth and probes it.
func (sm *Manager) ConnectWithPassword(idx int, h config.Host, password string) PasswordRetryResult {
	entry := h.Entry
	hostname := entry.Hostname
	port := entry.Port
	if port == 0 {
		port = 22
	}

	sshConfig := &gossh.ClientConfig{
		User: entry.User,
		Auth: []gossh.AuthMethod{
			gossh.Password(password),
			gossh.KeyboardInteractive(func(user, instruction string, questions []string, echos []bool) ([]string, error) {
				answers := make([]string, len(questions))
				for i := range questions {
					answers[i] = password
				}
				return answers, nil
			}),
		},
		HostKeyCallback: gossh.InsecureIgnoreHostKey(),
		Timeout:         entry.Timeout,
	}

	addr := fmt.Sprintf("%s:%d", hostname, port)
	client, err := gossh.Dial("tcp", addr, sshConfig)
	if err != nil {
		return PasswordRetryResult{Index: idx, Err: fmt.Errorf("password auth: %w", err)}
	}

	sm.mu.Lock()
	sm.conns[idx] = client
	sm.mu.Unlock()

	info, err := Probe(client, entry.SystemdMode, h.ErrorLogSince)
	if err != nil {
		return PasswordRetryResult{Index: idx, Err: fmt.Errorf("probe: %w", err)}
	}

	return PasswordRetryResult{Index: idx, Info: info}
}

// Close shuts down all SSH connections.
func (sm *Manager) Close() {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	for _, c := range sm.conns {
		if c != nil {
			c.Close()
		}
	}
	sm.conns = make(map[int]*gossh.Client)
	sm.cachedPassword = ""
}

// Probe runs a single SSH command to gather all host info in one roundtrip.
func Probe(client *gossh.Client, systemdMode string, errorLogSince string) (ProbeInfo, error) {
	session, err := client.NewSession()
	if err != nil {
		return ProbeInfo{}, fmt.Errorf("new session: %w", err)
	}
	defer session.Close()

	sysctl := "systemctl"
	if systemdMode == "user" {
		sysctl = "systemctl --user"
	}

	cmd := fmt.Sprintf(
		`hostname -f 2>/dev/null || hostname && `+
			`uptime -s 2>/dev/null || echo unknown && `+
			`(grep PRETTY_NAME /etc/os-release 2>/dev/null | cut -d= -f2 | tr -d '"') || echo unknown && `+
			`%s list-units --type=service --no-pager -q 2>/dev/null | wc -l && `+
			`%s list-units --type=service --state=running --no-pager -q 2>/dev/null | wc -l && `+
			`%s list-units --type=service --state=failed --no-pager -q 2>/dev/null | wc -l && `+
			`podman ps -q 2>/dev/null | wc -l && `+
			`podman ps -a -q 2>/dev/null | wc -l && `+
			`(dnf history list 2>/dev/null | grep -E '\| update ' | grep -v mdatp | head -1 | awk -F'|' '{gsub(/^ +| +$/,"",$3); print $3}' || echo unknown) && `+
			`(dnf history list 2>/dev/null | grep -E '\| update --security' | head -1 | awk -F'|' '{gsub(/^ +| +$/,"",$3); print $3}' || echo unknown) && `+
			`echo $(( $(crontab -l 2>/dev/null | grep -v '^#' | grep -v '^$' | wc -l) + $(ls /etc/cron.d/ 2>/dev/null | wc -l) )) && `+
			`sudo journalctl -p err --since '%s' --no-pager -q 2>/dev/null | wc -l && `+
			`dnf check-update --quiet 2>/dev/null | grep -E '^\S+\.\S+\s' | wc -l && `+
			`df -h --output=pcent -x tmpfs -x devtmpfs 2>/dev/null | tail -n+2 | wc -l && `+
			`df -h --output=pcent -x tmpfs -x devtmpfs 2>/dev/null | tail -n+2 | awk '{gsub(/%%/,""); if ($1 >= 80) print}' | wc -l && `+
			`(getent passwd | awk -F: '$3 >= 1000 && $3 != 65534 {print $1}'; for d in /home/*/; do u=$(basename "$d"); getent passwd "$u" >/dev/null 2>&1 && echo "$u"; done) | sort -u | wc -l && `+
			`((getent passwd | awk -F: '$3 >= 1000 && $3 != 65534 {print $1}'; for d in /home/*/; do u=$(basename "$d"); getent passwd "$u" >/dev/null 2>&1 && echo "$u"; done) | sort -u | while read u; do sudo passwd -S "$u" 2>/dev/null; done | { grep -c ' L ' || true; }) && `+
			`(ip -br link | grep -c UP || echo 0) && `+
			`ip -br link | wc -l && `+
			`(ss -tlnp 2>/dev/null | tail -n +2 | wc -l || echo 0)`,
		sysctl, sysctl, sysctl, errorLogSince,
	)

	out, err := session.CombinedOutput(cmd)
	if err != nil {
		session2, err2 := client.NewSession()
		if err2 != nil {
			return ProbeInfo{}, fmt.Errorf("probe failed: %w", err)
		}
		defer session2.Close()

		if systemdMode == "user" {
			sysctl = "systemctl"
			systemdMode = "system"
		} else {
			sysctl = "systemctl --user"
			systemdMode = "user"
		}

		cmd = fmt.Sprintf(
			`hostname -f 2>/dev/null || hostname && `+
				`uptime -s 2>/dev/null || echo unknown && `+
				`(grep PRETTY_NAME /etc/os-release 2>/dev/null | cut -d= -f2 | tr -d '"') || echo unknown && `+
				`%s list-units --type=service --no-pager -q 2>/dev/null | wc -l && `+
				`echo 0 && `+
				`echo 0 && `+
				`podman ps -q 2>/dev/null | wc -l`,
			sysctl,
		)

		out, err = session2.CombinedOutput(cmd)
		if err != nil {
			return ProbeInfo{}, fmt.Errorf("probe failed both modes: %w", err)
		}
	}

	return ParseProbeOutput(string(out), systemdMode)
}

// dial establishes an SSH connection to a host.
func (sm *Manager) dial(h config.Host) (*gossh.Client, error) {
	entry := h.Entry
	hostname := entry.Hostname
	port := entry.Port
	user := entry.User
	timeout := entry.Timeout

	if user == "" {
		user = ssh_config.Get(hostname, "User")
	}
	if user == "" {
		user = os.Getenv("USER")
	}

	configPort := ssh_config.Get(hostname, "Port")
	if port == 0 && configPort != "" {
		fmt.Sscanf(configPort, "%d", &port)
	}
	if port == 0 {
		port = 22
	}

	if timeout == 0 {
		timeout = 10 * time.Second
	}

	configHost := ssh_config.Get(hostname, "Hostname")
	if configHost != "" {
		hostname = configHost
	}

	var authMethods []gossh.AuthMethod

	agentAuth, agentConn := sshAgentAuth()
	if agentAuth != nil {
		authMethods = append(authMethods, agentAuth)
	}

	identityFile := ssh_config.Get(entry.Hostname, "IdentityFile")
	if identityFile != "" {
		if key := publicKeyFile(ExpandPath(identityFile)); key != nil {
			authMethods = append(authMethods, key)
		}
	}

	home, _ := os.UserHomeDir()
	if home != "" {
		for _, name := range []string{"id_ed25519", "id_rsa", "id_ecdsa"} {
			path := filepath.Join(home, ".ssh", name)
			if key := publicKeyFile(path); key != nil {
				authMethods = append(authMethods, key)
			}
		}
	}

	if len(authMethods) == 0 {
		return nil, fmt.Errorf("no SSH auth methods available")
	}

	addr := fmt.Sprintf("%s:%d", hostname, port)

	var lastErr error
	for _, auth := range authMethods {
		sshConfig := &gossh.ClientConfig{
			User:            user,
			Auth:            []gossh.AuthMethod{auth},
			HostKeyCallback: gossh.InsecureIgnoreHostKey(), // trusted fleet network
			Timeout:         timeout,
		}
		client, err := gossh.Dial("tcp", addr, sshConfig)
		if err == nil {
			return client, nil
		}
		lastErr = err
	}

	// close agent socket on failure — no SSH client to use it
	if agentConn != nil {
		agentConn.Close()
	}

	return nil, fmt.Errorf("dial %s: %w", addr, lastErr)
}

// sshAgentAuth returns an auth method from the SSH agent and the underlying
// connection. The caller must close agentConn when the SSH client is established
// or when all auth attempts fail.
func sshAgentAuth() (gossh.AuthMethod, net.Conn) {
	sock := os.Getenv("SSH_AUTH_SOCK")
	if sock == "" {
		return nil, nil
	}
	conn, err := net.Dial("unix", sock)
	if err != nil {
		return nil, nil
	}
	return gossh.PublicKeysCallback(agent.NewClient(conn).Signers), conn
}

// publicKeyFile returns an auth method from a private key file, or nil.
func publicKeyFile(path string) gossh.AuthMethod {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	signer, err := gossh.ParsePrivateKey(data)
	if err != nil {
		return nil
	}
	return gossh.PublicKeys(signer)
}
