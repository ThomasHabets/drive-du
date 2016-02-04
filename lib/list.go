package lib

/*
 * This file contains library functions for recursively listing a folder on Google Drive.
 */

import (
	"log"
	"math/rand"
	"sync"
	"time"

	drive "github.com/google/google-api-go-client/drive/v2"
)

const (
	DriveFolder = "application/vnd.google-apps.folder"
	backoffBase = 500 * time.Millisecond
	backoff     = 1.5
	maxBackoff  = 2 * time.Minute
)

var (
	Verbose = false
)

func ListRecursive(d *drive.Service, workers int, ch chan<- *File, id string) {
	defer close(ch)
	work := newWork()
	work.add(func() {
		Find(d, work, ch, id, "", nil)
	})
	for i := 0; i < workers; i++ {
		go func() {
			for work.get() {
			}
		}()
	}
	work.wait()
}

func getFile(d *drive.Service, id string) (*drive.File, error) {
	backoff := backoffBase
	for {
		st := time.Now()
		f, err := d.Files.Get(id).Do()
		if err == nil {
			if Verbose {
				log.Printf("Files.Get: %v", time.Since(st))
			}
			return f, nil
		}
		log.Printf("Failed Files.Get: %v\n", err)
		time.Sleep(time.Duration((1.0 + rand.Float64()/2.0) * float64(backoff)))
		backoff *= backoff
		if backoff > maxBackoff {
			backoff = maxBackoff
		}
	}
}

func listDir(d *drive.Service, id, pageToken string) (*drive.ChildList, error) {
	backoff := backoffBase
	for {
		st := time.Now()
		l, err := d.Children.List(id).PageToken(pageToken).Do()
		if err == nil {
			if Verbose {
				log.Printf("Children.List: %v", time.Since(st))
			}
			return l, nil
		}
		log.Printf("Failed Children.List(%s, %q): %v\n", id, pageToken, err)
		time.Sleep(time.Duration((1.0 + rand.Float64()/2.0) * float64(backoff)))
		backoff *= backoff
		if backoff > maxBackoff {
			backoff = maxBackoff
		}
	}
}

type work struct {
	mutex sync.Mutex
	cond  sync.Cond
	work  []func()
	out   int
	done  <-chan bool
}

func newWork() *work {
	w := &work{}
	w.cond.L = &w.mutex
	return w
}

func (w *work) add(f func()) {
	w.mutex.Lock()
	defer w.mutex.Unlock()
	w.work = append(w.work, f)
	w.cond.Signal()
}

func (w *work) get() bool {
	w.mutex.Lock()
	defer w.mutex.Unlock()
	for {
		if len(w.work) == 0 && w.out == 0 {
			w.cond.Broadcast()
			return false
		}
		if len(w.work) != 0 {
			var ret func()
			ret, w.work = w.work[len(w.work)-1], w.work[:len(w.work)-1]
			w.out++
			w.mutex.Unlock()
			ret()
			w.mutex.Lock()
			w.out--
			w.cond.Signal()
			return true
		}
		w.cond.Wait()
	}
}

func (w *work) wait() {
	w.mutex.Lock()
	defer w.mutex.Unlock()
	for {
		if len(w.work) == 0 && w.out == 0 {
			return
		}
		w.cond.Wait()
	}
}

func Find(d *drive.Service, wg *work, ch chan<- *File, id, page string, path []string) {
	//log.Printf("Processing folder: %s", id)
	l, err := listDir(d, id, "")
	if err != nil {
		log.Fatal(err)
	}
	if l.NextPageToken != "" {
		wg.add(func() {
			Find(d, wg, ch, id, l.NextPageToken, path)
		})
	}

	for _, child := range l.Items {
		c := child
		wg.add(func() {
			//log.Printf("Looking at item %v", c.Id)
			f, err := getFile(d, c.Id)
			if err != nil {
				log.Fatal(err)
			}
			if f.ExplicitlyTrashed {
				return
			}
			if f.MimeType == DriveFolder {
				wg.add(func() { Find(d, wg, ch, c.Id, "", append(path, f.Title)) })
			} else {
				ch <- &File{
					Path: path,
					File: f,
				}
			}
		})
	}
}

type File struct {
	Path []string
	File *drive.File
}
