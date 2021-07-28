package services

import (
	"context"
	"github.com/steinfletcher/bus/example/models"
)

type TodoService struct {
	todos map[string]models.Todo
}

func (s *TodoService) getTodoByIDHandler(_ context.Context, query *models.GetTodoByIDQuery) error {
	if todo, ok := s.todos[query.ID]; ok {
		query.Result = todo
		return nil
	}
	return models.ErrTodoNotFound
}

func createFakeTodos() map[string]models.Todo{
	todos := make(map[string]models.Todo)
	todos["1234"] = models.Todo{
		ID:    "1234",
		Title: "Buy Milk",
	}
	return todos
}
