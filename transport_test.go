package main

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRetryTransport(t *testing.T) {
	var counter int
	var httpCode = http.StatusInternalServerError

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if counter == 2 {
			w.Write([]byte("Hello"))
			return
		}
		counter++
		http.Error(w, "Error", httpCode)
	}))
	defer ts.Close()

	counter = 0
	httpCli := http.DefaultClient
	res, err := httpCli.Get(ts.URL)
	if err != nil {
		t.Fatal(err)
	}
	body, err := ioutil.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		t.Fatal(err)
	}
	bodyStr := strings.TrimSpace(string(body))
	if bodyStr != "Error" {
		t.Errorf("Unexpected response: %s", bodyStr)
	}

	counter = 0
	httpCli.Transport = RetryTransport()
	res, err = httpCli.Get(ts.URL)
	if err != nil {
		t.Fatal(err)
	}
	body, err = ioutil.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		t.Fatal(err)
	}
	bodyStr = strings.TrimSpace(string(body))
	if bodyStr != "Hello" {
		t.Errorf("Unexpected response: %s", bodyStr)
	}

	counter = 0
	httpCode = http.StatusTooManyRequests
	httpCli.Transport = RetryTransport()
	res, err = httpCli.Get(ts.URL)
	if err != nil {
		t.Fatal(err)
	}
	body, err = ioutil.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		t.Fatal(err)
	}
	bodyStr = strings.TrimSpace(string(body))
	if bodyStr != "Hello" {
		t.Errorf("Unexpected response: %s", bodyStr)
	}
}
