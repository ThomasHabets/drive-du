package lib

/*
 * This file contains library functions for doing OAuth.
 */

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strings"

	oauth "golang.org/x/oauth2"
	"google.golang.org/api/googleapi/transport"
)

const (
	spaces               = "\n\t\r "
	OAuthRedirectOffline = "urn:ietf:wg:oauth:2.0:oob"
)

type ConfigOAuth struct {
	ClientID, ClientSecret, RefreshToken, AccessToken, ApiKey string
}

type Config struct {
	OAuth ConfigOAuth
}

func OAuthConfig(cfg ConfigOAuth, scope, redir, accessType string) *oauth.Config {
	return &oauth.Config{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		Endpoint: oauth.Endpoint{
			AuthURL:  "https://accounts.google.com/o/oauth2/auth",
			TokenURL: "https://accounts.google.com/o/oauth2/token",
		},
		Scopes:      []string{scope},
		RedirectURL: redir,
	}
}

func ReadConfig(fn string) (*Config, error) {
	f, err := ioutil.ReadFile(fn)
	if err != nil {
		return nil, err
	}
	var config Config
	if err := json.Unmarshal(f, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

type addKey struct {
	key string
}

func (t *addKey) RoundTrip(req *http.Request) (*http.Response, error) {
	req.URL, _ = url.Parse(req.URL.String() + "&key=" + t.key)
	return http.DefaultTransport.RoundTrip(req)
}

type ts struct {
	token oauth.Token
}

func (t *ts) Token() (*oauth.Token, error) {
	return &t.token, nil
}
func Connect(cfg ConfigOAuth, scope, accessType string) (*http.Client, error) {
	// APIKey may or may not be set.
	// Sometimes APIKey is used instead of normal auth.
	ctx := context.WithValue(context.Background(), oauth.HTTPClient, &http.Client{
		Transport: &transport.APIKey{Key: cfg.ApiKey},
	})

	// If using token, set that up.
	token := &oauth.Token{
		AccessToken:  cfg.AccessToken,
		RefreshToken: cfg.RefreshToken,
	}
	return OAuthConfig(cfg, scope, OAuthRedirectOffline, accessType).Client(ctx, token), nil
}

func auth(cfg ConfigOAuth, scope, at string) (string, error) {
	accessType := oauth.AccessTypeOffline
	if at == "online" {
		accessType = oauth.AccessTypeOnline
	}
	ocfg := OAuthConfig(cfg, scope, OAuthRedirectOffline, at)
	fmt.Printf("Cut and paste this URL into your browser:\n  %s\n", ocfg.AuthCodeURL("", accessType))
	fmt.Printf("Returned code: ")
	line, _ := bufio.NewReader(os.Stdin).ReadString('\n')
	token, err := ocfg.Exchange(oauth.NoContext, line)
	if err != nil {
		return "", err
	}
	return token.RefreshToken, nil
}

func ReadLine(s string) (string, error) {
	reader := bufio.NewReader(os.Stdin)
	fmt.Printf(s)
	id, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	id = strings.Trim(id, spaces)
	return id, nil
}

func ConfigureWrite(scope, at, fn string) error {
	conf, err := Configure(scope, at, "", "")
	if err != nil {
		return err
	}
	b, err := json.Marshal(conf)
	if err != nil {
		return err
	}
	if err := ioutil.WriteFile(fn, b, 0600); err != nil {
		return err
	}
	return nil
}

func ConfigureWriteSharedSecrets(scope, at, fn, clientID, clientSecret string) error {
	conf, err := Configure(scope, at, clientID, clientSecret)
	if err != nil {
		return err
	}
	b, err := json.Marshal(conf)
	if err != nil {
		return err
	}
	if err := ioutil.WriteFile(fn, b, 0600); err != nil {
		return err
	}
	return nil
}

func Configure(scope, at, id, secret string) (*Config, error) {
	var err error

	if id == "" {
		id, err = ReadLine("ClientID: ")
		if err != nil {
			return nil, err
		}
	}
	if secret == "" {
		secret, err = ReadLine("ClientSecret: ")
		if err != nil {
			return nil, err
		}
	}

	token, err := auth(ConfigOAuth{
		ClientID:     id,
		ClientSecret: secret,
	}, scope, at)
	if err != nil {
		return nil, err
	}
	conf := &Config{
		OAuth: ConfigOAuth{
			ClientID:     id,
			ClientSecret: secret,
			RefreshToken: token,
		},
	}
	return conf, nil
}
