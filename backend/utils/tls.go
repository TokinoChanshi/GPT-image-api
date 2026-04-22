package utils

import (
	"bufio"
	"context"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	utls "github.com/refraction-networking/utls"
	"golang.org/x/net/http2"
)

// forceH1 为 true 时强制使用 HTTP/1.1 绕过 OpenAI 对 Go 原生 HTTP/2 SETTINGS 帧的检测
var forceH1 = true

type utlsRoundTripper struct {
	proxyURL    *url.URL
	dialer      *net.Dialer
	idleTimeout time.Duration

	mu sync.Mutex
	h1 *http.Transport
	h2 *http2.Transport
}

func NewTLSClient(proxyURL string) (*http.Client, error) {
	rt, err := NewUTLSTransport(proxyURL, 30*time.Second)
	if err != nil {
		return nil, err
	}
	return &http.Client{
		Transport: rt,
		Timeout:   60 * time.Second,
	}, nil
}

func NewUTLSTransport(proxyStr string, idleTimeout time.Duration) (http.RoundTripper, error) {
	rt := &utlsRoundTripper{
		dialer:      &net.Dialer{Timeout: 30 * time.Second, KeepAlive: 30 * time.Second},
		idleTimeout: idleTimeout,
	}
	if proxyStr != "" {
		u, err := url.Parse(proxyStr)
		if err == nil {
			rt.proxyURL = u
		}
	}
	rt.h1 = &http.Transport{
		DialTLSContext: rt.dialTLS,
		ForceAttemptHTTP2: false,
	}
	rt.h2 = &http2.Transport{
		DialTLSContext: func(ctx context.Context, network, addr string, cfg *tls.Config) (net.Conn, error) {
			return rt.dialTLS(ctx, network, addr)
		},
	}
	return rt, nil
}

func (rt *utlsRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if forceH1 {
		return rt.h1.RoundTrip(req)
	}
	return rt.h2.RoundTrip(req)
}

func (rt *utlsRoundTripper) dialTLS(ctx context.Context, network, addr string) (net.Conn, error) {
	host, _, _ := net.SplitHostPort(addr)
	raw, err := rt.dialRaw(ctx, addr)
	if err != nil {
		return nil, err
	}

	alpn := []string{"h2", "http/1.1"}
	if forceH1 {
		alpn = []string{"http/1.1"}
	}

	uconn := utls.UClient(raw, &utls.Config{
		ServerName: host,
		NextProtos: alpn,
		InsecureSkipVerify: true,
	}, utls.HelloChrome_131)

	if forceH1 {
		if err := uconn.BuildHandshakeState(); err != nil {
			raw.Close()
			return nil, err
		}
		for _, ext := range uconn.Extensions {
			if alpnExt, ok := ext.(*utls.ALPNExtension); ok {
				alpnExt.AlpnProtocols = []string{"http/1.1"}
			}
		}
	}

	if err := uconn.HandshakeContext(ctx); err != nil {
		raw.Close()
		return nil, err
	}
	return uconn, nil
}

func (rt *utlsRoundTripper) dialRaw(ctx context.Context, addr string) (net.Conn, error) {
	if rt.proxyURL == nil {
		return rt.dialer.DialContext(ctx, "tcp", addr)
	}
	proxyAddr := rt.proxyURL.Host
	if !strings.Contains(proxyAddr, ":") {
		if strings.EqualFold(rt.proxyURL.Scheme, "https") {
			proxyAddr += ":443"
		} else {
			proxyAddr += ":80"
		}
	}
	conn, err := rt.dialer.DialContext(ctx, "tcp", proxyAddr)
	if err != nil {
		return nil, err
	}
	
	if strings.EqualFold(rt.proxyURL.Scheme, "https") {
		tlsConn := tls.Client(conn, &tls.Config{ServerName: rt.proxyURL.Hostname()})
		if err := tlsConn.Handshake(); err != nil {
			conn.Close()
			return nil, err
		}
		conn = tlsConn
	}

	connectReq := &http.Request{
		Method: http.MethodConnect,
		URL:    &url.URL{Opaque: addr},
		Host:   addr,
		Header: make(http.Header),
	}
	connectReq.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36")
	if rt.proxyURL.User != nil {
		pw, _ := rt.proxyURL.User.Password()
		auth := base64.StdEncoding.EncodeToString([]byte(rt.proxyURL.User.Username() + ":" + pw))
		connectReq.Header.Set("Proxy-Authorization", "Basic "+auth)
	}
	
	if err := connectReq.Write(conn); err != nil {
		conn.Close()
		return nil, err
	}

	br := bufio.NewReader(conn)
	resp, err := http.ReadResponse(br, connectReq)
	if err != nil || resp.StatusCode != http.StatusOK {
		conn.Close()
		return nil, fmt.Errorf("proxy connect failed")
	}
	return conn, nil
}
