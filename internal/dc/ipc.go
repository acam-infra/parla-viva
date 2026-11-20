package dc

import (
	"bufio"
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Caller is the minimal JSON-RPC invocation interface implemented by both
// *Transport (direct stdio deltachat-rpc-server) and *IPCClient (localhost bridge).
type Caller interface {
	Call(ctx context.Context, method string, params any) (json.RawMessage, error)
}

const ipcStateFilename = ".parla-viva-ipc.json"

// IPCState is written to <accountsDir>/.parla-viva-ipc.json (0600) while a GUI
// or `parla-viva serve` instance owns deltachat-rpc-server and accounts.lock.
type IPCState struct {
	Addr         string `json:"addr"`
	Token        string `json:"token"`
	PID          int    `json:"pid"`
	AccountsPath string `json:"accountsPath"`
	StartedAt    string `json:"startedAt"`
}

// IPCStatePath returns the lock-adjacent IPC descriptor path for accountsPath.
func IPCStatePath(accountsPath string) string {
	if accountsPath == "" {
		accountsPath = DefaultAccountsDir()
	}
	return filepath.Join(accountsPath, ipcStateFilename)
}

// IPCServer exposes an authenticated loopback HTTP bridge for a live Transport
// so Agent CLI commands can share the active deltachat-rpc-server without
// colliding on accounts.lock.
type IPCServer struct {
	caller       Caller
	accountsPath string
	statePath    string
	token        string
	listener     net.Listener
	server       *http.Server

	subMu sync.Mutex
	subs  map[chan Event]struct{}
}

// StartIPCServer binds an ephemeral 127.0.0.1 port, writes the descriptor file,
// and starts serving /health, /rpc, and /events.
func StartIPCServer(caller Caller, accountsPath string) (*IPCServer, error) {
	if caller == nil {
		return nil, errors.New("dc: ipc server requires non-nil caller")
	}
	if accountsPath == "" {
		accountsPath = DefaultAccountsDir()
	}
	_ = os.MkdirAll(accountsPath, 0o700)

	var tokBytes [24]byte
	if _, err := rand.Read(tokBytes[:]); err != nil {
		return nil, fmt.Errorf("dc: ipc token: %w", err)
	}
	token := hex.EncodeToString(tokBytes[:])

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("dc: ipc listen: %w", err)
	}

	s := &IPCServer{
		caller:       caller,
		accountsPath: accountsPath,
		statePath:    IPCStatePath(accountsPath),
		token:        token,
		listener:     ln,
		subs:         make(map[chan Event]struct{}),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/rpc", s.handleRPC)
	mux.HandleFunc("/events", s.handleEvents)
	s.server = &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}

	state := IPCState{
		Addr:         ln.Addr().String(),
		Token:        token,
		PID:          os.Getpid(),
		AccountsPath: accountsPath,
		StartedAt:    time.Now().UTC().Format(time.RFC3339),
	}
	raw, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		_ = ln.Close()
		return nil, err
	}
	if err := os.WriteFile(s.statePath, raw, 0o600); err != nil {
		_ = ln.Close()
		return nil, fmt.Errorf("dc: write ipc state: %w", err)
	}

	go func() {
		_ = s.server.Serve(ln)
	}()
	return s, nil
}

// Addr returns the bound loopback host:port.
func (s *IPCServer) Addr() string {
	if s == nil || s.listener == nil {
		return ""
	}
	return s.listener.Addr().String()
}

// PublishEvent fans out a core event to all connected /events subscribers without blocking.
func (s *IPCServer) PublishEvent(ev Event) {
	if s == nil {
		return
	}
	s.subMu.Lock()
	defer s.subMu.Unlock()
	for ch := range s.subs {
		select {
		case ch <- ev:
		default:
		}
	}
}

// Close stops the HTTP server and removes the IPC descriptor file if it still belongs to us.
func (s *IPCServer) Close() error {
	if s == nil {
		return nil
	}
	_ = s.server.Close()

	s.subMu.Lock()
	for ch := range s.subs {
		close(ch)
		delete(s.subs, ch)
	}
	s.subMu.Unlock()

	if data, err := os.ReadFile(s.statePath); err == nil {
		var st IPCState
		if json.Unmarshal(data, &st) == nil && st.Token == s.token {
			_ = os.Remove(s.statePath)
		}
	}
	return nil
}

func (s *IPCServer) authorized(r *http.Request) bool {
	return r.Header.Get("Authorization") == "Bearer "+s.token
}

func (s *IPCServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	if !s.authorized(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"ok":           true,
		"pid":          os.Getpid(),
		"accountsPath": s.accountsPath,
	})
}

type ipcRPCRequest struct {
	Method string `json:"method"`
	Params any    `json:"params"`
}

type ipcRPCResponse struct {
	Result json.RawMessage `json:"result,omitempty"`
	Error  string          `json:"error,omitempty"`
}

func (s *IPCServer) handleRPC(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !s.authorized(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	var req ipcRPCRequest
	if err := json.NewDecoder(io.LimitReader(r.Body, 4<<20)).Decode(&req); err != nil {
		http.Error(w, "invalid json: "+err.Error(), http.StatusBadRequest)
		return
	}
	if req.Method == "" {
		http.Error(w, "missing method", http.StatusBadRequest)
		return
	}
	res, err := s.caller.Call(r.Context(), req.Method, req.Params)
	w.Header().Set("Content-Type", "application/json")
	if err != nil {
		w.WriteHeader(http.StatusBadGateway)
		_ = json.NewEncoder(w).Encode(ipcRPCResponse{Error: err.Error()})
		return
	}
	_ = json.NewEncoder(w).Encode(ipcRPCResponse{Result: res})
}

func (s *IPCServer) handleEvents(w http.ResponseWriter, r *http.Request) {
	if !s.authorized(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}
	ch := make(chan Event, 64)
	s.subMu.Lock()
	s.subs[ch] = struct{}{}
	s.subMu.Unlock()
	defer func() {
		s.subMu.Lock()
		if _, exists := s.subs[ch]; exists {
			delete(s.subs, ch)
		}
		s.subMu.Unlock()
	}()

	w.Header().Set("Content-Type", "application/x-ndjson")
	w.Header().Set("Cache-Control", "no-cache")
	w.WriteHeader(http.StatusOK)
	flusher.Flush()

	enc := json.NewEncoder(w)
	for {
		select {
		case <-r.Context().Done():
			return
		case ev, open := <-ch:
			if !open {
				return
			}
			if err := enc.Encode(ev); err != nil {
				return
			}
			flusher.Flush()
		}
	}
}

// IPCClient connects to a running IPCServer and implements Caller.
type IPCClient struct {
	state      IPCState
	httpClient *http.Client
}

// DiscoverIPC checks for a healthy local IPC bridge for accountsPath.
// Returns (nil, nil) when no active bridge is running.
func DiscoverIPC(ctx context.Context, accountsPath string) (*IPCClient, error) {
	p := IPCStatePath(accountsPath)
	raw, err := os.ReadFile(p)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	var st IPCState
	if err := json.Unmarshal(raw, &st); err != nil || st.Addr == "" || st.Token == "" {
		return nil, nil
	}
	client := &IPCClient{
		state:      st,
		httpClient: &http.Client{Timeout: 60 * time.Second},
	}
	hctx, cancel := context.WithTimeout(ctx, 1500*time.Millisecond)
	defer cancel()
	if err := client.Ping(hctx); err != nil {
		return nil, nil
	}
	return client, nil
}

// State returns the discovered IPC metadata.
func (c *IPCClient) State() IPCState {
	return c.state
}

// Ping verifies the remote IPC bridge is reachable and authenticated.
func (c *IPCClient) Ping(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://"+c.state.Addr+"/health", nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.state.Token)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("dc: ipc health status %d", resp.StatusCode)
	}
	return nil
}

// Call forwards a JSON-RPC request over the authenticated localhost IPC bridge.
func (c *IPCClient) Call(ctx context.Context, method string, params any) (json.RawMessage, error) {
	if params == nil {
		params = []any{}
	}
	body, err := json.Marshal(ipcRPCRequest{Method: method, Params: params})
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "http://"+c.state.Addr+"/rpc", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.state.Token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var out ipcRPCResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("dc: ipc decode response (status %d): %w", resp.StatusCode, err)
	}
	if out.Error != "" {
		return nil, errors.New(out.Error)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("dc: ipc rpc status %d", resp.StatusCode)
	}
	return out.Result, nil
}

// StreamEvents subscribes to the NDJSON /events feed from the running GUI/daemon.
func (c *IPCClient) StreamEvents(ctx context.Context, onEvent func(Event) bool) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://"+c.state.Addr+"/events", nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.state.Token)
	streamClient := &http.Client{}
	resp, err := streamClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("dc: ipc events status %d", resp.StatusCode)
	}
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 64*1024), 4*1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var ev Event
		if err := json.Unmarshal([]byte(line), &ev); err != nil {
			continue
		}
		if !onEvent(ev) {
			return nil
		}
	}
	if ctx.Err() != nil {
		return ctx.Err()
	}
	return scanner.Err()
}
