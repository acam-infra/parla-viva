package notify

import (
	"sync"
	"testing"
	"time"
)

func TestNotifierThrottleAndNonBlocking(t *testing.T) {
	var mu sync.Mutex
	var got []string
	done := make(chan struct{}, 4)

	n := &Notifier{
		ch:       make(chan Item, 16),
		lastSent: make(map[int]time.Time),
		minGap:   200 * time.Millisecond,
		sendFn: func(title, body string) {
			mu.Lock()
			got = append(got, title+":"+body)
			mu.Unlock()
			done <- struct{}{}
		},
	}
	go n.loop()
	defer n.Close()

	// Burst on chat 12 should throttle to 1 notification; chat 15 should fire independently
	n.Push(Item{ChatID: 12, Title: "Mac", Body: "msg 1"})
	n.Push(Item{ChatID: 12, Title: "Mac", Body: "msg 2 (throttled)"})
	n.Push(Item{ChatID: 15, Title: "Bot", Body: "ping"})

	for i := 0; i < 2; i++ {
		select {
		case <-done:
		case <-time.After(time.Second):
			t.Fatal("timed out waiting for notifications")
		}
	}
	mu.Lock()
	defer mu.Unlock()
	if len(got) != 2 {
		t.Fatalf("expected 2 notifications (1 throttled), got %v", got)
	}
}
