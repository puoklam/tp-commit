package commit

import (
	"sort"
	"testing"

	"github.com/google/uuid"
	"github.com/puoklam/tp-commit/internal/set"
)

var (
	id   = CommitID{uuid.New()}
	host = "0.0.0.0:9000"
	ip2  = "1.1.1.1:9000"
	ips  = sort.StringSlice([]string{host, ip2})
)

func initTest(t *testing.T) *Commit {
	c := New(id, "0.0.0.0:9000", ips, 0)
	Ips := c.Participants()
	votes := c.Votes()
	if votes.Len() != 0 {
		t.Errorf("votes = %v, want %v", votes, set.New[string]())
	}
	if Ips.Len() != len(ips) {
		t.Errorf("ips = %v, want %v", Ips, ips)
	}
	for _, ip := range ips {
		if !Ips.Has(ip) {
			t.Errorf("ips does not contain %s", ip)
		}
	}
	return c
}

func verifyCommit(t *testing.T, c *Commit, d bool, ok bool) {
	if c.Decided() != d {
		t.Errorf("decided = %t, want %t", c.Decided(), d)
	}
	if d {
		if len(c.Ok) != 1 {
			t.Errorf("decided but signal not sent")
		}
		if Ok := <-c.Ok; Ok != ok {
			t.Errorf("signal = %t, want %t", Ok, ok)
		}
	}
}

func TestVoteTrue(t *testing.T) {
	c := initTest(t)

	c.Vote(host, true)
	votes := c.Votes()
	if votes.Len() != 1 {
		t.Errorf("votes = %v, want %v", votes, set.New[string](set.WithElements([]string{host})))
	}
	verifyCommit(t, c, false, false)
}

func TestVoteFalse(t *testing.T) {
	c := initTest(t)

	c.Vote(host, false)
	votes := c.Votes()
	if votes.Len() != 0 {
		t.Errorf("votes = %v, want %v", votes, set.New[string]())
	}
	verifyCommit(t, c, true, false)
}

func TestVoteFinish(t *testing.T) {
	c := initTest(t)

	c.Vote(host, true)
	c.Vote(ip2, true)
	votes := c.Votes()
	if votes.Len() != 2 {
		t.Errorf("votes = %v, want %v", votes, set.New[string](set.WithElements([]string{host, ip2})))
	}
	verifyCommit(t, c, true, true)
}

func TestInvalidIp(t *testing.T) {
	c := initTest(t)

	c.Vote("", true)
	votes := c.Votes()
	if votes.Len() != 0 {
		t.Errorf("votes = %v, want %v", votes, set.New[string]())
	}
	verifyCommit(t, c, false, false)
}
