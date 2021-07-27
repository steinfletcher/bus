# bus

A simple synchronous message bus.

```go
msgBus := bus.New()
```

## Publish

Publish a message to the bus.

```go
bus.Publish(context.Background(), Message{Content: "hello"})
```

## Subscribe

Subscribe to a message by its type.

```go
bus.Subscribe(func(ctx context.Context, message *Message) error {
    message.Result = "1234"
    return nil
})
```
