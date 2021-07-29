package bus

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"sync"
)

// Bus exposes the Subscriber and Publisher and is the main interface used to interact with the message bus.
//
// Define a new message bus like so
//
// msgBus := bus.New()
//
// Subscribe to a message
//
// msgBus.Subscribe(func(ctx context.Context, message Message) error {
//    message.Result = "1234"
//    return nil
// })
//
// Then publish a message to the bus
//
// msgBus.Publish(context.Background(), Message{Content: "hello"})
type Bus interface {
	Subscriber
	Publisher
}

// Subscriber listens to events published to the bus. Use Subscribe to listen to events synchronously and
// SubscribeAsync to listen to events asynchronously. The Must methods simplify subscription but panic internally
// if there are no subscribers, therefore these methods should only be used for defining static relationships, not
// dynamic relationships defined at runtime.
// The Message type must match the handler subscriber type. Pointer and
// non-pointer messages are considered as separate types - internally subscribers are keyed using the message type which
// includes a pointer symbol in the lookup key.
type Subscriber interface {
	// Subscribe is used to listen to events synchronously
	Subscribe(fn interface{}) error

	// MustSubscribe is used to listen to events synchronously. This method simplifies subscription but panics internally
	// if there are no subscribers. It is recommended to only use this for defining static relationships rather than
	// dynamic relationships defined at runtime
	MustSubscribe(fn interface{})

	// SubscribeAsync is used to listen to events asynchronously. Subscribers are run in a separate go routine and data
	// is passed into the subscriber via a channel
	SubscribeAsync(fn interface{}) error

	// MustSubscribeAsync is used to listen to events asynchronously. This method simplifies subscription but panics internally
	// if there are no subscribers. It is recommended to only use this for defining static relationships rather than
	// dynamic relationships defined at runtime
	MustSubscribeAsync(fn interface{})
}

// Publisher publishes an event to the bus. The Message type must match the handler subscriber type. Pointer and
//// non-pointer messages are considered as separate types - internally subscribers are keyed using the message type which
//// includes a pointer symbol in the lookup key.
type Publisher interface {
	Publish(ctx context.Context, msg Message) error
}

// ErrHandlerNotFound is returned when publishing an event that does not have any subscribers
var ErrHandlerNotFound = errors.New("handler not found")

// Message the data that is published. The implementing type is used as the handler key
type Message interface{}

// New create a new message bus.
func New(queueSize ...int) Bus {
	handlers := newHandlers()
	size := defaultAsyncHandlerQueueSize
	if len(queueSize) > 0 {
		size = queueSize[0]
	}
	return &eventBus{
		handlers:  handlers,
		queueSize: size,
	}
}

type eventBus struct {
	handlers  *handlers
	queueSize int
}

type handler struct {
	Handler reflect.Value
	isAsync bool
	queue   chan []reflect.Value
}

func (e *eventBus) Subscribe(fn interface{}) error {
	return e.subscribe(fn, false)
}

func (e *eventBus) MustSubscribe(fn interface{}) {
	if err := e.subscribe(fn, false); err != nil {
		panic(err)
	}
}

func (e *eventBus) SubscribeAsync(fn interface{}) error {
	return e.subscribe(fn, true)
}

func (e *eventBus) MustSubscribeAsync(fn interface{}) {
	if err := e.subscribe(fn, true); err != nil {
		panic(err)
	}
}

func (e *eventBus) subscribe(fn interface{}, isAsync bool) error {
	if err := validateHandler(fn); err != nil {
		return err
	}
	handlerArgTypeName := reflect.TypeOf(fn).In(1).String()
	handler := handler{
		Handler: reflect.ValueOf(fn),
		isAsync: isAsync,
	}
	if isAsync {
		handler.queue = make(chan []reflect.Value, defaultAsyncHandlerQueueSize)
		go func() {
			for params := range handler.queue {
				handler.Handler.Call(params)
			}
		}()
	}
	e.handlers.Add(handlerArgTypeName, handler)
	return nil
}

func (e *eventBus) Publish(ctx context.Context, msg Message) error {
	msgTypeName := reflect.TypeOf(msg).String()
	_, ok := e.handlers.Get(msgTypeName)
	if !ok {
		return ErrHandlerNotFound
	}

	var params = []reflect.Value{}
	params = append(params, reflect.ValueOf(ctx))
	params = append(params, reflect.ValueOf(msg))

	// dispatch async handlers first
	for messageHandlers := range e.handlers.Iter() {
		if messageHandlers.Key == msgTypeName {
			for _, handler := range messageHandlers.Value {
				if handler.isAsync {
					handler.queue <- params
				}
			}
		}
	}

	// handle sync handlers. If a handler errors we end the chain
	for messageHandlers := range e.handlers.Iter() {
		if messageHandlers.Key == msgTypeName {
			for _, handler := range messageHandlers.Value {
				isSync := !handler.isAsync
				if isSync {
					result := handler.Handler.Call(params)
					if err := result[0].Interface(); err != nil {
						return err.(error)
					}
				}
			}
		}
	}

	return nil
}

func validateHandler(fn interface{}) error {
	typeOf := reflect.TypeOf(fn)
	if typeOf.Kind() != reflect.Func {
		return fmt.Errorf("'%s' is not a function", typeOf)
	}
	if typeOf.NumIn() < 2 {
		return errors.New("invalid number of handler arguments. Must be context.Context followed by a struct")
	}
	if typeOf.In(0).String() != "context.Context" {
		return errors.New("first argument must be context.Context")
	}
	return nil
}

type handlers struct {
	sync.RWMutex
	items map[string][]handler
}

type handlerItem struct {
	Key   string
	Value []handler
}

func newHandlers() *handlers {
	cm := &handlers{
		items: make(map[string][]handler),
	}
	return cm
}

func (cm *handlers) Add(key string, value handler) {
	cm.Lock()
	defer cm.Unlock()
	cm.items[key] = append(cm.items[key], value)
}

func (cm *handlers) Get(key string) ([]handler, bool) {
	cm.Lock()
	defer cm.Unlock()
	value, ok := cm.items[key]
	return value, ok
}

func (cm *handlers) Iter() <-chan handlerItem {
	c := make(chan handlerItem)
	f := func() {
		cm.Lock()
		defer cm.Unlock()
		for k, v := range cm.items {
			c <- handlerItem{k, v}
		}
		close(c)
	}
	go f()
	return c
}

const defaultAsyncHandlerQueueSize = 1000
