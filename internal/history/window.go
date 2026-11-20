package history

// DefaultBatchSize matches Parla's message_history.vala initial batch size.
const DefaultBatchSize = 30

// Window represents a bounded slice [Start, End) over a chat's full message ID list.
// Ported from Parla's CORE-001 (src/message_history.vala): distant search jumps load
// at most 61 surrounding messages instead of every intervening page.
type Window struct {
	AllIDs []int
	Start  int
	End    int
}

// NewLatest initializes a window covering the tail of allIDs, expanding backwards
// to include one message before firstUnreadID when an unread divider is present.
func NewLatest(allIDs []int, firstUnreadID int, batchSize int) Window {
	if batchSize <= 0 {
		batchSize = DefaultBatchSize
	}
	n := len(allIDs)
	start := 0
	if n > batchSize {
		start = n - batchSize
	}
	if idx := FindID(allIDs, firstUnreadID); idx >= 0 {
		withContext := idx - 1
		if withContext < 0 {
			withContext = 0
		}
		if withContext < start {
			start = withContext
		}
	}
	return Window{AllIDs: allIDs, Start: start, End: n}
}

// JumpTo returns a bounded context window centered around targetMsgID
// (up to 30 before and 30 after, 61 messages total).
func JumpTo(allIDs []int, targetMsgID int) (Window, bool) {
	idx := FindID(allIDs, targetMsgID)
	if idx < 0 {
		return Window{}, false
	}
	start, end := ContextRange(len(allIDs), idx)
	return Window{AllIDs: allIDs, Start: start, End: end}, true
}

// ContextRange computes [start, end) around target index with at most 61 entries.
func ContextRange(total, target int) (start, end int) {
	if total <= 0 {
		return 0, 0
	}
	if target < 0 {
		target = 0
	}
	if target >= total {
		target = total - 1
	}
	start = target - 30
	if start < 0 {
		start = 0
	}
	end = target + 31
	if end > total {
		end = total
	}
	return start, end
}

// VisibleIDs returns the currently loaded message IDs in the window.
func (w Window) VisibleIDs() []int {
	if w.Start >= w.End || w.Start < 0 || w.End > len(w.AllIDs) {
		return nil
	}
	return w.AllIDs[w.Start:w.End]
}

// HasEarlier reports whether older messages exist before the current window.
func (w Window) HasEarlier() bool {
	return w.Start > 0
}

// HasLater reports whether newer messages exist after the current window.
func (w Window) HasLater() bool {
	return w.End < len(w.AllIDs)
}

// ExpandEarlier extends the window toward older messages by step entries and returns newly added IDs.
func (w *Window) ExpandEarlier(step int) []int {
	if step <= 0 {
		step = DefaultBatchSize
	}
	if w.Start == 0 {
		return nil
	}
	prev := w.Start
	w.Start -= step
	if w.Start < 0 {
		w.Start = 0
	}
	return w.AllIDs[w.Start:prev]
}

// ExpandLater extends the window toward newer messages by step entries and returns newly added IDs.
func (w *Window) ExpandLater(step int) []int {
	if step <= 0 {
		step = DefaultBatchSize
	}
	if w.End >= len(w.AllIDs) {
		return nil
	}
	prev := w.End
	w.End += step
	if w.End > len(w.AllIDs) {
		w.End = len(w.AllIDs)
	}
	return w.AllIDs[prev:w.End]
}

// FindID returns the index of messageID in ids, or -1 if absent.
func FindID(ids []int, messageID int) int {
	if messageID <= 0 {
		return -1
	}
	for i, id := range ids {
		if id == messageID {
			return i
		}
	}
	return -1
}
