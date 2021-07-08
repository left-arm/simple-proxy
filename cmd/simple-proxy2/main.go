package main

import (
	"context"
	"flag"
	"log"
	"net"
	"net/http"

	"github.com/elazarl/goproxy"

	"github.com/left-arm/simple-proxy/internal/proxy"
)

func main() {
	verbose := flag.Bool("v", false, "should every proxy request be logged to stdout")
	targetURL := flag.String("t", "", "target proxy")
	addr := flag.String("addr", ":8080", "proxy listen address")

	flag.Parse()

	dialer, err := proxy.New(*targetURL)
	if err != nil {
		log.Fatalf("bad target URL %v", err)
	}

	p := goproxy.NewProxyHttpServer()
	p.Verbose = *verbose
	p.ConnectDial = func(network, addr string) (net.Conn, error) {
		return dialer.DialContext(context.Background(), network, addr)
	}

	log.Fatal(http.ListenAndServe(*addr, p))
}
