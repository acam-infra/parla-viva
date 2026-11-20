package dc

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"sync/atomic"
)

var (
	ErrNotConnected = errors.New("dc: rpc transport is not connected")
	ErrClosed       = errors.New("dc: rpc transport closed")
)

type rpcRequest struct {
	JSONRPC string `json:"jsonrpc"`
	ID      int64  `json:"id"`
	Method  string `json:"method"`
	Params  any    `json:"params"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (e *rpcError) Error() string {
	return fmt.Sprintf("rpc error %d: %s", e.Code, e.Message)
}

type rpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int64           `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

type callResult struct {
	raw json.RawMessage
	err error
}

// Transport owns the stdio JSON-RPC 2.0 connection to deltachat-rpc-server.
// Mirrors Parla's RpcTransport: a single writer goroutine keeps pipe
// backpressure off callers while preserving request order, and each connection
// increments a generation counter to invalidate stale callbacks.
type Transport struct {
	mu           sync.Mutex
	cmd          *exec.Cmd
	stdin        io.WriteCloser
	stdout       io.ReadCloser
	writeCh      chan []byte
	doneCh       chan struct{}
	pending      map[int64]chan callResult
	nextID       atomic.Int64
	generation   atomic.Uint64
	connected    atomic.Bool
	onDisconnect func(error)
}

// NewTransport creates a disconnected Transport.
func NewTransport() *Transport {
	return &Transport{
		pending: make(map[int64]chan callResult),
	}
}

// SetOnDisconnect registers an optional callback fired when the read loop ends.
func (t *Transport) SetOnDisconnect(fn func(error)) {
	t.mu.Lock()
	t.onDisconnect = fn
	t.mu.Unlock()
}

// Connected reports whether the transport currently has an active connection.
func (t *Transport) Connected() bool {
	return t.connected.Load()
}

// Generation returns the monotonically increasing connection generation.
func (t *Transport) Generation() uint64 {
	return t.generation.Load()
}

// StartProcess launches deltachat-rpc-server with DC_ACCOUNTS_PATH set and attaches stdio pipes.
func (t *Transport) StartProcess(binary string, accountsPath string) error {
	if binary == "" {
		return errors.New("dc: empty rpc server binary path")
	}
	cmd := exec.Command(binary)
	if accountsPath != "" {
		cmd.Env = append(os.Environ(), "DC_ACCOUNTS_PATH="+accountsPath)
	}
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("dc: stdin pipe: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		_ = stdin.Close()
		return fmt.Errorf("dc: stdout pipe: %w", err)
	}
	cmd.Stderr = io.Discard
	if err := cmd.Start(); err != nil {
		_ = stdin.Close()
		_ = stdout.Close()
		return fmt.Errorf("dc: start %s: %w", binary, err)
	}
	t.attach(cmd, stdout, stdin)
	return nil
}

// StartPipes attaches the transport to arbitrary reader/writer pipes (used by tests).
func (t *Transport) StartPipes(r io.ReadCloser, w io.WriteCloser) {
	t.attach(nil, r, w)
}

func (t *Transport) attach(cmd *exec.Cmd, r io.ReadCloser, w io.WriteCloser) {
	_ = t.Close()

	t.mu.Lock()
	gen := t.generation.Add(1)
	t.cmd = cmd
	t.stdout = r
	t.stdin = w
	t.writeCh = make(chan []byte, 64)
	t.doneCh = make(chan struct{})
	t.pending = make(map[int64]chan callResult)
	t.connected.Store(true)
	writeCh := t.writeCh
	doneCh := t.doneCh
	t.mu.Unlock()

	go t.writeLoop(w, writeCh, doneCh)
	go t.readLoop(r, gen)
}

func (t *Transport) writeLoop(w io.Writer, writeCh <-chan []byte, doneCh <-chan struct{}) {
	for {
		select {
		case <-doneCh:
			return
		case payload, ok := <-writeCh:
			if !ok {
				return
			}
			if _, err := w.Write(payload); err != nil {
				t.failAll(err)
				return
			}
		}
	}
}

func (t *Transport) readLoop(r io.Reader, gen uint64) {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 64*1024), 16*1024*1024)

	var readErr error
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var resp rpcResponse
		if err := json.Unmarshal(line, &resp); err != nil {
			continue
		}
		t.mu.Lock()
		if t.generation.Load() != gen {
			t.mu.Unlock()
			return
		}
		ch, ok := t.pending[resp.ID]
		if ok {
			delete(t.pending, resp.ID)
		}
		t.mu.Unlock()

		if ok {
			if resp.Error != nil {
				ch <- callResult{err: resp.Error}
			} else {
				ch <- callResult{raw: resp.Result}
			}
		}
	}
	readErr = scanner.Err()
	if readErr == nil {
		readErr = io.EOF
	}
	t.failAllForGeneration(gen, readErr)
}

func (t *Transport) failAll(err error) {
	t.failAllForGeneration(t.generation.Load(), err)
}

func (t *Transport) failAllForGeneration(gen uint64, err error) {
	t.mu.Lock()
	if t.generation.Load() != gen || !t.connected.Load() {
		t.mu.Unlock()
		return
	}
	t.connected.Store(false)
	pending := t.pending
	t.pending = make(map[int64]chan callResult)
	cb := t.onDisconnect
	doneCh := t.doneCh
	t.mu.Unlock()

	if doneCh != nil {
		select {
		case <-doneCh:
		default:
			close(doneCh)
		}
	}
	for _, ch := range pending {
		ch <- callResult{err: err}
	}
	if cb != nil {
		cb(err)
	}
}

// Call invokes a JSON-RPC method and returns the raw JSON result payload.
func (t *Transport) Call(ctx context.Context, method string, params any) (json.RawMessage, error) {
	if !t.connected.Load() {
		return nil, ErrNotConnected
	}
	if params == nil {
		params = []any{}
	}
	id := t.nextID.Add(1)
	payload, err := json.Marshal(rpcRequest{
		JSONRPC: "2.0",
		ID:      id,
		Method:  method,
		Params:  params,
	})
	if err != nil {
		return nil, err
	}
	payload = append(payload, '\n')

	ch := make(chan callResult, 1)

	t.mu.Lock()
	if !t.connected.Load() {
		t.mu.Unlock()
		return nil, ErrNotConnected
	}
	t.pending[id] = ch
	writeCh := t.writeCh
	doneCh := t.doneCh
	t.mu.Unlock()

	select {
	case writeCh <- payload:
	case <-doneCh:
		t.removePending(id)
		return nil, ErrClosed
	case <-ctx.Done():
		t.removePending(id)
		return nil, ctx.Err()
	}

	select {
	case res := <-ch:
		return res.raw, res.err
	case <-ctx.Done():
		t.removePending(id)
		return nil, ctx.Err()
	}
}

func (t *Transport) removePending(id int64) {
	t.mu.Lock()
	delete(t.pending, id)
	t.mu.Unlock()
}

// Close shuts down the transport, closes pipes, and waits for any subprocess.
func (t *Transport) Close() error {
	t.mu.Lock()
	t.connected.Swap(false)
	cmd := t.cmd
	stdin := t.stdin
	stdout := t.stdout
	doneCh := t.doneCh
	pending := t.pending
	t.cmd = nil
	t.stdin = nil
	t.stdout = nil
	t.pending = make(map[int64]chan callResult)
	t.mu.Unlock()

	if doneCh != nil {
		select {
		case <-doneCh:
		default:
			close(doneCh)
		}
	}
	for _, ch := range pending {
		ch <- callResult{err: ErrClosed}
	}
	if stdin != nil {
		_ = stdin.Close()
	}
	if stdout != nil {
		_ = stdout.Close()
	}
	if cmd != nil && cmd.Process != nil {
		_ = cmd.Wait()
	}
	return nil
}
