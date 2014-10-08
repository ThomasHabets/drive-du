// drive-du is a cloud service for listing folder sizes.
//
// To run you need to change the app name in app.yaml and create config.json, formatted something like:
// {
//     "OAuth": {
//         "ClientID": "xxxx",
//         "ClientSecret": "yyyyyy"
//     }
// }
package main

/*
 * This file contains all the app engine code for listing folder sizes.
 */

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"sort"

	"appengine"
	"appengine/urlfetch"

	"code.google.com/p/goauth2/oauth"
	drive "code.google.com/p/google-api-go-client/drive/v2"

	"lib"
)

const (
	scope       = "https://www.googleapis.com/auth/drive.readonly.metadata https://www.googleapis.com/auth/drive.install https://www.googleapis.com/auth/userinfo.profile"
	redirectURL = "https://drive-du.appspot.com/oauth2callback"
)

var (
	tmplDu    = template.Must(template.ParseFiles("templates/du.html"))
	tmplIndex = template.Must(template.ParseFiles("templates/index.html"))

	config *lib.Config
)

type myError struct {
	public, private string
}

func newError(public, private string) *myError {
	return &myError{
		public:  public,
		private: private,
	}
}

func (e *myError) Error() string {
	return e.private
}

func transport(ctx appengine.Context, s, code string) (*oauth.Transport, error) {
	at := &urlfetch.Transport{
		Context: ctx,
	}
	t := &oauth.Transport{
		Config:    oauthConfig(),
		Transport: at,
	}
	_, err := t.Exchange(code)
	return t, err
}

func oauthConfig() *oauth.Config {
	return lib.OAuthConfig(config.OAuth, scope, redirectURL, "online")
}

func rootHandler(w http.ResponseWriter, r *http.Request) error {
	var buf bytes.Buffer
	if err := tmplIndex.Execute(&buf, &struct {
		AuthURL string
	}{
		AuthURL: oauthConfig().AuthCodeURL(""),
	}); err != nil {
		return newError("Internal error: Template render error", fmt.Sprintf("Template execution error: %v", err))
	}
	_, err := w.Write(buf.Bytes())
	return err
}

func duHandler(w http.ResponseWriter, r *http.Request) error {
	r.ParseForm()
	state := r.FormValue("state")
	http.Redirect(w, r, oauthConfig().AuthCodeURL(state), http.StatusFound)
	return nil
}

func oauth2Handler(w http.ResponseWriter, r *http.Request) error {
	r.ParseForm()
	c := appengine.NewContext(r)
	t, err := transport(c, scope, r.FormValue("code"))
	if err != nil {
		return newError("OAuth error", fmt.Sprintf("failed to get transport: %v", err))
	}
	d, err := drive.New(t.Client())
	if err != nil {
		return newError("OAuth drive client error", fmt.Sprintf("drive.New(): %v", err))
	}

	dirs := []string{"root"}
	state := map[string][]string{}
	err = json.Unmarshal([]byte(r.FormValue("state")), &state)
	if s, ok := state["ids"]; ok {
		dirs = s
	}
	ch := make(chan *lib.File)
	go lib.ListRecursive(d, 20, ch, dirs[0])

	var allTotal int64
	seen := make(map[string]bool)
	storageByDir := make(map[string]int64)
	storageByOwner := make(map[string]int64)
	for f := range ch {
		if seen[f.File.Id] {
			continue
		}
		seen[f.File.Id] = true
		storageByOwner[f.File.Owners[0].EmailAddress] += f.File.FileSize
		if len(f.Path) == 0 {
			storageByDir[f.File.Title] += f.File.FileSize
		} else {
			storageByDir[f.Path[0]] += f.File.FileSize
		}
		allTotal += f.File.FileSize
	}

	// StorageByFolder
	var dirSizes []lib.SizeEntry
	for k, v := range storageByDir {
		dirSizes = append(dirSizes, lib.SizeEntry{
			Key:   k,
			Value: lib.Size(v),
		})
	}
	sort.Sort(lib.ByName(dirSizes))
	dirSizes = append(dirSizes, lib.SizeEntry{
		Key:   "--- Total ---",
		Value: lib.Size(allTotal),
	})

	// StorageByOwner
	var userSizes []lib.SizeEntry
	for k, v := range storageByOwner {
		userSizes = append(userSizes, lib.SizeEntry{
			Key:   k,
			Value: lib.Size(v),
		})
	}
	sort.Sort(lib.ByName(userSizes))
	userSizes = append(userSizes, lib.SizeEntry{
		Key:   "--- Total ---",
		Value: lib.Size(allTotal),
	})

	var buf bytes.Buffer
	if err := tmplDu.Execute(&buf, struct {
		StorageByFolder, StorageByOwner []lib.SizeEntry
	}{
		StorageByFolder: dirSizes,
		StorageByOwner:  userSizes,
	}); err != nil {
		return newError("Internal error: Template render error", fmt.Sprintf("Template execution error: %v", err))
	}
	_, err = w.Write(buf.Bytes())
	return err
}

func wrapHandler(w http.ResponseWriter, r *http.Request, h func(http.ResponseWriter, *http.Request) error) {
	c := appengine.NewContext(r)
	if err := h(w, r); err != nil {
		c.Errorf("Handler error: %s", err.Error())
		if e, ok := err.(*myError); ok {
			http.Error(w, e.public, http.StatusInternalServerError)
		} else {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
	}
}

func init() {
	var err error
	config, err = lib.ReadConfig("config.json")
	if err != nil {
		panic(err)
	}
	for _, h := range []struct {
		path string
		h    func(w http.ResponseWriter, r *http.Request) error
	}{
		{"/", rootHandler},
		{"/du", duHandler},
		{"/oauth2callback", oauth2Handler},
	} {
		h2 := h
		http.HandleFunc(h2.path, func(w http.ResponseWriter, r *http.Request) { wrapHandler(w, r, h2.h) })
	}
}
