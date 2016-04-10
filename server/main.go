package main

import (
	"net/http"

	log "github.com/Sirupsen/logrus"
	"github.com/bbokorney/dockworker"
)

func main() {
	wsContainer := dockworker.InitWSContainer()
	log.Info("Starting up...")
	log.Fatal(http.ListenAndServe(":4321", wsContainer))
}
