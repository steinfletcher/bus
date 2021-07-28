package services

import (
	"context"
	"fmt"
	"github.com/steinfletcher/bus/example/models"
)

type Auditor struct {}

func (s *Auditor) getTodos(_ context.Context, query *models.GetTodoByIDQuery) {
	fmt.Println(fmt.Sprintf("Auditor: received todos request: %+v", query))
}
