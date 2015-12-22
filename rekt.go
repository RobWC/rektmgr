// Package rektmgr The http request manager so you don't get rekt
package rektmgr

import (
	"io/ioutil"
	"net"
	"net/http"
	"sync"
	"time"
)

// REKTManager manages http requests with the correct level of parallelism
type REKTManager struct {
	client      *http.Client
	workers     int
	respchan    chan Response
	tokens      chan struct{}
	headers     http.Header
	username    string
	password    string
	resphandler func([]byte, error)
	wg          sync.WaitGroup
	respwg      sync.WaitGroup
}

// NewREKTManager create a new REKTManager
func NewREKTManager(workers int) *REKTManager {
	rm := &REKTManager{
		workers: workers,
		client: &http.Client{
			Jar:     nil,
			Timeout: time.Second * 5,
			Transport: &http.Transport{
				Proxy: http.ProxyFromEnvironment,
				Dial: (&net.Dialer{
					Timeout:   30 * time.Second,
					KeepAlive: 30 * time.Second,
				}).Dial,
				TLSHandshakeTimeout: 10 * time.Second,
			}},
		tokens:   make(chan struct{}, workers),
		respchan: make(chan Response),
	}

	rm.respwg.Add(1)
	go func() {
		defer rm.respwg.Done()
		for {
			resp, ok := <-rm.respchan

			if !ok {
				return
			}

			if rm.resphandler != nil {
				go func() {
					rm.resphandler(resp.resp, resp.err)
					rm.respwg.Done()
				}()
			}
		}
	}()

	return rm
}

// Close close down all existing workers
func (rm *REKTManager) Close() {
	rm.wg.Wait()
	close(rm.respchan)
	rm.respwg.Wait()
}

// worker worker to handle work process
func (rm *REKTManager) worker(req http.Request) Response {
	resp, err := rm.client.Do(&req)
	if err != nil {
		return Response{make([]byte, 0), err}
	}
	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return Response{make([]byte, 0), err}
	}

	return Response{data, err}
}

// SetHeader set a header to be sent with all requests
func (rm *REKTManager) SetHeader(key, value string) {
	rm.headers.Set(key, value)
}

// SetBasicAuth setup the basic auth for each request
func (rm *REKTManager) SetBasicAuth(username, password string) {
	rm.username = username
	rm.password = password
}

// SetRespHandler set the response handler for dealing with responses
func (rm *REKTManager) SetRespHandler(rh func([]byte, error)) {
	rm.resphandler = rh
}

// Do execute a request in parallel
func (rm *REKTManager) Do(req http.Request) {

	rm.wg.Add(1)
	go func(req http.Request) {
		defer rm.wg.Done()
		if rm.headers != nil {
			for k, v := range rm.headers {
				req.Header[k] = v
			}
		}

		if (rm.username != "") || (rm.password != "") {
			req.SetBasicAuth(rm.username, rm.password)
		}

		rm.tokens <- struct{}{}
		rm.respwg.Add(1)
		rm.respchan <- rm.worker(req)
		<-rm.tokens
	}(req)
}
