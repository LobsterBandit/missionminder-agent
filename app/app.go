package app

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/lobsterbandit/missionminder-agent/addon"
	"github.com/lobsterbandit/missionminder-agent/timer"
)

const RefreshSeconds int64 = 30

func Run(ctx context.Context, cancel context.CancelFunc) error {
	sv := addon.NewSV("MissionMinder.lua")

	// force trigger file write events on a timer
	// go triggerWrites(ctx, sv, time.Second*49) //nolint:gomnd

	// listen for changes and refresh mission timers
	go refreshLoop(ctx, sv)

	// read initial sv contents on startup
	go printData(sv, cancel)

	// watch sv file for changes
	if err := sv.Watch(ctx); err != nil {
		return fmt.Errorf("error running app: %w", err)
	}

	return nil
}

func refreshLoop(ctx context.Context, sv *addon.SavedVariables) {
	refreshDuration := time.Second * time.Duration(RefreshSeconds)
	rt := timer.NewRefreshTimer(refreshDuration)

	for {
		rt.Next()
		select {
		// listen for sv data
		case data := <-sv.Data:
			log.Println("new data received")
			rt.Stop()
			data.Print()
		// refresh every X seconds to recalculate times
		case <-rt.C:
			log.Println("refreshing calculations")
			sv.Current.Print()
		case <-ctx.Done():
			log.Printf("exiting refresh loop: %v\n", ctx.Err())
			rt.Stop()

			return
		}
	}
}

func printData(sv *addon.SavedVariables, cancel context.CancelFunc) {
	if err := sv.LoadAddonData(); err != nil {
		log.Println(err)
		cancel()

		return
	}

	sv.Current.Print()
}

func triggerWrites(ctx context.Context, sv *addon.SavedVariables, d time.Duration) {
	for {
		select {
		case <-time.After(d):
			sv.TriggerWrite()
		case <-ctx.Done():
			log.Printf("exiting event trigger: %v\n", ctx.Err())

			return
		}
	}
}
