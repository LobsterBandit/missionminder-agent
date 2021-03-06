package addon

import (
	"bytes"
	"compress/zlib"
	"context"
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
	svDir           string        = "/wow/_retail_/WTF/Account/2DP3/SavedVariables/"
	PollingInterval time.Duration = time.Millisecond * 100
	RetryInterval   time.Duration = time.Millisecond * 500
	MaxRetries      int           = 3
)

var (
	exportPrefix = []byte(`["export"] = "`) //nolint:gochecknoglobals
	exportSuffix = []byte(`",`)             //nolint:gochecknoglobals

	// The saved variables file is empty.
	ErrAddonContentsEmpty = errors.New("error parsing file: contents empty")

	// Error decoding 0 bytes.
	ErrDecodeData = errors.New("error decoding data: nil or empty byte array")

	// Error decompressing data.
	ErrDecompressData = errors.New("error decompressing data")

	// Export string not found in saved variables file.
	ErrExportMatchNotFound = errors.New("export match not found")
)

type SavedVariables struct {
	Current *Data
	Data    chan *Data
	File    string
	watcher *watcher.Watcher
}

func NewSV(name string) *SavedVariables {
	return &SavedVariables{
		File:    name,
		Current: &Data{Characters: make(map[string]*Character)},
		Data:    make(chan *Data, 1),
		watcher: watcher.New(),
	}
}

func (sv *SavedVariables) path() string {
	return path.Join(svDir, sv.File)
}

func (sv *SavedVariables) read() ([]byte, error) {
	return os.ReadFile(sv.path()) //nolint:wrapcheck
}

func (sv *SavedVariables) Watch(ctx context.Context) error {
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
			case <-ctx.Done():
				sv.watcher.Close()

				return
			case <-sv.watcher.Closed:
				return
			}
		}
	}()

	if err := sv.watcher.Add(sv.path()); err != nil {
		return err //nolint:wrapcheck
	}

	return sv.watcher.Start(PollingInterval) //nolint:wrapcheck
}

// func (sv *SavedVariables) Stop() <-chan struct{} {
// 	sv.watcher.Close()

// 	return sv.watcher.Closed
// }

func (sv *SavedVariables) handleWatchEvent(e watcher.Event) {
	log.Println("watch event:", e)

	if err := sv.LoadAddonData(); err != nil {
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

	if len(rawData) == 0 {
		return nil, ErrAddonContentsEmpty
	}

	b64z, err := extractExport(rawData)
	if err != nil {
		return nil, fmt.Errorf("error extracting data: %w", err)
	}

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

func (sv *SavedVariables) LoadAddonData() error {
	contents, err := sv.getContents()
	if err != nil {
		return err
	}

	if err = json.Unmarshal(contents, sv.Current); err != nil {
		return fmt.Errorf("error loading addon data: %w", err)
	}

	return nil
}

func (sv *SavedVariables) TriggerWrite() {
	sv.watcher.TriggerEvent(watcher.Write, sv.watcher.WatchedFiles()[sv.path()])
}

func (sv *SavedVariables) retryWatch() {
	retries := 0
	for retries < MaxRetries {
		select {
		case <-sv.watcher.Closed:
			// abort retries if watcher is closed
			return
		case <-time.After(RetryInterval * time.Duration(retries)):
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

			if retries < MaxRetries {
				log.Printf("error rewatching deleted file: retry again in %s\n", RetryInterval*time.Duration(retries))
			}
		}
	}
	log.Printf("error rewatching file: exhausted %d retries\n", MaxRetries)
}

func extractExport(data []byte) ([]byte, error) {
	start := bytes.Index(data, exportPrefix)
	if start == -1 {
		return nil, ErrExportMatchNotFound
	}

	end := bytes.Index(data[start:], exportSuffix)
	if end == -1 {
		return nil, ErrExportMatchNotFound
	}

	return data[start+len(exportPrefix) : start+end], nil
}

func decode(b64 []byte) ([]byte, error) {
	if len(b64) == 0 {
		return nil, ErrDecodeData
	}

	out := make([]byte, base64.StdEncoding.DecodedLen(len(b64)))
	if _, err := base64.StdEncoding.Decode(out, b64); err != nil {
		return nil, fmt.Errorf("%s: %w", ErrDecodeData, err)
	}

	return out, nil
}

func decompress(data []byte) ([]byte, error) {
	reader, err := zlib.NewReader(bytes.NewReader(data))
	defer reader.Close()

	if err != nil {
		return nil, fmt.Errorf("%s: %w", ErrDecompressData, err)
	}

	result, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", ErrDecompressData, err)
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
