package main

import (
	"github.com/steinfletcher/bus"
	"github.com/steinfletcher/bus/example/http"
	"github.com/steinfletcher/bus/example/services"
	"log"
)

func main() {
	msgBus := bus.New()
	services.Register(msgBus)
	log.Fatal(http.NewServer(msgBus).Start(":8081"))
}
