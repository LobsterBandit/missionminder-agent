package main

import (
	"log"
	"time"
)

const RECALC_SECONDS int64 = 30

func main() {
	log.Println("missionminder-agent starting")

	sv := &SavedVariables{
		File: "MissionMinder.lua",
		Data: make(chan *AddonData),
	}

	go func() {
		// read initial sv contents on startup
		data, err := sv.getAddonData()
		if err != nil {
			log.Fatalln(err)
		}
		data.print()

		for {
			select {
			// listen for sv data
			case data = <-sv.Data:
				data.print()
			// refresh every X seconds to recalculate times
			case <-time.After(time.Second * time.Duration(RECALC_SECONDS)):
				data.print()
			}
		}
	}()

	// watch sv file for changes
	if err := sv.watch(); err != nil {
		log.Println(err)
	}
}
