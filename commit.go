package commit

import (
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/puoklam/tp-commit/internal/set"
)

type SignalType int
type SignalPayload any

const (
	TypePrepare SignalType = iota
	TypeResp
)

var (
	MsgOK    SignalPayload = true
	MsgNotOK SignalPayload = false
)

type Interface interface {
	Vote(p string, ok bool)
	Decided() bool
	Participants() set.Set[string]
	Votes() set.Set[string]
	Close() error
}

type CommitID struct {
	uuid.UUID
}

func (id *CommitID) UnmarshalJSON(b []byte) error {
	s := strings.Trim(string(b), "\"")
	u, err := uuid.Parse(s)
	if err != nil {
		return err
	}

	id.UUID = u
	return nil
}

type MsgBody struct {
	Host    string        `json:"host"`
	Ip      string        `json:"ip"`
	Type    SignalType    `json:"type"`
	ID      CommitID      `json:"commit_id"`
	Payload SignalPayload `json:"payload"`
	Timeout time.Duration `json:"timeout"`
}

type Commit struct {
	mu    sync.RWMutex
	id    CommitID
	host  string           // host ip
	ps    *set.Set[string] // participants, slice of ips
	votes *set.Set[string] // votes, slice of ips replied "ok"
	d     bool             // dicided
	Ok    chan bool        // able to commit
	it    *time.Timer      // idle timeout timer
	t     time.Duration    // commit timeout duration
}

func (c *Commit) ID() CommitID {
	return c.id
}

func (c *Commit) vote(p string, ok bool) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.d || !c.ps.Has(p) || c.votes.Has(p) {
		return false
	}

	d := true
	if ok {
		c.votes.Add(p)
		if !set.Equal(c.ps, c.votes) {
			d = false
		}
	}
	c.d = d
	if c.d && c.it != nil {
		c.it.Stop()
	}
	return d
}

func (c *Commit) Vote(p string, ok bool) {
	d := c.vote(p, ok)
	if d {
		// just decided after this vote
		c.Ok <- ok
	}
}

func (c *Commit) Decided() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.d
}

func (c *Commit) Participants() set.Set[string] {
	return *c.ps.Clone()
}

func (c *Commit) Votes() set.Set[string] {
	return *c.votes.Clone()
}

func (c *Commit) StartTimer(idle time.Duration) {
	if idle > 0 {
		c.it = time.AfterFunc(idle, func() {
			c.Ok <- false
		})
	}
}

// Duration implements node.Detector
func (c *Commit) Duration() time.Duration {
	return c.t
}

func (c *Commit) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.ps = nil
	c.votes = nil
	close(c.Ok)
	<-c.Ok
	c.it.Stop()
	return nil
}

func New(id CommitID, host string, ips []string, timeout time.Duration) *Commit {
	ps := set.New[string](set.WithLock[string](), set.WithElements(ips))
	return &Commit{
		id:    id,
		host:  host,
		ps:    ps,
		votes: set.New[string](set.WithLock[string]()),
		d:     false,
		Ok:    make(chan bool, 1),
		t:     timeout,
	}
}
