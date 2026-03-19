package monitor

import (
	"fmt"
	"os/exec"
	"sync"
	"sync/atomic"
)

type Notifier struct {
	mu            sync.Mutex
	notified      map[string]SessionStatus
	notifyConfirm atomic.Bool
	notifyWaiting atomic.Bool
	fireFunc      func(projectName string, status SessionStatus)
}

func NewNotifier() *Notifier {
	n := &Notifier{notified: make(map[string]SessionStatus)}
	n.notifyConfirm.Store(true)
	n.notifyWaiting.Store(false)
	return n
}

func (n *Notifier) SetNotifyConfirm(v bool) { n.notifyConfirm.Store(v) }
func (n *Notifier) SetNotifyWaiting(v bool) { n.notifyWaiting.Store(v) }

func (n *Notifier) shouldNotify(status SessionStatus) bool {
	switch status {
	case StatusConfirm:
		return n.notifyConfirm.Load()
	case StatusWaiting:
		return n.notifyWaiting.Load()
	}
	return false
}

func (n *Notifier) Check(prev, curr []Session) {
	n.mu.Lock()
	defer n.mu.Unlock()

	currMap := make(map[string]Session, len(curr))
	for _, s := range curr {
		currMap[s.SessionID] = s
	}

	for id, prevStatus := range n.notified {
		s, ok := currMap[id]
		if !ok || s.Status != prevStatus {
			delete(n.notified, id)
		}
	}

	for _, s := range curr {
		if !n.shouldNotify(s.Status) {
			continue
		}
		if _, already := n.notified[s.SessionID]; already {
			continue
		}
		n.notified[s.SessionID] = s.Status
		if n.fireFunc != nil {
			n.fireFunc(s.ProjectName, s.Status)
		} else {
			fireNotification(s.ProjectName, s.Status)
		}
	}
}

var statusMessages = map[SessionStatus]string{
	StatusConfirm: "needs confirmation",
	StatusWaiting: "waiting for input",
}

func fireNotification(projectName string, status SessionStatus) {
	text := statusMessages[status]
	if text == "" {
		text = string(status)
	}
	msg := fmt.Sprintf(`display notification "%s %s" with title "Vigil"`, projectName, text)
	exec.Command("osascript", "-e", msg).Run()
}
