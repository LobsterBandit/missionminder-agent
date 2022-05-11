package main

import (
	"context"
	"log"
	"os"
	"os/signal"

	"github.com/lobsterbandit/missionminder-agent/app"
)

// main parses and validates configuration and then starts the application.
func main() {
	log.Println("missionminder-agent starting")

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
	end := make(chan struct{})

	go func() {
		<-ctx.Done()
		log.Printf("shutting down: %v\n", ctx.Err())
		close(end)
	}()

	if err := app.RunPrintLoop(ctx, cancel); err != nil {
		log.Println(err)
		cancel()
	}

	<-end
	log.Println("missionminder-agent exiting...")
}
