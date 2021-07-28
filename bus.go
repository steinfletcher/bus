package bus

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"sync"
)

const defaultAsyncHandlerQueueSize = 1000

var ErrHandlerNotFound = errors.New("handler not found")

type handler struct {
	Handler reflect.Value
	isAsync bool
	queue   chan []reflect.Value
}

type Message interface{}

type Subscriber interface {
	Subscribe(fn interface{}) error
	MustSubscribe(fn interface{})
	SubscribeAsync(fn interface{}) error
	MustSubscribeAsync(fn interface{})
}

type Publisher interface {
	Publish(ctx context.Context, msg Message) error
}

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

type Bus interface {
	Subscriber
	Publisher
}

type eventBus struct {
	handlers  *handlers
	queueSize int
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
