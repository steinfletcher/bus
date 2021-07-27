package bus

import (
	"context"
	"errors"
	"reflect"
)

var ErrHandlerNotFound = errors.New("handler not found")

type Handler interface{}

type Message interface{}

type Subscriber interface {
	Subscribe(Handler)
}

type Publisher interface {
	Publish(ctx context.Context, msg Message) error
}

func New() Bus {
	return &eventBus{
		handlers:      make(map[string][]Handler),
	}
}

type Bus interface {
	Subscriber
	Publisher
}

type eventBus struct {
	handlers map[string][]Handler
}

func (e *eventBus) Subscribe(handler Handler) {
	handlerArg := reflect.TypeOf(handler).In(1)
	handlerArgTypeName := handlerArg.Elem().Name()
	e.handlers[handlerArgTypeName] = append(e.handlers[handlerArgTypeName], handler)
}

func (e *eventBus) Publish(ctx context.Context, msg Message) error {
	msgTypeName := reflect.TypeOf(msg).Elem().Name()
	handlers, ok := e.handlers[msgTypeName]
	if !ok {
		return ErrHandlerNotFound
	}

	var params = []reflect.Value{}
	params = append(params, reflect.ValueOf(ctx))
	params = append(params, reflect.ValueOf(msg))
	for _, handler := range handlers {
		result := reflect.ValueOf(handler).Call(params)
		if err := result[0].Interface(); err != nil {
			return err.(error)
		}
	}
	return nil
}
