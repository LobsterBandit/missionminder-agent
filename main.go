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
		log.Printf("data: %d characters, %d complete / %d active\n",
			len(data.Characters),
			data.totalMissionsComplete(),
			data.totalMissionsActive())

		for key, char := range data.Characters {
			log.Printf("\t%s: %d / %d\n",
				key,
				len(data.missionsComplete(char)),
				len(data.missionsActive(char)))
		}
	}()

	// listen for new sv data
	go func() {
		for {
			select {
			case data := <-sv.Data:
				log.Printf("data: %d characters, %d complete / %d active\n",
					len(data.Characters),
					data.totalMissionsComplete(),
					data.totalMissionsActive())

				for key, char := range data.Characters {
					log.Printf("\t%s: %d / %d\n",
						key,
						len(data.missionsComplete(char)),
						len(data.missionsActive(char)))
				}
			}
		}
	}()

	// watch sv file for changes
	if err := sv.watch(); err != nil {
		log.Println(err)
	}
}
