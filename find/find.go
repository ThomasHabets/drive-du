package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"

	drive "code.google.com/p/google-api-go-client/drive/v2"

	"github.com/ThomasHabets/drive-du/lib"
)

var (
	config    = flag.String("config", "", "Config file.")
	configure = flag.Bool("configure", false, "Configure oauth.")
	workers   = flag.Int("workers", 10, "Number of Google API workers.")
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
	d, err := drive.New(t.Client())
	if err != nil {
		log.Fatal(err)
	}

	ch := make(chan *lib.File)
	go lib.ListRecursive(d, *workers, ch, flag.Args()[0])
	var size int64
	seen := make(map[string]bool)
	for e := range ch {
		if seen[e.File.Id] {
			continue
		}
		seen[e.File.Id] = true
		fmt.Println(e.Path, e.File.Title)
		size += e.File.FileSize
	}
	fmt.Println("Total size: ", size)
}
