package lib

/*
 * This file contains library functions for doing OAuth.
 */

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"code.google.com/p/goauth2/oauth"
)

const (
	spaces               = "\n\t\r "
	OAuthRedirectOffline = "urn:ietf:wg:oauth:2.0:oob"
)

type ConfigOAuth struct {
	ClientID, ClientSecret, RefreshToken string
}

type Config struct {
	OAuth ConfigOAuth
}

func OAuthConfig(cfg ConfigOAuth, scope, redir, accessType string) *oauth.Config {
	return &oauth.Config{
		ClientId:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		AuthURL:      "https://accounts.google.com/o/oauth2/auth",
		Scope:        scope,
		TokenURL:     "https://accounts.google.com/o/oauth2/token",
		RedirectURL:  redir,
		AccessType:   accessType,
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

func Connect(cfg ConfigOAuth, scope, accessType string) (*oauth.Transport, error) {
	t := &oauth.Transport{
		Config: OAuthConfig(cfg, scope, OAuthRedirectOffline, accessType),
		Token: &oauth.Token{
			RefreshToken: cfg.RefreshToken,
		},
	}
	return t, t.Refresh()
}

func auth(cfg ConfigOAuth, scope, at string) (string, error) {
	ocfg := OAuthConfig(cfg, scope, OAuthRedirectOffline, at)
	fmt.Printf("Cut and paste this URL into your browser:\n  %s\n", ocfg.AuthCodeURL(""))
	fmt.Printf("Returned code: ")
	line, _ := bufio.NewReader(os.Stdin).ReadString('\n')
	t := oauth.Transport{Config: ocfg}
	token, err := t.Exchange(line)
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
