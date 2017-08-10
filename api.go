/*
Package unifi provides programmatic access to UniFi hardware.
*/
package unifi

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// API is an interface to a UniFi controller.
type API struct {
	hc         *http.Client
	cookieBase *url.URL

	as   AuthStore
	auth *Auth
}

// Auth holds the authentication information for accessing a UniFi controller.
type Auth struct {
	Username, Password string
	ControllerHost     string
	Cookies            []*http.Cookie
}

// NewAPI constructs a new API.
func NewAPI(as AuthStore) (*API, error) {
	auth, err := as.Load()
	if err != nil {
		return nil, err
	}
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, err
	}
	cookieBase := &url.URL{
		Scheme: "https",
		Host:   auth.ControllerHost,
	}
	jar.SetCookies(cookieBase, auth.Cookies)

	api := &API{
		hc: &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					// TODO: support proper certs
					InsecureSkipVerify: true,
				},
			},
			Jar: jar,
		},
		cookieBase: cookieBase,
		as:         as,
		auth:       auth,
	}
	return api, nil
}

// WriteConfig writes the configuration to the configured AuthStore.
func (api *API) WriteConfig() error {
	api.auth.Cookies = api.hc.Jar.Cookies(api.cookieBase)
	return api.as.Save(api.auth)
}

func (api *API) get(u string, dst interface{}) error {
	u = api.baseURL() + u

	dec := struct {
		Data interface{} `json:"data"`
		Meta struct {
			Code string `json:"rc"`
			Msg  string `json:"msg"`
		} `json:"meta"`
	}{Data: dst}

	triedLogin := false
	for {
		resp, err := api.hc.Get(u)
		if err != nil {
			return err
		}
		body, err := ioutil.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return err
		}

		if err := json.Unmarshal(body, &dec); err != nil {
			return fmt.Errorf("parsing response body: %v", err)
		}

		if resp.StatusCode == 200 {
			if dec.Meta.Code != "ok" {
				return fmt.Errorf("non-ok return code %q (%s)", dec.Meta.Code, dec.Meta.Msg)
			}
			return nil
		}

		if resp.StatusCode == http.StatusUnauthorized && !triedLogin { // 401
			if dec.Meta.Code == "error" && dec.Meta.Msg == "api.err.LoginRequired" {
				if err := api.login(); err != nil {
					return err
				}
				triedLogin = true
				continue
			}
		}

		return fmt.Errorf("HTTP response %s", resp.Status)
	}
}

func (api *API) baseURL() string {
	return "https://" + api.auth.ControllerHost + ":8443"
}

func (api *API) login() error {
	// TODO: proper JSON encoding
	body := fmt.Sprintf(`{'username':'%s', 'password':'%s'}`, api.auth.Username, api.auth.Password)
	req, err := http.NewRequest("POST", api.baseURL()+"/api/login", strings.NewReader(body))
	if err != nil {
		return fmt.Errorf("building login request: %v", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Referer", api.baseURL()+"/login")
	resp, err := api.hc.Do(req)
	if err != nil {
		return fmt.Errorf("POSTing login form: %v", err)
	}
	_, err = ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return fmt.Errorf("reading login form response: %v", err)
	}
	// A successful response sets a cookie in api.hc.
	if resp.StatusCode != 200 {
		return fmt.Errorf("login form response was %s", resp.Status)
	}
	return nil
}

// An AuthStore is an interface for loading and saving authentication information.
// See FileAuthStore for a file-based implementation.
type AuthStore interface {
	Load() (*Auth, error)
	Save(*Auth) error
}

// DefaultAuthFile is a default place to store authentication information.
// Pass this to FileAuthStore if an alternate path isn't required.
var DefaultAuthFile = filepath.Join(os.Getenv("HOME"), ".unifi-auth")

// FileAuthStore returns an AuthStore that stores authentication information in a named file.
func FileAuthStore(filename string) AuthStore {
	return fileAuthStore{filename}
}

type fileAuthStore struct {
	filename string
}

func (f fileAuthStore) Load() (*Auth, error) {
	// Security check.
	fi, err := os.Stat(f.filename)
	if err != nil {
		return nil, err
	}
	if fi.Mode()&0077 != 0 {
		return nil, fmt.Errorf("security check failed on %s: mode is %04o; it should not be accessible by group/other", f.filename, fi.Mode())
	}

	raw, err := ioutil.ReadFile(f.filename)
	if err != nil {
		return nil, err
	}
	auth := new(Auth)
	if err := json.Unmarshal(raw, auth); err != nil {
		return nil, fmt.Errorf("bad auth file %s: %v", f.filename, err)
	}
	return auth, nil
}

func (f fileAuthStore) Save(auth *Auth) error {
	raw, err := json.Marshal(auth)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(f.filename, raw, 0600)
}

type Client struct {
	ID       string `json:"_id"`
	Name     string `json:"name"`
	Hostname string `json:"hostname"`
	Wired    bool   `json:"is_wired"`

	MAC string `json:"mac"`
	IP  string `json:"ip"`

	LastSeen time.Time

	// TODO: other fields
}

func (c *Client) UnmarshalJSON(data []byte) error {
	type Alias Client
	aux := struct {
		*Alias

		LastSeen int64 `json:"last_seen"`
		// TODO: do this for MAC, IP
	}{Alias: (*Alias)(c)}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	c.LastSeen = time.Unix(aux.LastSeen, 0)
	return nil
}

func (api *API) ListClients(site string) ([]Client, error) {
	var resp []Client
	if err := api.get("/api/s/"+site+"/stat/sta", &resp); err != nil {
		return nil, err
	}
	return resp, nil
}

type WirelessNetwork struct {
	ID      string `json:"_id"`
	Name    string `json:"name"`
	Enabled bool   `json:"enabled"`

	Security string `json:"security"`
	WPAMode  string `json:"wpa_mode"`

	Guest bool `json:"is_guest,omitempty"`

	// TODO: other fields
}

func (api *API) ListWirelessNetworks(site string) ([]WirelessNetwork, error) {
	var resp []WirelessNetwork
	err := api.get("/api/s/"+site+"/list/wlanconf", &resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}
