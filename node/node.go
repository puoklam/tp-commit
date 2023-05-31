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
	mu      sync.Mutex
	Ip      string
	Timeout time.Duration // commit timeout
	commits map[string]*commit.Commit
}

func (n *Node) lazyInit() {
	n.mu.Lock()
	defer n.mu.Unlock()
	if n.commits == nil {
		n.commits = make(map[string]*commit.Commit)
	}
}

// Prepare implements Coordinator
func (n *Node) Prepare(ctx context.Context, id commit.CommitID, ips []string) error {
	// regiter commit before broadcasting
	n.RegisterCommit(commit.New(id, n.Ip, ips, n.Timeout))

	// broadcast
	body := &commit.MsgBody{
		Ip:      n.Ip,
		Type:    commit.TypePrepare,
		ID:      id,
		Payload: ips,
	}
	b, err := json.Marshal(body)
	if err != nil {
		return err
	}
	return n.Emit(ctx, b)
}

func (n *Node) NewCommit(id commit.CommitID, host string, ips []string) *commit.Commit {
	c := commit.New(id, host, ips, n.Timeout)
	n.RegisterCommit(c)
	return c
}

func (n *Node) GetCommit(id commit.CommitID) *commit.Commit {
	n.lazyInit()
	n.mu.Lock()
	defer n.mu.Unlock()
	return n.commits[id.String()]
}

func (n *Node) RegisterCommit(c *commit.Commit) {
	n.lazyInit()
	n.mu.Lock()
	defer n.mu.Unlock()
	id := c.ID().String()
	if n.commits[id] != nil {
		return
	}
	n.commits[id] = c
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

func (n *Node) Detect() {

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
