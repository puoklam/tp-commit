# tp-commit
A two-phase commit protocol Golang implementation

note that this package just reduces the chance of getting inconsistent state among different server, there is still posibility to fail transaction atomicity

see example at [examples/rabbitmq](examples/rabbitmq)

You can also write your own notifier e.g. [rabbitmq](notify/rabbitmq.go)