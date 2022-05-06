package main

import (
	"context"
	"log"
	"net/http"
	_ "net/http/pprof" //nolint:gosec
	"os"
	"os/signal"
	"time"
)

const RefreshSeconds int64 = 30

func flushTimer(t *time.Timer) {
	log.Println("stopping any existing refresh timer")

	if !t.Stop() {
		log.Println("refresh timer already stopped or expired, draining channel")
		<-t.C
	}
}

//nolint:funlen
func main() {
	log.Println("missionminder-agent starting")

	sv := NewSV("MissionMinder.lua")

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
	done := make(chan bool, 1)

	go func() {
		defer close(done)

		refreshDuration := time.Second * time.Duration(RefreshSeconds)
		log.Println("starting refresh timer:", refreshDuration)
		refreshTimer := time.NewTimer(refreshDuration)

		for {
			select {
			// listen for sv data
			case data := <-sv.Data:
				log.Println("new data received")
				flushTimer(refreshTimer)
				data.print()
			// refresh every X seconds to recalculate times
			case <-refreshTimer.C:
				log.Println("refreshing calculations")
				sv.Current.print()
			case <-ctx.Done():
				log.Printf("exiting refresh loop: %v\n", ctx.Err())
				flushTimer(refreshTimer)

				return
			}
			log.Println("resetting refresh timer:", refreshDuration)
			refreshTimer.Reset(refreshDuration)
		}
	}()

	// setup pprof listener
	go func() {
		log.Println(http.ListenAndServe(":8081", nil))
	}()

	closed := make(chan bool, 1)

	go func() {
		<-ctx.Done()
		log.Printf("terminating: %v\n", ctx.Err())
		sv.watcher.Close()
		<-done
		log.Println("watcher closed")
		close(closed)
	}()

	go func() {
		// read initial sv contents on startup
		if err := sv.loadAddonData(); err != nil {
			log.Println(err)
			cancel()

			return
		}

		sv.Current.print()
	}()

	// watch sv file for changes
	if err := sv.watch(); err != nil {
		log.Println(err)
		cancel()
	}

	<-closed
	log.Println("missionminder-agent exiting...")
}
