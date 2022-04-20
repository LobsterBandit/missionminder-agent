package main

import (
	"log"
)

func main() {
	log.Println("missionminder-agent starting")

	sv := &SavedVariables{
		File: "MissionMinder.lua",
		Data: make(chan string),
	}

	// read initial sv contents
	go func() {
		data, err := sv.getAddonData()
		if err != nil {
			log.Fatalln(err)
		}
		log.Println("data:", len(data))
	}()

	// listen for new sv data
	go func() {
		for {
			select {
			case data := <-sv.Data:
				log.Println("data:", len(data))
			}
		}
	}()

	// watch sv file for changes
	if err := sv.watch(); err != nil {
		log.Println(err)
	}
}
