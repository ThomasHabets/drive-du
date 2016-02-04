// du is meant to be something like the unix binary du. Currently does appox "du -hcs $FOLDER/*".
//
// Configure with:  ./du -config du.json -configure
// Then run with:   ./du -config du.json 0x_XXXXNNNNAAAABBB
// (Google Drive folder ID can be found in the Web UI URL)
package main

import (
	"flag"
	"fmt"
	"log"
	"sort"

	drive "google.golang.org/api/drive/v2"

	"github.com/ThomasHabets/drive-du/lib"
)

var (
	config     = flag.String("config", "", "Config file.")
	configure  = flag.Bool("configure", false, "Configure oauth.")
	workers    = flag.Int("workers", 10, "Number of Google API workers.")
	sortBySize = flag.Bool("s", false, "Sort by size.")
)

const (
	scope      = "https://www.googleapis.com/auth/drive.readonly.metadata"
	accessType = "offline"
)

func main() {
	flag.Parse()
	if *config == "" {
		log.Fatalf("-config required")
	}

	if *configure {
		if err := lib.ConfigureWrite(scope, accessType, *config); err != nil {
			log.Fatal(err)
		}
		return
	}

	conf, err := lib.ReadConfig(*config)
	if err != nil {
		log.Fatal(err)
	}
	t, err := lib.Connect(conf.OAuth, scope, accessType)
	if err != nil {
		log.Fatal(err)
	}
	d, err := drive.New(t)
	if err != nil {
		log.Fatal(err)
	}

	ch := make(chan *lib.File)
	go lib.ListRecursive(d, *workers, ch, flag.Args()[0])
	//log.Println("Running...")
	//log.Println("Streaming results...")
	var size int64
	dirSizes := make(map[string]int64)
	storageByOwner := make(map[string]int64)
	seen := make(map[string]bool)
	for e := range ch {
		if seen[e.File.Id] {
			continue
		}
		seen[e.File.Id] = true
		size += e.File.FileSize
		for _, o := range e.File.Owners {
			storageByOwner[o.EmailAddress] += e.File.FileSize
		}
		if len(e.Path) == 0 {
			dirSizes[e.File.Title] += e.File.FileSize
		} else {
			dirSizes[e.Path[0]+"/"] += e.File.FileSize
		}
	}

	fmt.Printf("Storage by folder\n----------------\n  T   G   M   k   B\n")
	// Output directories.
	var ds []lib.SizeEntry
	for k, v := range dirSizes {
		ds = append(ds, lib.SizeEntry{
			Key:   k,
			Value: lib.Size(v),
		})
	}
	if *sortBySize {
		sort.Sort(lib.BySize(ds))
	} else {
		sort.Sort(lib.ByName(ds))
	}
	for _, d := range ds {
		fmt.Printf("%19s %s\n", d.Value.Pretty(), d.Key)
	}
	fmt.Printf("\n")

	// Output owners.
	fmt.Printf("Storage by owner\n----------------\n  T   G   M   k   B\n")
	ds = nil
	for k, v := range storageByOwner {
		ds = append(ds, lib.SizeEntry{
			Key:   k,
			Value: lib.Size(v),
		})
	}
	sort.Sort(lib.BySize(ds))
	for _, d := range ds {
		fmt.Printf("%19s %s\n", d.Value.Pretty(), d.Key)
	}
	fmt.Printf("\n")

	fmt.Println("Total size: ", lib.Pretty(size))
}
