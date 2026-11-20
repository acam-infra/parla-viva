package history

import "testing"

func makeIDs(n int) []int {
	ids := make([]int, n)
	for i := range ids {
		ids[i] = i + 1
	}
	return ids
}

func TestJumpToBounded61Messages(t *testing.T) {
	ids := makeIDs(2000)
	win, ok := JumpTo(ids, 1000)
	if !ok {
		t.Fatal("expected target 1000 to be found")
	}
	vis := win.VisibleIDs()
	if len(vis) != 61 {
		t.Fatalf("len(VisibleIDs) = %d, want 61", len(vis))
	}
	if vis[0] != 970 || vis[len(vis)-1] != 1030 {
		t.Fatalf("range = [%d..%d], want [970..1030]", vis[0], vis[len(vis)-1])
	}
	if !win.HasEarlier() || !win.HasLater() {
		t.Fatal("expected both earlier and later history to be available")
	}

	addedEarlier := win.ExpandEarlier(20)
	if len(addedEarlier) != 20 || addedEarlier[0] != 950 {
		t.Fatalf("ExpandEarlier = %d items starting at %d", len(addedEarlier), addedEarlier[0])
	}
	addedLater := win.ExpandLater(15)
	if len(addedLater) != 15 || addedLater[len(addedLater)-1] != 1045 {
		t.Fatalf("ExpandLater = %d items ending at %d", len(addedLater), addedLater[len(addedLater)-1])
	}
}

func TestNewLatestIncludesUnreadContext(t *testing.T) {
	ids := makeIDs(200)
	win := NewLatest(ids, 151, 30)
	vis := win.VisibleIDs()
	if vis[0] != 150 || vis[len(vis)-1] != 200 {
		t.Fatalf("range = [%d..%d], want [150..200]", vis[0], vis[len(vis)-1])
	}
}
