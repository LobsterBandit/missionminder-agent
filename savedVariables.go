package main

import (
	"bytes"
	"compress/zlib"
	"encoding/base64"
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

func (sv *SavedVariables) getAddonData() (string, error) {
	rawData, err := sv.read()
	if err != nil {
		return "", fmt.Errorf("error reading file: %w", err)
	}

	b64z := extract(rawData)
	zData, err := decode(b64z)
	if err != nil {
		return "", fmt.Errorf("error decoding data: %w", err)
	}

	data, err := decompress(zData)
	if err != nil {
		return "", fmt.Errorf("error decompressing data: %w", err)
	}

	return data, nil
}

func extract(data string) string {
	match := exportRegex.FindStringSubmatch(data)
	if match == nil {
		return ""
	}
	return match[1]
}

func decode(b64 string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(b64)
}

func decompress(data []byte) (string, error) {
	reader, err := zlib.NewReader(bytes.NewReader(data))
	if err != nil {
		return "", err
	}

	result, err := io.ReadAll(reader)
	if err != nil {
		return "", err
	}

	return string(result), nil
}
