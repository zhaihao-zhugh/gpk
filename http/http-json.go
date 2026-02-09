package http

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

func NewClient(timeout int) HTTPRequset {
	return HTTPRequset{
		&http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
			Timeout: time.Duration(timeout) * time.Minute,
		},
	}
}

func NewDefaultClient() HTTPRequset {
	return HTTPRequset{
		&http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
			Timeout: 5 * time.Minute,
		},
	}
}

func Get(url string, params ...map[string]any) (*http.Response, error) {
	c := NewDefaultClient()
	return c.Get(url, params...)
}

func Post(url string, params ...map[string]any) (*http.Response, error) {
	c := NewDefaultClient()
	return c.Post(url, params...)
}

type HTTPRequset struct {
	*http.Client
}

func (c HTTPRequset) Get(url string, params ...map[string]any) (*http.Response, error) {
	r, err := c.newReq(url, http.MethodGet, params...)
	if err != nil {
		return nil, err
	}
	return c.Do(r)
}

func (c HTTPRequset) Post(url string, params ...map[string]any) (*http.Response, error) {
	r, err := c.newReq(url, http.MethodPost, params...)
	if err != nil {
		return nil, err
	}
	return c.Do(r)
}

func (c HTTPRequset) newReq(urlPath, method string, params ...map[string]any) (*http.Request, error) {
	var buf bytes.Buffer

	if len(params) == 2 {
		switch method {
		case http.MethodGet:
			if params[0] != nil {
				urlPath = urlPath + "?"
				for k, v := range params[0] {
					urlPath = urlPath + k + "=" + url.QueryEscape(fmt.Sprintf("%v", v)) + "&"
				}
			}

			if params[1] != nil {
				if err := json.NewEncoder(&buf).Encode(params[1]); err != nil {
					return nil, err
				}
			}

		case http.MethodPost:
			if params[0] != nil {
				if err := json.NewEncoder(&buf).Encode(params[0]); err != nil {
					return nil, err
				}
			}

			if params[1] != nil {
				urlPath = urlPath + "?"
				for k, v := range params[1] {
					urlPath = urlPath + k + "=" + url.QueryEscape(fmt.Sprintf("%v", v)) + "&"
				}
			}

		}
	} else if len(params) == 1 {
		switch method {
		case http.MethodGet:
			if params[0] != nil {
				urlPath = urlPath + "?"
				for k, v := range params[0] {
					urlPath = urlPath + k + "=" + url.QueryEscape(fmt.Sprintf("%v", v)) + "&"
				}
			}
		case http.MethodPost:
			if params[0] != nil {
				if err := json.NewEncoder(&buf).Encode(params[0]); err != nil {
					return nil, err
				}
			}
		}
	}

	r, err := http.NewRequest(method, urlPath, &buf)
	if err != nil {
		return nil, err
	}
	r.Header.Set("Content-Type", "application/json")

	return r, nil
}
