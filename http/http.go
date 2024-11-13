package http

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"io"
	"net/http"
	"time"
)

type HTTP struct {
	client  *http.Client
	timeout time.Duration
}

func HandleRequest(url, method string, param any) (*http.Response, error) {
	var c http.Client
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	c.Transport = tr
	req, _ := http.NewRequest(method, url, nil)
	req.Header.Set("Content-Type", "application/json")
	if param != nil {
		body, _ := json.Marshal(param)
		req.Body = io.NopCloser(bytes.NewBuffer(body))
	}
	return c.Do(req)
}

func Get(url string, param any) (*http.Response, error) {
	var c http.Client
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	c.Transport = tr
	var buf bytes.Buffer
	if param != nil {
		if err := json.NewEncoder(&buf).Encode(param); err != nil {
			return nil, err
		}
	}
	req, err := http.NewRequest("GET", url, &buf)
	if err != nil {
		return nil, err
	}
	ctx, _ := context.WithTimeout(context.Background(), 2*time.Minute)
	req = req.WithContext(ctx)
	req.Header.Set("Content-Type", "application/json")
	// if param != nil {
	// 	body, _ := json.Marshal(param)
	// 	req.Body = io.NopCloser(bytes.NewBuffer(body))
	// }
	return c.Do(req)
}

func Post(url string, param any) (*http.Response, error) {
	var c http.Client
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	c.Transport = tr
	var buf bytes.Buffer
	if param != nil {
		if err := json.NewEncoder(&buf).Encode(param); err != nil {
			return nil, err
		}
	}
	req, err := http.NewRequest("POST", url, &buf)
	if err != nil {
		return nil, err
	}
	ctx, _ := context.WithTimeout(context.Background(), 2*time.Minute)
	req = req.WithContext(ctx)
	req.Header.Set("Content-Type", "application/json")
	// if param != nil {
	// 	body, _ := json.Marshal(param)
	// 	req.Body = io.NopCloser(bytes.NewBuffer(body))
	// }
	return c.Do(req)
}
