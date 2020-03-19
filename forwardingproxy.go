package tunnel

import (
	"bufio"
	"encoding/base64"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"strings"
	"time"

	"github.com/mmatczuk/go-http-tunnel/log"
	"github.com/mmatczuk/go-http-tunnel/proto"
)

// ForwardingProxy uses http tunnel.
type ForwardingProxy struct {
	// localAddr specifies default TCP address of the local server.
	localAddr string
	// localAddrMap specifies mapping from ControlMessage.ForwardedHost to
	// local server address, keys may contain host and port, only host or
	// only port. The order of precedence is the following
	// * host and port
	// * port
	// * host
	localAddrMap map[string]string
	// logger is the proxy logger.
	logger              log.Logger
	AuthUser            string
	AuthPass            string
	ForwardingHTTPProxy *httputil.ReverseProxy
	DestDialTimeout     time.Duration
	DestReadTimeout     time.Duration
	DestWriteTimeout    time.Duration
	ClientReadTimeout   time.Duration
	ClientWriteTimeout  time.Duration
}

// NewForwardingProxy creates new direct TCPProxy, everything will be proxied to
// localAddr.
func NewForwardingProxy(localAddr string, logger log.Logger) *ForwardingProxy {
	if logger == nil {
		logger = log.NewNopLogger()
	}

	return &ForwardingProxy{
		localAddr: localAddr,
		logger:    logger,
	}
}

// NewMultiForwardingProxy creates a new dispatching TCPProxy, connections may go to
// different backends based on localAddrMap.
func NewMultiForwardingProxy(localAddrMap map[string]string, logger log.Logger) *ForwardingProxy {
	if logger == nil {
		logger = log.NewNopLogger()
	}

	return &ForwardingProxy{
		localAddrMap:        localAddrMap,
		logger:              logger,
		ForwardingHTTPProxy: NewReverseProxy(),
		AuthUser:            "",
		AuthPass:            "",
		DestDialTimeout:     10 * time.Second,
		DestReadTimeout:     5 * time.Second,
		DestWriteTimeout:    5 * time.Second,
		ClientReadTimeout:   5 * time.Second,
		ClientWriteTimeout:  5 * time.Second,
	}
}

// Proxy is a ProxyFunc.
func (p *ForwardingProxy) Proxy(w io.Writer, r io.ReadCloser, msg *proto.ControlMessage) {
	switch msg.ForwardedProto {
	case proto.HTTPCONNECT:
		// ok
	default:
		p.logger.Log(
			"level", 0,
			"msg", "unsupported protocol",
			"ctrlMsg", msg,
		)
		return
	}

	rw, ok := w.(http.ResponseWriter)
	if !ok {
		p.logger.Log(
			"level", 0,
			"msg", "expected http.ResponseWriter",
			"ctrlMsg", msg,
		)
	}

	req, err := http.ReadRequest(bufio.NewReader(r))
	if err != nil {
		p.logger.Log(
			"level", 0,
			"msg", "failed to read request",
			"ctrlMsg", msg,
			"err", err,
		)
		return
	}
	setXForwardedFor(req.Header, msg.RemoteAddr)
	req.URL.Host = msg.ForwardedHost

	p.ServeHTTP(rw, req, r)

}

func (p *ForwardingProxy) ServeHTTP(w http.ResponseWriter, r *http.Request, rr io.ReadCloser) {
	p.logger.Log("Incoming request host", r.Host)
	if p.AuthUser != "" && p.AuthPass != "" {
		user, pass, ok := parseBasicProxyAuth(r.Header.Get("Proxy-Authorization"))
		if !ok || user != p.AuthUser || pass != p.AuthPass {
			//p.Logger.Warn("Authorization attempt with invalid credentials")
			http.Error(w, http.StatusText(http.StatusProxyAuthRequired), http.StatusProxyAuthRequired)
			return
		}
	}
	if r.URL.Scheme == "http" {
		p.ForwardingHTTPProxy.ServeHTTP(w, r)
	} else {
		p.handleTunneling(w, r, rr)
	}
}

func (p *ForwardingProxy) handleTunneling(w http.ResponseWriter, r *http.Request, rr io.ReadCloser) {
	if r.Method != http.MethodConnect {
		//p.Logger.Info("Method not allowed", zap.String("method", r.Method))
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}

	//p.Logger.Debug("Connecting", zap.String("host", r.Host))

	destConn, err := net.DialTimeout("tcp", r.Host, p.DestDialTimeout)
	if err != nil {
		//p.Logger.Error("Destination dial failed", zap.Error(err))
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}

	now := time.Now()
	destConn.SetReadDeadline(now.Add(p.DestReadTimeout))
	destConn.SetWriteDeadline(now.Add(p.DestWriteTimeout))

	//p.Logger.Debug("Connected", zap.String("host", r.Host))
	fmt.Println("Connected to host", r.Host)

	resp := http.Response{}
	resp.StatusCode = 200
	resp.Proto = "HTTP/1.1"
	resp.ProtoMajor = 1
	resp.ProtoMinor = 1

	resp.Write(w)

	w.(http.Flusher).Flush()

	fmt.Println("Flushed to host", r.Host)

	done := make(chan struct{})
	go func() {
		transfer(flushWriter{w}, destConn, log.NewContext(p.logger).With(
			// "dst", msg.ForwardedHost,
			"src", r.Host,
		))
		close(done)
	}()

	transfer(destConn, rr, log.NewContext(p.logger).With(
		"dst", r.Host,
		// "src", msg.ForwardedHost,
	))

	<-done

	fmt.Println("Done for", r.Host)
	destConn.Close()
	rr.Close()
}

// parseBasicProxyAuth parses an HTTP Basic Authorization string.
// "Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==" returns ("Aladdin", "open sesame", true).
func parseBasicProxyAuth(authz string) (username, password string, ok bool) {
	const prefix = "Basic "
	if !strings.HasPrefix(authz, prefix) {
		return
	}
	c, err := base64.StdEncoding.DecodeString(authz[len(prefix):])
	if err != nil {
		return
	}
	cs := string(c)
	s := strings.IndexByte(cs, ':')
	if s < 0 {
		return
	}
	return cs[:s], cs[s+1:], true
}

// NewReverseProxy retuns a new reverse proxy that takes an incoming
// request and sends it to another server, proxying the response back to the
// client.
//
// See: https://golang.org/pkg/net/http/httputil/#ReverseProxy
func NewReverseProxy() *httputil.ReverseProxy {
	director := func(req *http.Request) {
		if _, ok := req.Header["User-Agent"]; !ok {
			// explicitly disable User-Agent so it's not set to default value
			req.Header.Set("User-Agent", "")
		}
	}
	// TODO:(alesr) Use timeouts specified via flags to customize the default
	// transport used by the reverse proxy.
	return &httputil.ReverseProxy{
		Director: director,
	}
}
