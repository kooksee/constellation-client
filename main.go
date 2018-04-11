package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/tv42/httpunix"
	"io/ioutil"
	"net/http"
	"strings"
	"time"
)

func unixTransport(socketPath string) *httpunix.Transport {
	t := &httpunix.Transport{
		DialTimeout:           1 * time.Second,
		RequestTimeout:        5 * time.Second,
		ResponseHeaderTimeout: 5 * time.Second,
	}
	t.RegisterLocation("c", socketPath)
	return t
}

func unixClient(socketPath string) *http.Client {
	return &http.Client{
		Transport: unixTransport(socketPath),
	}
}

func RunNode(socketPath string) error {
	c := unixClient(socketPath)
	res, err := c.Get("http+unix://c/upcheck")
	if err != nil {
		return err
	}
	if res.StatusCode == 200 {
		return nil
	}
	return errors.New("Constellation Node API did not respond to upcheck request")
}

type Client struct {
	httpClient *http.Client
}

func (c *Client) doJson(path string, apiReq interface{}) (*http.Response, error) {
	buf := new(bytes.Buffer)
	err := json.NewEncoder(buf).Encode(apiReq)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest("POST", "http+unix://c/"+path, buf)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := c.httpClient.Do(req)
	if err == nil && res.StatusCode != 200 {
		return nil, fmt.Errorf("Non-200 status code: %+v", res)
	}
	return res, err
}

func (c *Client) SendPayload(pl []byte, b64From string, b64To []string) ([]byte, error) {
	buf := bytes.NewBuffer(pl)
	req, err := http.NewRequest("POST", "http+unix://c/sendraw", buf)
	if err != nil {
		return nil, err
	}
	if b64From != "" {
		req.Header.Set("c11n-from", b64From)
	}
	req.Header.Set("c11n-to", strings.Join(b64To, ","))
	req.Header.Set("Content-Type", "application/octet-stream")
	res, err := c.httpClient.Do(req)
	if err == nil && res.StatusCode != 200 {
		return nil, fmt.Errorf("Non-200 status code: %+v", res)
	}
	defer res.Body.Close()
	return ioutil.ReadAll(base64.NewDecoder(base64.StdEncoding, res.Body))
}

func (c *Client) ReceivePayload(key []byte) ([]byte, error) {
	req, err := http.NewRequest("GET", "http+unix://c/receiveraw", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("c11n-key", base64.StdEncoding.EncodeToString(key))
	res, err := c.httpClient.Do(req)
	if err == nil && res.StatusCode != 200 {
		return nil, fmt.Errorf("Non-200 status code: %+v", res)
	}
	defer res.Body.Close()
	return ioutil.ReadAll(res.Body)
}

func NewClient(socketPath string) (*Client, error) {
	return &Client{
		httpClient: unixClient(socketPath),
	}, nil
}

func main() {

	c1 := "/Users/barry/demo/quorum/cons/c1/c1.ipc"
	c2 := "/Users/barry/demo/quorum/cons/c2/c2.ipc"

	c1Pub := "TFjDjZqsAV9sc2Sf7noQ4swb90MOLNYA1gTKwmTDlRY="
	c2Pub := "fcH42dCGwmTt+b85/7joIiqdzJxKc4QzZk3bAjMt93U="

	if err := RunNode(c1); err != nil {
		panic(err.Error())
	}

	if err := RunNode(c2); err != nil {
		panic(err.Error())
	}

	c1s, _ := NewClient(c1)
	if res1, err := c1s.SendPayload([]byte("hello111"), c1Pub, []string{c2Pub}); err != nil {
		panic(err.Error())
	} else {

		c2s, _ := NewClient(c2)
		if res11, err := c2s.ReceivePayload(res1); err != nil {
			panic(err.Error())
		} else {
			fmt.Println(string(res11))
		}
	}
}
