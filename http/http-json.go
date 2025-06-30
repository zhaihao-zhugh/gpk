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

func Get(url string, param any) (*http.Response, error) {
	return c.get(url, param)
}

func Post(url string, param any) (*http.Response, error) {
	return c.post(url, param)
}

func SetTimeout(t time.Duration) {
	c.Timeout = t
}

type HTTPJSON struct {
	*http.Client
}

func (c *HTTPJSON) get(url string, param any) (*http.Response, error) {
	r, err := c.newJsonReq(url, http.MethodGet, param)
	if err != nil {
		return nil, err
	}
	// ctx, _ := context.WithTimeout(context.Background(), 2*time.Minute)
	// r = r.WithContext(ctx)
	return c.Do(r)
}

func (c *HTTPJSON) post(url string, param any) (*http.Response, error) {
	r, err := c.newJsonReq(url, http.MethodPost, param)
	if err != nil {
		return nil, err
	}
	// ctx, _ := context.WithTimeout(context.Background(), 2*time.Minute)
	// r = r.WithContext(ctx)
	return c.Do(r)
}

func (c *HTTPJSON) newJsonReq(url, method string, param any) (*http.Request, error) {
	var buf bytes.Buffer
	if param != nil {
		if err := json.NewEncoder(&buf).Encode(param); err != nil {
			return nil, err
		}
	}
	fmt.Println("params: ", buf.String())
	r, err := http.NewRequest(method, url, &buf)
	if err != nil {
		return nil, err
	}
	r.Header.Set("Content-Type", "application/json")
	return r, nil
}
