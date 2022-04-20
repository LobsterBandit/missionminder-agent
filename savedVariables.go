package main

import (
	"log"
	"os"
	"path"
	"time"

	"github.com/radovskyb/watcher"
)

const (
	svDir = "/wow/_retail_/WTF/Account/2DP3/SavedVariables/"
)

type SavedVariables struct {
	File string
	Data chan string
}

func (sv *SavedVariables) path() string {
	return path.Join(svDir, sv.File)
}

func (sv *SavedVariables) read() (string, error) {
	data, err := os.ReadFile(sv.path())
	return string(data), err
}

func (sv *SavedVariables) watch() error {
	w := watcher.New()

	go func() {
		for {
			select {
			case event := <-w.Event:
				log.Println("event:", event)
				data, err := sv.read()
				if err != nil {
					log.Println("error:", err)
				}
				sv.Data <- data
			case err := <-w.Error:
				log.Println("error:", err)
			case <-w.Closed:
				return
			}
		}
	}()

	if err := w.Add(sv.path()); err != nil {
		return err
	}

	if err := w.Start(time.Millisecond * 100); err != nil {
		return err
	}

	return nil
}
