package bus_test

import (
	"context"
	"errors"
	"github.com/steinfletcher/bus"
	"github.com/stretchr/testify/assert"
	"testing"
)

type GetUserQuery struct {
	ID     string
	Result UserResult
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
	b.Subscribe(handler)

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
	b.Subscribe(handler1)
	b.Subscribe(handler2)

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
	b.Subscribe(handler)

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
	b.Subscribe(handler)

	query := GetUserQuery{ID: "1234"}
	err := b.Publish(context.Background(), &query)

	assert.EqualError(t, err, "failed to get user")
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
	b.Subscribe(handler1)
	b.Subscribe(handler2)

	query := GetUserQuery{ID: "1234"}
	err := b.Publish(context.Background(), &query)

	assert.EqualError(t, err, "some error")
	assert.True(t, handler1Invoked)
	assert.False(t, handler2Invoked)
}
