package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/oschwald/maxminddb-golang"
)

func update() error {
	const root = "/data"
	stage, err := os.MkdirTemp(root, "release-")
	if err != nil {
		return err
	}
	published := false
	defer func() {
		if !published {
			os.RemoveAll(stage)
		}
	}()
	client := &http.Client{Timeout: 5 * time.Minute}
	for _, db := range []struct{ kind, name string }{{"ASN", "asn"}, {"City", "city"}, {"Country", "country"}} {
		url := "https://github.com/P3TERX/GeoLite.mmdb/raw/download/GeoLite2-" + db.kind + ".mmdb"
		response, err := client.Get(url)
		if err != nil {
			return err
		}
		if response.StatusCode != http.StatusOK {
			response.Body.Close()
			return fmt.Errorf("download %s: %s", db.kind, response.Status)
		}
		path := filepath.Join(stage, db.name+".mmdb")
		file, err := os.Create(path)
		if err != nil {
			response.Body.Close()
			return err
		}
		_, copyErr := io.Copy(file, io.LimitReader(response.Body, 256<<20))
		response.Body.Close()
		closeErr := file.Close()
		if copyErr != nil {
			return copyErr
		}
		if closeErr != nil {
			return closeErr
		}
		reader, err := maxminddb.Open(path)
		if err != nil {
			return fmt.Errorf("validate %s: %w", db.kind, err)
		}
		metadata := reader.Metadata
		verifyErr := reader.Verify()
		reader.Close()
		if verifyErr != nil {
			return fmt.Errorf("validate %s: %w", db.kind, verifyErr)
		}
		if metadata.DatabaseType != "GeoLite2-"+db.kind || metadata.BuildEpoch == 0 {
			return fmt.Errorf("unexpected metadata for %s: %+v", db.kind, metadata)
		}
		log.Printf("Validated %s: built %s", metadata.DatabaseType, time.Unix(int64(metadata.BuildEpoch), 0).UTC().Format(time.RFC3339))
	}
	// Publish the complete set together; running readers retain the old files until restart.
	link := filepath.Join(stage, "next")
	if err := os.Symlink(filepath.Base(stage), link); err != nil {
		return err
	}
	old, _ := os.Readlink(filepath.Join(root, "current"))
	if err := os.Rename(link, filepath.Join(root, "current")); err != nil {
		return err
	}
	published = true
	// Keep the preceding release for recovery.
	previous, _ := os.Readlink(filepath.Join(root, "previous"))
	if old != "" {
		os.Remove(filepath.Join(root, "previous"))
		if err := os.Symlink(old, filepath.Join(root, "previous")); err != nil {
			return err
		}
	}
	if previous != "" && previous != old && filepath.Base(previous) == previous {
		os.RemoveAll(filepath.Join(root, previous))
	}
	log.Printf("Published %s", filepath.Base(stage))
	return nil
}

func main() {
	if err := update(); err != nil {
		log.Fatal(err)
	}
}
