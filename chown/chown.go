package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"strings"

	drive "google.golang.org/api/drive/v2"

	"github.com/ThomasHabets/drive-du/lib"
)

var (
	srcconfig = flag.String("src_config", "", "Config file.")
	dstconfig = flag.String("dst_config", "", "Config file.")
	configure = flag.Bool("configure", false, "Configure oauth.")
	workers   = flag.Int("workers", 10, "Number of Google API workers.")
	folder    = flag.String("folder", "", "Folder.")
)

const (
	scope           = "https://www.googleapis.com/auth/drive"
	accessType      = "offline"
	folderSeparator = "////!!////"
)

var (
	paths = make(map[string]string)
)

func findDir(d *drive.Service, p []string) string {
	if len(p) == 0 {
		return *folder
	}
	id, ok := paths[strings.Join(p, folderSeparator)]
	if ok {
		return id
	}
	up, this := p[:len(p)-1], p[len(p)-1]
	upID := findDir(d, up)
	thisFile, err := d.Files.Insert(&drive.File{
		Title: this,
		Parents: []*drive.ParentReference{
			&drive.ParentReference{Id: upID},
		},
		MimeType: lib.DriveFolder,
	}).Do()
	if err != nil {
		log.Fatal("mkdir(%q): %v", paths, err)
	}
	paths[strings.Join(p, folderSeparator)] = thisFile.Id
	return thisFile.Id
}

func copyFile(d *drive.Service, t *http.Client, p []string, f *drive.File) error {
	if f.DownloadUrl == "" {
		return fmt.Errorf("file is not downloadable")
	}
	req, err := http.NewRequest("GET", f.DownloadUrl, nil)
	if err != nil {
		return err
	}
	resp, err := t.Do(req)
	defer resp.Body.Close()
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("downloading status: want 200, got %d: %v", resp.StatusCode)
	}
	fout := drive.File{
		Title:            f.Title,
		Description:      f.Description,
		MimeType:         f.MimeType,
		CreatedDate:      f.CreatedDate,
		OriginalFilename: f.OriginalFilename,
		Properties:       f.Properties,
		Labels:           f.Labels,
		Parents: []*drive.ParentReference{
			&drive.ParentReference{Id: findDir(d, p)},
		},
	}
	_, err = d.Files.Insert(&fout).Media(resp.Body).Do()
	if err != nil {
		return err
	}
	return nil
}

func main() {
	flag.Parse()
	if *srcconfig == "" || *dstconfig == "" {
		log.Fatalf("-srcconfig and -dstconfig required")
	}

	if *configure {
		fmt.Printf("------------ Source user ---------------\n")
		if err := lib.ConfigureWrite(scope, accessType, *srcconfig); err != nil {
			log.Fatal(err)
		}
		fmt.Printf("------------ Destination user ---------------\n")
		if err := lib.ConfigureWrite(scope, accessType, *dstconfig); err != nil {
			log.Fatal(err)
		}
		return
	}

	srcconf, err := lib.ReadConfig(*srcconfig)
	if err != nil {
		log.Fatal(err)
	}
	dstconf, err := lib.ReadConfig(*dstconfig)
	if err != nil {
		log.Fatal(err)
	}

	srct, err := lib.Connect(srcconf.OAuth, scope, accessType)
	if err != nil {
		log.Fatal(err)
	}
	dstt, err := lib.Connect(dstconf.OAuth, scope, accessType)
	if err != nil {
		log.Fatal(err)
	}
	srcd, err := drive.New(srct)
	if err != nil {
		log.Fatal(err)
	}
	dstd, err := drive.New(dstt)
	if err != nil {
		log.Fatal(err)
	}

	//lib.Verbose = true
	ch := make(chan *lib.File)
	go lib.ListRecursive(dstd, *workers, ch, *folder)

	seen := make(map[string]bool)
	for e := range ch {
		if seen[e.File.Id] {
			continue
		}
		seen[e.File.Id] = true
		log.Printf("%q %q", e.File.Title, e.File.OwnerNames)
		if e.File.OwnerNames[0] != "Insecure User" {
			continue
		}
		log.Printf("Copying %q", e.File.Title)
		if err := copyFile(dstd, dstt, e.Path, e.File); err != nil {
			log.Fatalf("Failed to download %q (%s): %v", e.File.Title, e.File.Id, err)
		}
		log.Printf("Trashing %s in source user", e.File.Id)
		if _, err := srcd.Files.Trash(e.File.Id).Do(); err != nil {
			log.Fatalf("Failed to trash %q (%s): %v", e.File.Title, e.File.Id, err)
		}
	}
}
