package main

import (
	"log"
)

func main() {
	log.Println("missionminder-agent starting")

	sv := &SavedVariables{
		File: "MissionMinder.lua",
		Data: make(chan *AddonData),
	}

	// read initial sv contents
	go func() {
		data, err := sv.getAddonData()
		if err != nil {
			log.Fatalln(err)
		}
		log.Printf("data: %d characters, %d active missions, %d available missions\n",
			len(data.Characters),
			data.numMissionsActive(),
			data.numMissionsAvailable())
	}()

	// listen for new sv data
	go func() {
		for {
			select {
			case data := <-sv.Data:
				log.Printf("data: %d characters, %d active missions, %d available missions\n",
					len(data.Characters),
					data.numMissionsActive(),
					data.numMissionsAvailable())
			}
		}
	}()

	// watch sv file for changes
	if err := sv.watch(); err != nil {
		log.Println(err)
	}
}
