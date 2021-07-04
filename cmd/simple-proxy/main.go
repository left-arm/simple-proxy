package main

import (
	"flag"
	"log"
	"net/http"

	"github.com/left-arm/simple-proxy/internal/proxy"
)

func main() {
	var (
		targetURL string
		listen    string
	)

	flag.StringVar(&targetURL, "t", "", "target proxy")
	flag.StringVar(&listen, "l", ":8080", "listen at host:port")

	flag.Parse()

	p, err := proxy.New(targetURL)
	if err != nil {
		log.Fatalf("can't create proxy object %v", err)
		return
	}

	http.ListenAndServe(listen, p)
}
