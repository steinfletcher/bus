# bus

**Warning: This hasn't been used in anger, so I don't recommend using it in a production application just yet.**

Simple message bus that supports synchronous and asynchronous message processing. It is intended to be used within a single application to help implement loosely coupled components. 

```go
msgBus := bus.New()
```

## Publish

Publish a message to the bus.

```go
msgBus.Publish(context.Background(), Message{Content: "hello"})
```

## Subscribe

Subscribe to a message by its type.

```go
msgBus.Subscribe(func(ctx context.Context, message Message) error {
    message.Result = "1234"
    return nil
})
```

The is synchronous, so the subscriber can mutate the message if the publisher needs a return value, e.g.

```go
b := bus.New()
b.Subscribe(func(ctx context.Context, query *GetUserQuery) {
    query.Result = UserResult{
        Name:  "Jan",
        Email: "jan@hey.com",
    }
})

query := GetUserQuery{ID: "1234"}
b.Publish(context.Background(), &query)

fmt.Println(query.Result.Name) // prints Jan 
```

## SubscribeAsync

Subscribe to a message by its type. The handler is invoked in a separate go routine and doesn't block the calling go routine.

```go
msgBus.SubscribeAsync(func(ctx context.Context, message Message) {
    fmt.Println(message.Content) 
})
```
