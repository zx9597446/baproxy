package main

import (
	"crypto/rand"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/kataras/basicauth"
)

var (
	flgAuth, flgRemoteHost, flgListen string
	flgRemotePort                     int
)

func initFlags() {
	flag.StringVar(&flgListen, "l", ":8089", "listen addr")
	flag.StringVar(&flgAuth, "a", "", "auth for user:pass")
	flag.StringVar(&flgRemoteHost, "h", "localhost", "remote host")
	flag.IntVar(&flgRemotePort, "p", 8088, "remote port")
	flag.Parse()
}

func main() {
	initFlags()

	remote := fmt.Sprintf("http://%s:%d", flgRemoteHost, flgRemotePort)

	// initialize a reverse proxy and pass the actual backend server url here
	proxy, err := NewProxy(remote)
	if err != nil {
		panic(err)
	}

	h := ProxyRequestHandler(proxy)
	s := strings.Split(flgAuth, ":")

	if len(s) == 2 {
		auth := basicauth.Default(map[string]string{
			s[0]: s[1],
		})
		h = basicauth.HandlerFunc(auth, h)
	}

	// handle all requests to your server using the proxy
	http.HandleFunc("/", h)
	log.Fatal(http.ListenAndServe(flgListen, nil))
}

// NewProxy takes target host and creates a reverse proxy
func NewProxy(targetHost string) (*httputil.ReverseProxy, error) {
	url, err := url.Parse(targetHost)
	if err != nil {
		return nil, err
	}

	proxy := httputil.NewSingleHostReverseProxy(url)

	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)
		modifyRequest(req)
	}

	// proxy.ModifyResponse = modifyResponse()
	proxy.ErrorHandler = errorHandler()
	return proxy, nil
}

func modifyRequest(req *http.Request) {
	req.Header.Set("X-Proxy", "Simple-Reverse-Proxy")
}

func errorHandler() func(http.ResponseWriter, *http.Request, error) {
	return func(w http.ResponseWriter, req *http.Request, err error) {
		fmt.Printf("Got error while modifying response: %v \n", err)
	}
}

func modifyResponse() func(*http.Response) error {
	return func(resp *http.Response) error {
		return errors.New("response body is invalid")
	}
}

// ProxyRequestHandler handles the http request using proxy
func ProxyRequestHandler(proxy *httputil.ReverseProxy) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		proxy.ServeHTTP(w, r)
	}
}

var chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890"

func shortID(length int) string {
	ll := len(chars)
	b := make([]byte, length)
	rand.Read(b) // generates len(b) random bytes
	for i := 0; i < length; i++ {
		b[i] = chars[int(b[i])%ll]
	}
	return string(b)
}
