package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"strings"

	drive "code.google.com/p/google-api-go-client/drive/v2"

	"github.com/ThomasHabets/drive-du/lib"
)

var (
	config    = flag.String("config", "", "Config file.")
	configure = flag.Bool("configure", false, "Configure oauth.")
	workers   = flag.Int("workers", 10, "Number of Google API workers.")
	src       = flag.String("src", "", "Source folder.")
	dst       = flag.String("dst", "", "Destination folder.")
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
		return *dst
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

func copyFile(d *drive.Service, t http.RoundTripper, p []string, f *drive.File) error {
	if f.DownloadUrl == "" {
		return fmt.Errorf("file is not downloadable")
	}
	req, err := http.NewRequest("GET", f.DownloadUrl, nil)
	if err != nil {
		return err
	}
	resp, err := t.RoundTrip(req)
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
	go lib.ListRecursive(d, *workers, ch, *src)

	seen := make(map[string]bool)
	for e := range ch {
		if seen[e.File.Id] {
			continue
		}
		seen[e.File.Id] = true
		log.Printf("Copying %q", e.File.Title)
		if err := copyFile(d, t, e.Path, e.File); err != nil {
			log.Fatalf("Failed to download %q (%s): %v", e.File.Title, e.File.Id, err)
		}
	}
}
