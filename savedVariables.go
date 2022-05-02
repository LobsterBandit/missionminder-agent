package main

import (
	"bytes"
	"compress/zlib"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"regexp"
	"time"

	"github.com/radovskyb/watcher"
)

const (
	svDir         = "/wow/_retail_/WTF/Account/2DP3/SavedVariables/"
	exportPattern = `\[\"export\"\] = \"(.*)\",`
)

var exportRegex = regexp.MustCompile(exportPattern)

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
		Data:    make(chan *AddonData),
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
			case <-sv.watcher.Closed:
				return
			}
		}
	}()

	if err := sv.watcher.Add(sv.path()); err != nil {
		return err
	}

	return sv.watcher.Start(time.Millisecond * 100)
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

func extractExport(data []byte) []byte {
	match := exportRegex.FindSubmatch(data)
	if match == nil {
		return nil
	}
	return match[1]
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
	if err != nil {
		return nil, err
	}

	result, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}

	return result, nil
}
