package commit

type Node struct {
	Ip string
}

type Commit struct {
	ps    []*Node // participats
	votes []bool  // votes
	d     bool    // dicided
}

func New(nodes []*Node) *Commit {
	len := len(nodes)
	c := &Commit{
		ps:    make([]*Node, len),
		votes: make([]bool, len),
		d:     false,
	}
	copy(c.ps, nodes)
	return c
}
