package node

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	commit "github.com/puoklam/tp-commit"
	"github.com/puoklam/tp-commit/notify"
)

type Node struct {
	notify.Notifier
	mu        sync.Mutex
	Ip        string
	commits   map[string]*commit.Commit
	Idle      time.Duration // commit idle timeout
	Detectors []Detector
}

func (n *Node) lazyInit() {
	n.mu.Lock()
	defer n.mu.Unlock()
	if n.commits == nil {
		n.commits = make(map[string]*commit.Commit)
	}
}

// Prepare implements Coordinator
func (n *Node) Prepare(ctx context.Context, id commit.CommitID, ips []string, timeout time.Duration) error {
	// regiter commit before broadcasting
	n.RegisterCommit(commit.New(id, n.Ip, ips, timeout))

	// broadcast
	body := &commit.MsgBody{
		Ip:      n.Ip,
		Type:    commit.TypePrepare,
		ID:      id,
		Payload: ips,
		Timeout: timeout,
	}
	b, err := json.Marshal(body)
	if err != nil {
		return err
	}
	return n.Emit(ctx, b)
}

// abort
func (n *Node) Abort(ctx context.Context, id commit.CommitID, ip string) error {
	body := &commit.MsgBody{
		Ip:      ip,
		Type:    commit.TypeResp,
		ID:      id,
		Payload: false,
	}
	b, err := json.Marshal(body)
	if err != nil {
		return err
	}
	return n.Emit(ctx, b)
}

func (n *Node) NewCommit(id commit.CommitID, host string, ips []string, timeout time.Duration) *commit.Commit {
	c := commit.New(id, host, ips, timeout)
	if n.RegisterCommit(c) {
		return c
	}
	return n.GetCommit(id)
}

func (n *Node) GetCommit(id commit.CommitID) *commit.Commit {
	n.lazyInit()
	n.mu.Lock()
	defer n.mu.Unlock()
	return n.commits[id.String()]
}

func (n *Node) RegisterCommit(c *commit.Commit) bool {
	n.lazyInit()
	n.mu.Lock()
	defer n.mu.Unlock()
	id := c.ID().String()
	if n.commits[id] != nil {
		return false
	}
	c.StartTimer(n.Idle)
	n.commits[id] = c
	return true
}

func (n *Node) Done(ctx context.Context, id commit.CommitID, ok bool) error {
	body := &commit.MsgBody{
		Ip:      n.Ip,
		Type:    commit.TypeResp,
		ID:      id,
		Payload: ok,
	}
	b, err := json.Marshal(body)
	if err != nil {
		return err
	}
	return n.Emit(ctx, b)
}

func (n *Node) CloseCommit(id commit.CommitID) error {
	c := n.GetCommit(id)
	n.mu.Lock()
	defer n.mu.Unlock()
	if c == nil {
		return nil
	}
	delete(n.commits, id.String())
	return c.Close()
}

func (n *Node) Close() error {
	for _, c := range n.commits {
		n.CloseCommit(c.ID())
	}
	return n.Notifier.Close()
}

// Coordinator only responsible for broadcasting prepare signal
type Coordinator interface {
	Prepare(ctx context.Context, id commit.CommitID, ips []string) error
}
