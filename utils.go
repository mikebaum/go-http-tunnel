package h2tun

import (
	"fmt"
	"io"
	"net/http"

	"github.com/koding/logging"
)

func url(host string) string {
	return fmt.Sprint("https://", host)
}

type closeWriter interface {
	CloseWrite() error
}
type closeReader interface {
	CloseRead() error
}

// TransferLog is a dedicated logger for reporting bytes read/written.
var TransferLog = logging.NewLogger("transfer")

func transfer(side string, dst io.Writer, src io.ReadCloser) {
	n, err := io.Copy(dst, src)
	if err != nil {
		TransferLog.Error("%s: copy error: %s", side, err)
	}

	if d, ok := dst.(closeWriter); ok {
		d.CloseWrite()
	}

	if s, ok := src.(closeReader); ok {
		s.CloseRead()
	} else {
		src.Close()
	}

	TransferLog.Debug("Coppied %d bytes from %s", n, side)
}

func copyHeader(dst, src http.Header) {
	for k, v := range src {
		vv := make([]string, len(v))
		copy(vv, v)
		dst[k] = vv
	}
}

type countWriter struct {
	w     io.Writer
	count int64
}

func (cw *countWriter) Write(p []byte) (n int, err error) {
	n, err = cw.w.Write(p)
	cw.count += int64(n)
	return
}

type flushWriter struct {
	w io.Writer
}

func (fw flushWriter) Write(p []byte) (n int, err error) {
	n, err = fw.w.Write(p)
	if f, ok := fw.w.(http.Flusher); ok {
		f.Flush()
	}
	return
}