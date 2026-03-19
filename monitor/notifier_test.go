package monitor

import (
	"sync"
	"testing"
)

type firedEntry struct {
	project string
	status  SessionStatus
}

func newTestNotifier() (*Notifier, *[]firedEntry) {
	fired := &[]firedEntry{}
	n := &Notifier{
		notified: make(map[string]SessionStatus),
		fireFunc: func(projectName string, status SessionStatus) {
			*fired = append(*fired, firedEntry{projectName, status})
		},
	}
	n.notifyConfirm.Store(true)
	n.notifyWaiting.Store(false)
	return n, fired
}

func TestNotifier_FiresOnNewConfirm(t *testing.T) {
	n, fired := newTestNotifier()
	curr := []Session{{SessionID: "s1", ProjectName: "myproject", Status: StatusConfirm}}
	n.Check(nil, curr)
	if len(*fired) != 1 || (*fired)[0].project != "myproject" {
		t.Errorf("expected 1 notification for myproject, got %v", *fired)
	}
}

func TestNotifier_DoesNotRepeatWhileConfirm(t *testing.T) {
	n, fired := newTestNotifier()
	curr := []Session{{SessionID: "s1", ProjectName: "p", Status: StatusConfirm}}
	n.Check(nil, curr)
	n.Check(curr, curr)
	if len(*fired) != 1 {
		t.Errorf("expected exactly 1 notification, got %d", len(*fired))
	}
}

func TestNotifier_RefiresAfterLeavingConfirm(t *testing.T) {
	n, fired := newTestNotifier()
	confirm := []Session{{SessionID: "s1", ProjectName: "p", Status: StatusConfirm}}
	idle := []Session{{SessionID: "s1", ProjectName: "p", Status: StatusIdle}}
	n.Check(nil, confirm)
	n.Check(confirm, idle)
	n.Check(idle, confirm)
	if len(*fired) != 2 {
		t.Errorf("expected 2 notifications, got %d", len(*fired))
	}
}

func TestNotifier_SkipsConfirmWhenDisabled(t *testing.T) {
	n, fired := newTestNotifier()
	n.notifyConfirm.Store(false)
	curr := []Session{{SessionID: "s1", ProjectName: "p", Status: StatusConfirm}}
	n.Check(nil, curr)
	if len(*fired) != 0 {
		t.Errorf("expected 0 notifications when confirm disabled, got %d", len(*fired))
	}
}

func TestNotifier_FiresOnWaitingWhenEnabled(t *testing.T) {
	n, fired := newTestNotifier()
	n.notifyWaiting.Store(true)
	curr := []Session{{SessionID: "s1", ProjectName: "p", Status: StatusWaiting}}
	n.Check(nil, curr)
	if len(*fired) != 1 || (*fired)[0].status != StatusWaiting {
		t.Errorf("expected 1 waiting notification, got %v", *fired)
	}
}

func TestNotifier_SkipsWaitingWhenDisabled(t *testing.T) {
	n, fired := newTestNotifier()
	curr := []Session{{SessionID: "s1", ProjectName: "p", Status: StatusWaiting}}
	n.Check(nil, curr)
	if len(*fired) != 0 {
		t.Errorf("expected 0 waiting notifications when disabled, got %d", len(*fired))
	}
}

func TestNotifier_ConcurrentConfig(t *testing.T) {
	n, _ := newTestNotifier()
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(v bool) {
			defer wg.Done()
			n.SetNotifyConfirm(v)
			n.SetNotifyWaiting(v)
		}(i%2 == 0)
	}
	wg.Wait()
}
