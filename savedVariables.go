package main

import (
	"bytes"
	"compress/zlib"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"time"

	"github.com/radovskyb/watcher"
)

const (
	svDir            string        = "/wow/_retail_/WTF/Account/2DP3/SavedVariables/"
	POLLING_INTERVAL time.Duration = time.Millisecond * 100
	RETRY_INTERVAL   time.Duration = time.Millisecond * 500
	RETRY_MAX        int           = 3
)

var (
	exportPrefix = []byte(`["export"] = "`)
	exportSuffix = []byte(`",`)
)

type SavedVariables struct {
	Current *AddonData
	Data    chan *AddonData
	File    string
	watcher *watcher.Watcher
}

func NewSV(name string) *SavedVariables {
	return &SavedVariables{
		File:    name,
		Current: &AddonData{},
		Data:    make(chan *AddonData, 1),
		watcher: watcher.New(),
	}
}

func (sv *SavedVariables) path() string {
	return path.Join(svDir, sv.File)
}

func (sv *SavedVariables) read() ([]byte, error) {
	return os.ReadFile(sv.path())
}

func (sv *SavedVariables) watch() error {
	go func() {
		for {
			select {
			case event := <-sv.watcher.Event:
				sv.handleWatchEvent(event)
			case err := <-sv.watcher.Error:
				log.Println("watcher error:", err)
				if errors.Is(err, watcher.ErrWatchedFileDeleted) {
					go sv.retryWatch()
				}
			case <-sv.watcher.Closed:
				return
			}
		}
	}()

	if err := sv.watcher.Add(sv.path()); err != nil {
		return err
	}

	return sv.watcher.Start(POLLING_INTERVAL)
}

func (sv *SavedVariables) handleWatchEvent(e watcher.Event) {
	log.Println("watch event:", e)

	if err := sv.loadAddonData(); err != nil {
		sv.Current = nil
		log.Println("error handling watch event:", err)
		return
	}

	sv.Data <- sv.Current
}

func (sv *SavedVariables) getContents() ([]byte, error) {
	rawData, err := sv.read()
	if err != nil {
		return nil, fmt.Errorf("error reading file: %w", err)
	}

	b64z := extractExport(rawData)
	zData, err := decode(b64z)
	if err != nil {
		return nil, fmt.Errorf("error decoding data: %w", err)
	}

	data, err := decompress(zData)
	if err != nil {
		return nil, fmt.Errorf("error decompressing data: %w", err)
	}

	return data, nil
}

func (sv *SavedVariables) loadAddonData() error {
	contents, err := sv.getContents()
	if err != nil {
		return err
	}
	if err = json.Unmarshal(contents, sv.Current); err != nil {
		return err
	}
	return nil
}

func (sv *SavedVariables) refresh() {
	sv.watcher.TriggerEvent(watcher.Write, sv.watcher.WatchedFiles()[sv.path()])
}

func (sv *SavedVariables) retryWatch() {
	retries := 0
	for retries < RETRY_MAX {
		select {
		case <-sv.watcher.Closed:
			// abort retries if watcher is closed
			return
		case <-time.After(RETRY_INTERVAL * time.Duration(retries)):
			retries++
			log.Printf("retry %d attempting to watch %s\n", retries, sv.path())

			if fileExist(sv.path()) {
				log.Println(sv.path(), "exists: adding to watcher")
				err := sv.watcher.Add(sv.path())
				if err == nil {
					// added file to watcher, break out of retry
					return
				}
				log.Println("error rewatching file:", err)
			} else {
				log.Println("error rewatching file:", sv.path(), "does not exist")
			}

			if retries < RETRY_MAX {
				log.Printf("error rewatching deleted file: retry again in %s\n", RETRY_INTERVAL*time.Duration(retries))
			}
		}
	}
	log.Printf("error rewatching file: exhausted %d retries\n", RETRY_MAX)
}

func extractExport(data []byte) []byte {
	start := bytes.Index(data, exportPrefix)
	if start == -1 {
		return nil
	}

	end := bytes.Index(data[start:], exportSuffix)
	if end == -1 {
		return nil
	}
	return data[start+len(exportPrefix) : start+end]
}

func decode(b64 []byte) ([]byte, error) {
	out := make([]byte, base64.StdEncoding.DecodedLen(len(b64)))
	if _, err := base64.StdEncoding.Decode(out, b64); err != nil {
		return nil, err
	}
	return out, nil
}

func decompress(data []byte) ([]byte, error) {
	reader, err := zlib.NewReader(bytes.NewReader(data))
	defer reader.Close()
	if err != nil {
		return nil, err
	}

	result, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func fileExist(file string) bool {
	_, err := os.Stat(file)
	if err != nil {
		if errors.Is(err, os.ErrExist) {
			return true
		}
		if errors.Is(err, os.ErrNotExist) {
			return false
		}
		return false
	}
	return true
}
