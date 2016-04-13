package http

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"net/http"
)

// SendRequest sends http requests to Ops Man
func SendRequest(method string, url string, user string, passwd string, data string) (string, error) {
	// TODO: Don't skip ssl validation
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	req, err := http.NewRequest(method, url, bytes.NewBufferString(data))
	if err != nil {
		return "", err
	}

	if user != "" && passwd != "" {
		req.SetBasicAuth(user, passwd)
	}

	if method == "POST" {
		req.Header.Add("Content-type", "application/json")
	}

	client := http.Client{Transport: tr}
	res, err := client.Do(req)
	if err != nil {
		return "", err
	}

	body, err := ioutil.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		return "", err
	}

	if method == "POST" && res.Status != "200 OK" {
		return "", fmt.Errorf("got " + res.Status + " on call to opsman; expecting 200")
	}

	return string(body), nil
}
