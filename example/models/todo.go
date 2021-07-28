package models

import "errors"

var ErrTodoNotFound = errors.New("todo not found")

type GetTodoByIDQuery struct {
	ID     string
	Result Todo
}

type Todo struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}
