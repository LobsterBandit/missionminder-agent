package main

import (
	"log"

	"github.com/fsnotify/fsnotify"
)

const svPath = "/wow/_retail_/WTF/Account/2DP3/SavedVariables/MissionMinder.lua"

func main() {
	log.Println("missionminder-agent starting")

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	defer watcher.Close()

	done := make(chan bool)
	go func() {
		for {
			select {
			case event := <-watcher.Events:
				log.Printf("event:%s,%s", event.Name, event.Op)
			case err := <-watcher.Errors:
				log.Println("error:", err)
			}
		}
	}()

	err = watcher.Add(svPath)
	if err != nil {
		log.Fatal(err)
	}

	<-done
}
