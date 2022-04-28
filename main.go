package main

import (
	"context"
	"log"
	"net/http"
	_ "net/http/pprof"
	"time"
)

const RECALC_SECONDS int64 = 30

func flushTimer(t *time.Timer) {
	log.Println("stopping any existing refresh timer")
	if !t.Stop() {
		log.Println("refresh timer already stopped or expired, draining channel")
		<-t.C
	}
}

func main() {
	log.Println("missionminder-agent starting")

	sv := &SavedVariables{
		File: "MissionMinder.lua",
		Data: make(chan *AddonData),
	}

	ctx, cancel := context.WithCancel(context.Background())
	go func(ctx context.Context) {
		// read initial sv contents on startup
		data, err := sv.getAddonData()
		if err != nil {
			log.Fatalln(err)
		}
		data.print()

		refreshDuration := time.Second * time.Duration(RECALC_SECONDS)
		log.Println("starting refresh timer:", refreshDuration)
		refreshTimer := time.NewTimer(refreshDuration)
		// defer refreshTimer.Stop()
		for {
			// log.Println("resetting refresh timer to", refreshDuration)
			// refreshTimer.Reset(refreshDuration)
			select {
			// listen for sv data
			case data = <-sv.Data:
				log.Println("new data received")
				data.print()
				flushTimer(refreshTimer)
			// refresh every X seconds to recalculate times
			case <-refreshTimer.C:
				// case <-time.After(refreshDuration):
				log.Println("refreshing calculations")
				data.print()
			case <-ctx.Done():
				log.Println("received cancel signal")
				flushTimer(refreshTimer)
				return
			}
			log.Println("resetting refresh timer:", refreshDuration)
			refreshTimer.Reset(refreshDuration)
		}
	}(ctx)

	// setup pprof listener
	go func() {
		log.Println(http.ListenAndServe(":8081", nil))
	}()

	// watch sv file for changes
	if err := sv.watch(); err != nil {
		log.Println(err)
		cancel()
	}

	log.Println("missionminder-agent exiting...")
}
