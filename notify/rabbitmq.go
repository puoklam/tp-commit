package notify

import (
	"context"
	"fmt"

	"github.com/puoklam/tp-commit/internal/sync"
	amqp "github.com/rabbitmq/amqp091-go"
)

type BindFunc func(prefix string) string
type ReceiveFunc func(msg amqp.Delivery) error

var DefaultBindFunc BindFunc = func(prefix string) string {
	return fmt.Sprintf("%s.%s", prefix, "*")
}

var ReceiveNop ReceiveFunc = func(msg amqp.Delivery) error {
	return nil
}

type RabbitMQ struct {
	once      sync.Once
	Exchange  string           // exchange name
	Queue     string           // queue name
	Prefix    string           // routing key prefix
	Key       string           // routing key suffix
	BindFn    BindFunc         // binding func
	ReceiveFn ReceiveFunc      // message receiving func
	Conn      *amqp.Connection // rabbitmq connection
	ch        *amqp.Channel
	q         amqp.Queue
}

func (r *RabbitMQ) lazyInit() (err error) {
	r.once.Do(func() {
		r.ch, err = r.Conn.Channel()
		if r.BindFn == nil {
			r.BindFn = DefaultBindFunc
		}
		err = r.declare()
		if err != nil {
			return
		}
		err = r.bind()
	})
	return
}

func (r *RabbitMQ) declare() error {
	// Declare exchange
	err := r.ch.ExchangeDeclare(
		r.Exchange,
		"topic",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return err
	}

	// Declare queue
	r.q, err = r.ch.QueueDeclare(
		r.Queue, // name
		false,   // durable
		false,   // delete when unused
		true,    // exclusive
		false,   // no-wait
		nil,     // arguments
	)

	return err
}

func (r *RabbitMQ) bind() error {
	// Bind queue
	bindKey := r.BindFn(r.Prefix)
	return r.ch.QueueBind(
		r.q.Name,
		bindKey,
		r.Exchange,
		false,
		nil,
	)
}

func (r *RabbitMQ) Init(reset bool) error {
	if reset {
		r.once.Reset()
	}
	return r.lazyInit()
}

func (r *RabbitMQ) Publish(ctx context.Context, body []byte) error {
	if err := r.lazyInit(); err != nil {
		return err
	}

	if ctx == nil {
		ctx = context.Background()
	}
	err := r.ch.PublishWithContext(
		ctx,
		r.Exchange,
		r.Prefix+"."+r.Key,
		false,
		false,
		amqp.Publishing{
			ContentType: "text/plain",
			Body:        body,
		},
	)
	return err
}

func (r *RabbitMQ) Consume() (func() error, error) {
	if err := r.lazyInit(); err != nil {
		return nil, err
	}

	msgs, err := r.ch.Consume(
		r.q.Name,
		"",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return nil, err
	}

	fn := func() error {
		for msg := range msgs {
			if err := r.Receive(msg); err != nil {
				return err
			}
		}
		return nil
	}

	return fn, nil
}

func (r *RabbitMQ) Emit(ctx context.Context, msg any) error {
	return r.Publish(ctx, msg.([]byte))
}

func (r *RabbitMQ) Receive(msg any) error {
	if r.ReceiveFn != nil {
		return r.ReceiveFn(msg.(amqp.Delivery))
	}
	return nil
}

func (r *RabbitMQ) Close() error {
	if r.ch != nil {
		r.ch.Close()
	}
	if r.Conn == nil {
		return nil
	}
	return r.Conn.Close()
}
