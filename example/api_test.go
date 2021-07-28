package main_test

import (
	"github.com/steinfletcher/apitest"
	"github.com/steinfletcher/bus"
	appHttp "github.com/steinfletcher/bus/example/http"
	"github.com/steinfletcher/bus/example/services"
	"net/http"
	"testing"
)

func TestGetTodoByID(t *testing.T) {
	test().Get("/todo/1234").
		Expect(t).
		Status(http.StatusOK).
		Body(`{"id": "1234", "title": "Buy Milk"}`).
		End()
}

func TestGetTodoByID_TodoNotFound(t *testing.T) {
	test().Get("/todo/12345").
		Expect(t).
		Status(http.StatusBadRequest).
		Body(`{"error": "no todo with id=12345"}`).
		End()
}

func test() *apitest.APITest {
	b := bus.New()
	services.Register(b)
	handler := appHttp.NewServer(b).Router
	return apitest.New().Handler(handler)
}
