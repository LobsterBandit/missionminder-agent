package app

import (
	"context"
	"fmt"
	"log"
	"sort"
	"time"

	"github.com/fatih/color"
	"github.com/lobsterbandit/missionminder-agent/addon"
	"github.com/lobsterbandit/missionminder-agent/timer"
)

const (
	RefreshSeconds      int64 = 30
	MaxShowNextComplete int   = 3 // max number of missions to show in list of next to complete
)

func RunPrintLoop(ctx context.Context, cancel context.CancelFunc) error {
	sv := addon.NewSV("MissionMinder.lua")

	// force trigger file write events on a timer
	// go triggerWrites(ctx, sv, time.Second*49) //nolint:gomnd

	// listen for changes and refresh mission timers
	go refreshLoop(ctx, sv)

	// read initial sv contents on startup
	go readAndPrintOnce(sv, cancel)

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
			printData(data)
		// refresh every X seconds to recalculate times
		case <-rt.C:
			log.Println("refreshing calculations")
			printData(sv.Current)
		case <-ctx.Done():
			log.Printf("exiting refresh loop: %v\n", ctx.Err())
			rt.Stop()

			return
		}
	}
}

func readAndPrintOnce(sv *addon.SavedVariables, cancel context.CancelFunc) {
	if err := sv.LoadAddonData(); err != nil {
		log.Println(err)
		cancel()

		return
	}

	printData(sv.Current)
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

func printData(d *addon.Data) {
	log.Printf("%d characters, %d complete / %d active\n",
		len(d.Characters),
		d.NumMissionsComplete(addon.FollowerType_9_0),
		d.NumMissionsActive(addon.FollowerType_9_0))

	for _, key := range d.CharacterKeys() {
		char := d.Characters[key]
		table := char.Table(addon.FollowerType_9_0)
		if table == nil || len(table.Followers) == 0 {
			continue
		}

		missions := table.MissionsActive()
		log.Printf("\t%-30s M(%-2s / %2d) F(%-2s / %2d)\n",
			color.CyanString("%-30s", char.String()),
			color.GreenString("%2d", len(table.MissionsComplete())),
			len(missions),
			color.YellowString("%2d", len(table.IdleCompanions())),
			table.NumCompanions())

		sort.Slice(missions, func(i, j int) bool { return missions[i].MissionEndTime < missions[j].MissionEndTime })
		var shown int
		for _, m := range missions {
			// skip completed missions
			if m.IsComplete() {
				continue
			}

			log.Printf("\t\t- %11s    (%d) [%2d] %-35s %s\n",
				m.TimeRemaining(),
				len(table.CompanionsOnMission(m)),
				m.MissionScalar,
				m.Name,
				m.BonusReward())

			shown++
			// break loop if we're at max list length
			if shown >= MaxShowNextComplete {
				break
			}
		}
	}
}
