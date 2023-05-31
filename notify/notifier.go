package notify

import "context"

type Notifier interface {
	Emit(ctx context.Context, msg any) error
	Receive(msg any) error
	Close() error
}
