package bus_test

import (
	"context"
	"errors"
	"fmt"
	"github.com/steinfletcher/bus"
	"github.com/stretchr/testify/assert"
	"reflect"
	"sync"
	"testing"
	"time"
)

type GetUserQuery struct {
	ID     string
	Result UserResult
}

type SomeCommand struct {
	ID string
}

type UserResult struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

func TestBus(t *testing.T) {
	b := bus.New()

	handler := func(ctx context.Context, query *GetUserQuery) error {
		assert.Equal(t, "1234", query.ID)
		query.Result = UserResult{
			Name:  "Jan",
			Email: "jan@hey.com",
		}
		return nil
	}
	_ = b.Subscribe(handler)

	query := GetUserQuery{ID: "1234"}
	_ = b.Publish(context.Background(), &query)

	assert.Equal(t, GetUserQuery{
		ID: "1234",
		Result: UserResult{
			Name:  "Jan",
			Email: "jan@hey.com",
		},
	}, query)
}

func TestBus_InvalidHandler(t *testing.T) {
	tests := map[string]struct {
		handlerFunc interface{}
		errContains string
	}{
		"not a function": {
			handlerFunc: "not a func",
			errContains: "'string' is not a function",
		},
		"invalid args": {
			handlerFunc: func(ctx context.Context) {},
			errContains: "invalid number of handler arguments",
		},
		"first arg must be context": {
			handlerFunc: func(a string, b string) {},
			errContains: "first argument must be context.Context",
		},
		"success": {
			handlerFunc: func(ctx context.Context, arg *GetUserQuery) {},
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			b := bus.New()
			err := b.Subscribe(test.handlerFunc)
			if test.errContains != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), test.errContains)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestBus_MultipleSubscribers(t *testing.T) {
	b := bus.New()
	var handler1Invoked bool
	var handler2Invoked bool

	handler1 := func(ctx context.Context, query *GetUserQuery) error {
		handler1Invoked = true
		return nil
	}
	handler2 := func(ctx context.Context, query *GetUserQuery) error {
		handler2Invoked = true
		return nil
	}
	_ = b.Subscribe(handler1)
	_ = b.Subscribe(handler2)

	query := GetUserQuery{ID: "1234"}
	_ = b.Publish(context.Background(), &query)

	assert.True(t, handler1Invoked)
	assert.True(t, handler2Invoked)
}

func TestBus_PreservesContext(t *testing.T) {
	b := bus.New()

	handler := func(ctx context.Context, query *GetUserQuery) error {
		assert.Equal(t, "value", ctx.Value("key"))
		return nil
	}
	_ = b.Subscribe(handler)

	query := GetUserQuery{ID: "1234"}
	ctx := context.Background()
	ctx = context.WithValue(ctx, "key", "value")
	err := b.Publish(ctx, &query)

	assert.NoError(t, err)
}

func TestBus_HandlerNotFound(t *testing.T) {
	b := bus.New()

	query := GetUserQuery{ID: "1234"}
	err := b.Publish(context.Background(), &query)

	assert.EqualError(t, err, "handler not found")
}

func TestBus_HandlerError(t *testing.T) {
	b := bus.New()

	handler := func(ctx context.Context, query *GetUserQuery) error {
		return errors.New("failed to get user")
	}
	_ = b.Subscribe(handler)

	query := GetUserQuery{ID: "1234"}
	err := b.Publish(context.Background(), &query)

	assert.EqualError(t, err, "failed to get user")
}

func TestBus_SubscribeAsync_DoesNotRecordError(t *testing.T) {
	b := bus.New()

	handler := func(ctx context.Context, query *GetUserQuery) error {
		return errors.New("failed to get user")
	}
	_ = b.SubscribeAsync(handler)

	ctx := context.Background()
	ctx = context.WithValue(ctx, "key", "value")
	query := GetUserQuery{ID: "1234"}
	err := b.Publish(ctx, &query)

	time.Sleep(time.Millisecond * 200)
	assert.NoError(t, err)
}

func TestBus_MultipleAsyncHandlers(t *testing.T) {
	b := bus.New()
	wg := sync.WaitGroup{}
	wg.Add(2)
	var handler1Invoked bool
	var handler2Invoked bool
	var handlerDifferentTypeInvoked bool

	handler1 := func(ctx context.Context, query *GetUserQuery) {
		defer wg.Done()
		handler1Invoked = true
	}
	handler2 := func(ctx context.Context, query *GetUserQuery) {
		defer wg.Done()
		handler2Invoked = true
	}
	handlerDifferentType := func(ctx context.Context, command SomeCommand) {
		handlerDifferentTypeInvoked = true
	}
	_ = b.SubscribeAsync(handler1)
	_ = b.SubscribeAsync(handler2)
	_ = b.SubscribeAsync(handlerDifferentType)

	query := GetUserQuery{ID: "1234"}
	err := b.Publish(context.Background(), &query)

	wg.Wait()
	assert.True(t, handler1Invoked)
	assert.True(t, handler2Invoked)
	assert.False(t, handlerDifferentTypeInvoked)
	assert.NoError(t, err)
}

func TestBus_SyncAndAsyncHandlers(t *testing.T) {
	b := bus.New()
	wg := sync.WaitGroup{}
	wg.Add(1)
	var asyncInvoked bool

	handler := func(ctx context.Context, query *GetUserQuery) error {
		return nil
	}
	handlerAsync := func(ctx context.Context, query *GetUserQuery) {
		asyncInvoked = true
		wg.Done()
	}
	_ = b.Subscribe(handler)
	_ = b.SubscribeAsync(handlerAsync)

	query := GetUserQuery{ID: "1234"}
	err := b.Publish(context.Background(), &query)

	wg.Wait()

	assert.NoError(t, err)
	assert.True(t, asyncInvoked)
}

func TestBus_SyncAndAsyncHandlers_CallsAsyncWhenSyncFails(t *testing.T) {
	b := bus.New()
	var asyncInvoked bool
	wg := sync.WaitGroup{}
	wg.Add(1)

	handler := func(ctx context.Context, query *GetUserQuery) error {
		return errors.New("some error")
	}
	handlerAsync := func(ctx context.Context, query *GetUserQuery) {
		defer wg.Done()
		asyncInvoked = true
	}
	_ = b.Subscribe(handler)
	_ = b.SubscribeAsync(handlerAsync)

	query := GetUserQuery{ID: "1234"}
	err := b.Publish(context.Background(), &query)

	wg.Wait()

	assert.Error(t, err)
	assert.True(t, asyncInvoked)
}

func TestBus_MultipleSyncHandlers_PreventsFutureHandlersOnError(t *testing.T) {
	b := bus.New()
	var handler1Invoked bool
	var handler2Invoked bool

	handler1 := func(ctx context.Context, query *GetUserQuery) error {
		handler1Invoked = true
		return errors.New("some error")
	}
	handler2 := func(ctx context.Context, query *GetUserQuery) error {
		handler2Invoked = true
		return nil
	}
	_ = b.Subscribe(handler1)
	_ = b.Subscribe(handler2)

	query := GetUserQuery{ID: "1234"}
	err := b.Publish(context.Background(), &query)

	assert.EqualError(t, err, "some error")
	assert.True(t, handler1Invoked)
	assert.False(t, handler2Invoked)
}

func Test(t *testing.T) {
	fn := func(ctx context.Context, arg *SomeCommand) {
		fmt.Println(reflect.TypeOf(arg).String())
	}
	handlerArg := reflect.TypeOf(fn).In(1).String()
	fmt.Println(handlerArg)
	fn(context.Background(), &SomeCommand{})
}
