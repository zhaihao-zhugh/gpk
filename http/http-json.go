package http

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

var c *HTTPJSON

func init() {
	c = &HTTPJSON{
		&http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
			Timeout: 5 * time.Minute,
		},
	}
}

func Get(url string, param any, header http.Header) (*http.Response, error) {
	return c.get(url, param, header)
}

func Post(url string, param any, header http.Header) (*http.Response, error) {
	return c.post(url, param, header)
}

func SetTimeout(t time.Duration) {
	c.Timeout = t
}

type HTTPJSON struct {
	*http.Client
}

func (c *HTTPJSON) get(url string, param any, header http.Header) (*http.Response, error) {
	r, err := c.newReq(url, http.MethodGet, param, header)
	if err != nil {
		return nil, err
	}
	return c.Do(r)
}

func (c *HTTPJSON) post(url string, param any, header http.Header) (*http.Response, error) {
	r, err := c.newReq(url, http.MethodPost, param, header)
	if err != nil {
		return nil, err
	}
	return c.Do(r)
}

func (c *HTTPJSON) newReq(url, method string, param any, header http.Header) (*http.Request, error) {
	var buf bytes.Buffer
	if param != nil {
		if err := json.NewEncoder(&buf).Encode(param); err != nil {
			return nil, err
		}
		fmt.Println("params: ", buf.String())
	}
	r, err := http.NewRequest(method, url, &buf)
	if err != nil {
		return nil, err
	}
	r.Header.Set("Content-Type", "application/json")
	if header != nil {
		for k, v := range header {
			r.Header.Set(k, v[0])
		}
	}
	return r, nil
}
