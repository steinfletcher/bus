package services

import "github.com/steinfletcher/bus"

func Register(bus bus.Bus) {
	todoService := &TodoService{todos: createFakeTodos()}
	bus.MustSubscribe(todoService.getTodoByIDHandler)

	auditor := &Auditor{}
	bus.MustSubscribeAsync(auditor.getTodos)
}
