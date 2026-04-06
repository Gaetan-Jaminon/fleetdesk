package main

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/kevinburke/ssh_config"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"

	issh "github.com/Gaetan-Jaminon/fleetdesk/internal/ssh"
)

// sshManager holds persistent SSH connections for all hosts in a fleet.
type sshManager struct {
	mu             sync.Mutex
	conns          map[int]*ssh.Client // keyed by host index
	cachedPassword string              // temporary, cleared after batch retry
}

func newSSHManager() *sshManager {
	return &sshManager{
		conns: make(map[int]*ssh.Client),
	}
}

// connectAll opens SSH connections to all hosts in parallel.
// It sends a hostProbeResult for each host as it completes.
func (sm *sshManager) connectAll(hosts []host, results chan<- hostProbeResult) {
	var wg sync.WaitGroup
	for i, h := range hosts {
		wg.Add(1)
		go func(idx int, h host) {
			defer wg.Done()
			result := sm.connectAndProbe(idx, h)
			results <- result
		}(i, h)
	}
	go func() {
		wg.Wait()
		close(results)
	}()
}

// connectAndProbe connects to a single host and runs the probe command.
func (sm *sshManager) connectAndProbe(idx int, h host) hostProbeResult {
	client, err := sm.dial(h)
	if err != nil {
		return hostProbeResult{
			index: idx,
			err:   err,
		}
	}

	sm.mu.Lock()
	sm.conns[idx] = client
	sm.mu.Unlock()

	// run probe command
	info, err := probe(client, h.Entry.SystemdMode, h.ErrorLogSince)
	if err != nil {
		return hostProbeResult{
			index: idx,
			err:   fmt.Errorf("probe: %w", err),
		}
	}

	return hostProbeResult{
		index: idx,
		info:  info,
	}
}

// dial establishes an SSH connection to a host using the auth resolution order:
// 1. SSH agent
// 2. ~/.ssh/config IdentityFile
// 3. Default key paths
func (sm *sshManager) dial(h host) (*ssh.Client, error) {
	entry := h.Entry
	hostname := entry.Hostname
	port := entry.Port
	user := entry.User
	timeout := entry.Timeout

	// resolve from SSH config if not set
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

	// resolve actual hostname from SSH config
	configHost := ssh_config.Get(hostname, "Hostname")
	if configHost != "" {
		hostname = configHost
	}

	// collect individual auth methods — each will be tried in its own connection
	// to avoid "too many authentication failures" from servers with low MaxAuthTries
	var authMethods []ssh.AuthMethod

	// 1. SSH agent
	if agentAuth := sshAgentAuth(); agentAuth != nil {
		authMethods = append(authMethods, agentAuth)
	}

	// 2. Identity files from SSH config
	identityFile := ssh_config.Get(entry.Hostname, "IdentityFile")
	if identityFile != "" {
		if key := publicKeyFile(expandPath(identityFile)); key != nil {
			authMethods = append(authMethods, key)
		}
	}

	// 3. Default key paths
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

	// try each auth method individually to avoid MaxAuthTries exhaustion
	var lastErr error
	for _, auth := range authMethods {
		config := &ssh.ClientConfig{
			User:            user,
			Auth:            []ssh.AuthMethod{auth},
			HostKeyCallback: ssh.InsecureIgnoreHostKey(),
			Timeout:         timeout,
		}
		client, err := ssh.Dial("tcp", addr, config)
		if err == nil {
			return client, nil
		}
		lastErr = err
	}

	return nil, fmt.Errorf("dial %s: %w", addr, lastErr)
}

// runCommand executes a command on the given host index and returns stdout.
func (sm *sshManager) runCommand(idx int, cmd string) (string, error) {
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

// setCachedPassword stores a password temporarily for batch retries.
func (sm *sshManager) setCachedPassword(password string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.cachedPassword = password
}

// clearPassword zeroes out the cached password.
func (sm *sshManager) clearPassword() {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.cachedPassword = ""
}

// retryWithCachedPassword uses the temporarily cached password to connect a host.
func (sm *sshManager) retryWithCachedPassword(idx int, h host) passwordRetryResult {
	sm.mu.Lock()
	pw := sm.cachedPassword
	sm.mu.Unlock()

	if pw == "" {
		return passwordRetryResult{index: idx, err: fmt.Errorf("no cached password")}
	}

	return sm.connectWithPassword(idx, h, pw)
}

// connectWithPassword connects to a host using password auth and probes it.
func (sm *sshManager) connectWithPassword(idx int, h host, password string) passwordRetryResult {
	entry := h.Entry
	hostname := entry.Hostname
	port := entry.Port
	if port == 0 {
		port = 22
	}

	config := &ssh.ClientConfig{
		User: entry.User,
		Auth: []ssh.AuthMethod{
			ssh.Password(password),
			ssh.KeyboardInteractive(func(user, instruction string, questions []string, echos []bool) ([]string, error) {
				answers := make([]string, len(questions))
				for i := range questions {
					answers[i] = password
				}
				return answers, nil
			}),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         entry.Timeout,
	}

	addr := fmt.Sprintf("%s:%d", hostname, port)
	client, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		return passwordRetryResult{index: idx, err: fmt.Errorf("password auth: %w", err)}
	}

	sm.mu.Lock()
	sm.conns[idx] = client
	sm.mu.Unlock()

	info, err := probe(client, entry.SystemdMode, h.ErrorLogSince)
	if err != nil {
		return passwordRetryResult{index: idx, err: fmt.Errorf("probe: %w", err)}
	}

	return passwordRetryResult{index: idx, info: info}
}

// close shuts down all SSH connections.
func (sm *sshManager) close() {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	for _, c := range sm.conns {
		if c != nil {
			c.Close()
		}
	}
	sm.conns = make(map[int]*ssh.Client)
	sm.cachedPassword = ""
}

// sshAgentAuth returns an auth method from the SSH agent, or nil.
func sshAgentAuth() ssh.AuthMethod {
	sock := os.Getenv("SSH_AUTH_SOCK")
	if sock == "" {
		return nil
	}
	conn, err := net.Dial("unix", sock)
	if err != nil {
		return nil
	}
	return ssh.PublicKeysCallback(agent.NewClient(conn).Signers)
}

// publicKeyFile returns an auth method from a private key file, or nil.
func publicKeyFile(path string) ssh.AuthMethod {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	signer, err := ssh.ParsePrivateKey(data)
	if err != nil {
		// try without passphrase only
		return nil
	}
	return ssh.PublicKeys(signer)
}

// expandPath delegates to internal/ssh.ExpandPath.
var expandPath = issh.ExpandPath
