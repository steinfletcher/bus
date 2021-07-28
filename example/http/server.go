package http

import (
	"errors"
	"fmt"
	"github.com/labstack/echo/v4"
	"github.com/steinfletcher/bus"
	"github.com/steinfletcher/bus/example/models"
	"net/http"
)

type Server struct {
	Router *echo.Echo
	msgBus bus.Bus
}

func NewServer(msgBus bus.Bus) *Server {
	server := Server{Router: echo.New(), msgBus: msgBus}

	server.Router.GET("/todo/:id", server.getTodo)

	return &server
}

func (s *Server) Start(addr string) error {
	return s.Router.Start(addr)
}

func (s *Server) getTodo(ctx echo.Context) error {
	query := models.GetTodoByIDQuery{
		ID: ctx.Param("id"),
	}

	err := s.msgBus.Publish(ctx.Request().Context(), &query)
	if err != nil {
		if errors.Is(models.ErrTodoNotFound, err) {
			return ctx.JSON(http.StatusBadRequest, errorDTO{Message: fmt.Sprintf("no todo with id=%s", query.ID)})
		}
		return ctx.JSON(http.StatusInternalServerError, errorDTO{Message: "something failed"})
	}

	return ctx.JSON(http.StatusOK, query.Result)
}

type errorDTO struct {
	Message string `json:"error"`
}
