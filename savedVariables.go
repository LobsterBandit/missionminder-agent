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
	File string
	Data chan *AddonData
}

func (sv *SavedVariables) path() string {
	return path.Join(svDir, sv.File)
}

func (sv *SavedVariables) read() ([]byte, error) {
	return os.ReadFile(sv.path())
}

func (sv *SavedVariables) watch() error {
	w := watcher.New()

	go func() {
		for {
			select {
			case event := <-w.Event:
				sv.handleWatchEvent(event)
			case err := <-w.Error:
				log.Println("watcher error:", err)
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

func (sv *SavedVariables) handleWatchEvent(e watcher.Event) {
	log.Println("watch event:", e)

	data, err := sv.getAddonData()
	if err != nil {
		log.Println("error handling watch event:", err)
		return
	}

	sv.Data <- data
}

func (sv *SavedVariables) getContents() ([]byte, error) {
	rawData, err := sv.read()
	if err != nil {
		return nil, fmt.Errorf("error reading file: %w", err)
	}

	b64z := extractExport(string(rawData))
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

func (sv *SavedVariables) getAddonData() (*AddonData, error) {
	contents, err := sv.getContents()
	if err != nil {
		return nil, err
	}
	data := &AddonData{}
	if err = json.Unmarshal(contents, data); err != nil {
		return nil, err
	}
	return data, nil
}

func extractExport(data string) string {
	match := exportRegex.FindStringSubmatch(data)
	if match == nil {
		return ""
	}
	return match[1]
}

func decode(b64 string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(b64)
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
