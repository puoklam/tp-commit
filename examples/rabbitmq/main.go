package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	commit "github.com/puoklam/tp-commit"
	"github.com/puoklam/tp-commit/node"
	"github.com/puoklam/tp-commit/notify"
	amqp "github.com/rabbitmq/amqp091-go"
)

var (
	ErrJsonFormat     = errors.New("json format not match")
	ErrInvalidType    = errors.New("invalid signal type")
	ErrInvalidPayload = errors.New("invalid payload")
	ErrCommitNotFound = errors.New("commit not found")
)

func failOnError(err error, msg string) {
	if err != nil {
		log.Panicf("%s: %s", msg, err)
	}
}

func newNode(ip, key string, detect bool, receiveFn notify.ReceiveFunc) *node.Node {
	conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
	failOnError(err, "Failed to connect to RabbitMQ")

	mq := &notify.RabbitMQ{
		Conn:      conn,
		Exchange:  "commits_topic",
		Queue:     fmt.Sprintf("node[%s] queue", ip),
		Prefix:    "commits",
		Key:       key,
		ReceiveFn: receiveFn,
	}
	n := &node.Node{
		Notifier: mq,
		Ip:       ip,
		Idle:     5 * time.Second,
	}
	if detect {
		n.Detectors = append(n.Detectors, &node.TimeoutDetector{})
	}
	return n
}

func verifyMsg(msg amqp.Delivery) (body commit.MsgBody, err error) {
	err = json.Unmarshal(msg.Body, &body)
	if err != nil {
		return
	}

	p := body.Payload
	switch body.Type {
	case commit.TypePrepare:
		// check payload is a slice of string
		list, ok := p.([]any)
		if !ok {
			err = ErrInvalidPayload
			return
		}
		for _, ip := range list {
			if _, ok := ip.(string); !ok {
				err = ErrInvalidPayload
				return
			}
		}
	case commit.TypeResp:
		// check if payload is "ok" or "not ok"
		if p != commit.MsgOK && p != commit.MsgNotOK {
			err = ErrInvalidPayload
		}
	default:
		err = ErrInvalidType
	}
	return
}

func newReceiveFn(n *node.Node) notify.ReceiveFunc {
	return func(msg amqp.Delivery) error {
		body, err := verifyMsg(msg)
		if err != nil {
			return err
		}

		p := body.Payload
		switch body.Type {
		case commit.TypePrepare:
			// new two phase commit request
			ips := make([]string, 0)
			for _, ip := range p.([]any) {
				ips = append(ips, ip.(string))
			}
			c := n.NewCommit(body.ID, body.Ip, ips, body.Timeout)

			// detect failure
			for _, d := range n.Detectors {
				ch := d.Detect(c)
				go func() {
					// catch the diff between participants and votes after commit timeout
					diff := <-ch
					for _, ip := range diff.([]string) {
						go n.Abort(context.TODO(), c.ID(), ip)
					}
				}()
			}

			// emit a signal after 1s
			time.AfterFunc(1*time.Second, func() {
				n.Done(context.TODO(), c.ID(), true)
				// testing timeout detector
				// if n.Ip == "0.0.0.0" {
				// 	n.Done(context.TODO(), c.ID(), true)
				// }
			})
		case commit.TypeResp:
			c := n.GetCommit(body.ID)
			if c == nil {
				return ErrCommitNotFound
			}
			c.Vote(body.Ip, p == commit.MsgOK)
		}
		return nil
	}
}

func main() {
	// pretend we have k nodes, the first node will be initiating two phase commit request
	var k = 2
	nodes := make([]*node.Node, 0, k)
	ips := make([]string, 0, k)

	for i := 0; i < k; i++ {
		// only first node will be attaching a timeout detector
		n := newNode(fmt.Sprintf("%d.%d.%d.%d", i, i, i, i), "node1_signal", i == 0, nil)
		defer n.Close()

		nodes = append(nodes, n)
		ips = append(ips, n.Ip)

		fn, err := n.Notifier.(*notify.RabbitMQ).Consume()
		failOnError(err, "Failed to consume msg")

		go fn()
	}

	for _, n := range nodes {
		n.Notifier.(*notify.RabbitMQ).ReceiveFn = newReceiveFn(n)
	}

	// emit a prepare signal
	cid := commit.CommitID{
		UUID: uuid.New(),
	}
	nodes[0].Prepare(context.TODO(), cid, ips, 3*time.Second)
	fmt.Println(<-nodes[0].GetCommit(cid).Ok, <-nodes[1].GetCommit(cid).Ok)
}
