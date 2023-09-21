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

	"github.com/kataras/basicauth"
)

var (
	flgUser, flgPass, flgRemote, flgListen string
)

func initFlags() {
	flag.StringVar(&flgUser, "u", "", "username")
	flag.StringVar(&flgPass, "p", "", "password")
	flag.StringVar(&flgRemote, "r", "", "remote addr")
	flag.StringVar(&flgListen, "l", ":8089", "listen addr")
	flag.Parse()
}

func main() {
	initFlags()

	if flgRemote == "" {
		log.Fatal("remote addr is required")
	}
	if flgUser == "" {
		flgUser = shortID(6)
	}
	if flgPass == "" {
		flgPass = shortID(12)
	}

	fmt.Printf("listening on %s, username: %s, password: %s, remote: %s\n", flgListen, flgUser, flgPass, flgRemote)

	auth := basicauth.Default(map[string]string{
		flgUser: flgPass,
	})

	// initialize a reverse proxy and pass the actual backend server url here
	proxy, err := NewProxy(flgRemote)
	if err != nil {
		panic(err)
	}

	h := ProxyRequestHandler(proxy)
	newh := basicauth.HandlerFunc(auth, h)

	// handle all requests to your server using the proxy
	http.HandleFunc("/", newh)
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

	proxy.ModifyResponse = modifyResponse()
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
