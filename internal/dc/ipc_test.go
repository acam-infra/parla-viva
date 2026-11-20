package dc

import (
	"context"
	"encoding/json"
	"testing"
	"time"
)

type stubCaller struct {
	fn func(ctx context.Context, method string, params any) (json.RawMessage, error)
}

func (s stubCaller) Call(ctx context.Context, method string, params any) (json.RawMessage, error) {
	return s.fn(ctx, method, params)
}

func TestIPCBridgeRoundTripAndEvents(t *testing.T) {
	dir := t.TempDir()
	caller := stubCaller{
		fn: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
			switch method {
			case "get_selected_account_id":
				return json.RawMessage(`7`), nil
			case "send_msg":
				return json.RawMessage(`42`), nil
			default:
				return json.RawMessage(`null`), nil
			}
		},
	}

	srv, err := StartIPCServer(caller, dir)
	if err != nil {
		t.Fatalf("StartIPCServer: %v", err)
	}
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	ipcClient, err := DiscoverIPC(ctx, dir)
	if err != nil || ipcClient == nil {
		t.Fatalf("DiscoverIPC failed: client=%v err=%v", ipcClient, err)
	}

	dcClient := NewClientWithCaller(ipcClient)
	sel, err := dcClient.GetSelectedAccountID(ctx)
	if err != nil || sel != 7 {
		t.Fatalf("GetSelectedAccountID = %d, %v; want 7", sel, err)
	}
	msgID, err := dcClient.SendTextMessage(ctx, 7, 12, "hello ipc", 0)
	if err != nil || msgID != 42 {
		t.Fatalf("SendTextMessage = %d, %v; want 42", msgID, err)
	}

	// Verify /events streaming fan-out
	gotCh := make(chan Event, 1)
	streamCtx, streamCancel := context.WithCancel(ctx)
	defer streamCancel()
	go func() {
		_ = ipcClient.StreamEvents(streamCtx, func(ev Event) bool {
			gotCh <- ev
			return false
		})
	}()

	time.Sleep(80 * time.Millisecond)
	srv.PublishEvent(Event{
		ContextID: 7,
		Event: EventPayload{
			Kind:   "IncomingMsg",
			ChatID: 12,
			MsgID:  99,
		},
	})

	select {
	case ev := <-gotCh:
		if ev.ContextID != 7 || ev.Event.Kind != "IncomingMsg" || ev.Event.MsgID != 99 {
			t.Fatalf("unexpected event: %+v", ev)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for streamed IPC event")
	}
}
