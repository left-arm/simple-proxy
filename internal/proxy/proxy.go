package proxy

import (
	"bufio"
	"bytes"
	"context"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"strings"
)

type Proxy struct {
	targetURL *url.URL
	tlsDialer tls.Dialer
	dialer    net.Dialer
}

func New(forwardURL string) (*Proxy, error) {
	u, err := url.Parse(forwardURL)
	if err != nil {
		return nil, err
	}

	return &Proxy{
		targetURL: u,
	}, nil
}

func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method == "CONNECT" {
		proxyConn, err := p.connect(ctx, r.URL.Host)
		if err != nil {
			log.Printf("connect error %v", err)
			http.Error(w, "proxy erorr", http.StatusBadGateway)
			return
		}

		hj, ok := w.(http.Hijacker)
		if !ok {
			w.Header().Add("Connection", "close")
			http.Error(w, "webserver doesn't support hijacking", http.StatusInternalServerError)
			return
		}

		clientConn, bufrw, err := hj.Hijack()
		if err != nil {
			w.Header().Add("Connection", "close")
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer clientConn.Close()

		bufrw.Write([]byte("HTTP/1.1 200 OK\r\n\r\n"))
		err = bufrw.Flush()
		if err != nil {
			w.Header().Add("Connection", "close")
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if tcpConn, ok := clientConn.(*net.TCPConn); ok {
			tcpConn.SetKeepAlive(true)
		}

		// this is simplified implementation.

		go io.Copy(clientConn, proxyConn)

		io.Copy(proxyConn, clientConn)

		return
	}

	http.Error(w, "", http.StatusBadRequest)
}

func (p *Proxy) connect(ctx context.Context, address string) (net.Conn, error) {
	var (
		conn net.Conn
		err  error
	)

	proxyAddr := p.targetURL.Host

	if p.targetURL.Scheme == "https" {
		if strings.Index(proxyAddr, ":") == -1 {
			proxyAddr += ":443"
		}
		conn, err = p.tlsDialer.DialContext(ctx, "tcp", proxyAddr)
	} else {
		if strings.Index(proxyAddr, ":") == -1 {
			proxyAddr += ":80"
		}
		conn, err = p.dialer.Dial("tcp", proxyAddr)
	}

	if err != nil {
		return nil, err
	}

	log.Printf("connected to %q", proxyAddr)

	bw := bytes.NewBuffer(nil)

	fmt.Fprintf(bw, "CONNECT %s HTTP/1.0\r\n", address)
	if u := p.targetURL.User; u != nil {
		n := u.Username()
		p, _ := u.Password()
		fmt.Fprintf(bw, "Proxy-Authorization: Basic %s\r\n", basicAuth(n, p))
	}
	io.WriteString(bw, "\r\n")

	if _, err := conn.Write(bw.Bytes()); err != nil {
		conn.Close()
		return nil, err
	}

	br := bufio.NewReader(conn)

	// Require successful HTTP response
	resp, err := http.ReadResponse(br, &http.Request{Method: "CONNECT"})
	if err != nil {
		conn.Close()
		return nil, err
	}

	if resp.StatusCode != 200 {
		conn.Close()
		return nil, fmt.Errorf("proxy error %v", resp.StatusCode)
	}

	return conn, nil
}

func basicAuth(username, password string) string {
	auth := username + ":" + password
	return base64.StdEncoding.EncodeToString([]byte(auth))
}
