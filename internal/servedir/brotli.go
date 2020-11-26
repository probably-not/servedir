// Shamelessly Copied and adjusted to brotli from https://github.com/nytimes/gziphandler/blob/master/gzip.go

package servedir

import (
	"bufio"
	"fmt"
	"io"
	"mime"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"github.com/andybalholm/brotli"
)

const (
	vary            = "Vary"
	acceptEncoding  = "Accept-Encoding"
	contentEncoding = "Content-Encoding"
	contentType     = "Content-Type"
	contentLength   = "Content-Length"
)

type codings map[string]float64

const (
	DefaultQValue  = 1.0
	DefaultMinSize = 1400
)

var brotliWriterPools [brotli.BestCompression - brotli.BestSpeed + 2]*sync.Pool

func init() {
	for i := brotli.BestSpeed; i <= brotli.BestCompression; i++ {
		addLevelPool(i)
	}
}

func poolIndex(level int) int {
	return level - brotli.BestSpeed
}

func addLevelPool(level int) {
	brotliWriterPools[poolIndex(level)] = &sync.Pool{
		New: func() interface{} {
			w := brotli.NewWriterLevel(nil, level)
			return w
		},
	}
}

type BrotliResponseWriter struct {
	http.ResponseWriter
	index        int
	bw           *brotli.Writer
	code         int
	minSize      int
	buf          []byte
	ignore       bool
	contentTypes []parsedContentType
}

func (w *BrotliResponseWriter) Write(b []byte) (int, error) {
	if w.bw != nil {
		return w.bw.Write(b)
	}

	if w.ignore {
		return w.ResponseWriter.Write(b)
	}

	w.buf = append(w.buf, b...)

	var (
		cl, _ = strconv.Atoi(w.Header().Get(contentLength))
		ct    = w.Header().Get(contentType)
		ce    = w.Header().Get(contentEncoding)
	)

	if ce == "" && (cl == 0 || cl >= w.minSize) && (ct == "" || handleContentType(w.contentTypes, ct)) {
		if len(w.buf) < w.minSize && cl == 0 {
			return len(b), nil
		}

		if cl >= w.minSize || len(w.buf) >= w.minSize {
			if ct == "" {
				ct = http.DetectContentType(w.buf)
				w.Header().Set(contentType, ct)
			}

			if handleContentType(w.contentTypes, ct) {
				if err := w.startbrotli(); err != nil {
					return 0, err
				}
				return len(b), nil
			}
		}
	}

	if err := w.startPlain(); err != nil {
		return 0, err
	}

	return len(b), nil
}

func (w *BrotliResponseWriter) startbrotli() error {
	w.Header().Set(contentEncoding, "br")
	w.Header().Del(contentLength)

	if w.code != 0 {
		w.ResponseWriter.WriteHeader(w.code)
		w.code = 0
	}

	if len(w.buf) > 0 {
		w.init()
		n, err := w.bw.Write(w.buf)

		if err == nil && n < len(w.buf) {
			err = io.ErrShortWrite
		}

		return err
	}

	return nil
}

func (w *BrotliResponseWriter) startPlain() error {
	if w.code != 0 {
		w.ResponseWriter.WriteHeader(w.code)
		w.code = 0
	}

	w.ignore = true

	if w.buf == nil {
		return nil
	}

	n, err := w.ResponseWriter.Write(w.buf)
	w.buf = nil

	if err == nil && n < len(w.buf) {
		err = io.ErrShortWrite
	}

	return err
}

func (w *BrotliResponseWriter) WriteHeader(code int) {
	if w.code == 0 {
		w.code = code
	}
}

func (w *BrotliResponseWriter) init() {
	bw := brotliWriterPools[w.index].Get().(*brotli.Writer)
	bw.Reset(w.ResponseWriter)
	w.bw = bw
}

func (w *BrotliResponseWriter) Close() error {
	if w.ignore {
		return nil
	}

	if w.bw == nil {
		err := w.startPlain()
		if err != nil {
			err = fmt.Errorf("Brotlihandler: write to regular responseWriter at close gets error: %q", err.Error())
		}

		return err
	}

	err := w.bw.Close()
	brotliWriterPools[w.index].Put(w.bw)
	w.bw = nil
	return err
}

func (w *BrotliResponseWriter) Flush() {
	if w.bw == nil && !w.ignore {
		return
	}

	if w.bw != nil {
		w.bw.Flush()
	}

	if fw, ok := w.ResponseWriter.(http.Flusher); ok {
		fw.Flush()
	}
}

func (w *BrotliResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if hj, ok := w.ResponseWriter.(http.Hijacker); ok {
		return hj.Hijack()
	}

	return nil, nil, fmt.Errorf("http.Hijacker interface is not supported")
}

var _ http.Hijacker = &BrotliResponseWriter{}

func MustNewBrotliLevelHandler(level int) func(http.Handler) http.Handler {
	wrap, err := NewBrotliLevelHandler(level)
	if err != nil {
		panic(err)
	}

	return wrap
}

func NewBrotliLevelHandler(level int) (func(http.Handler) http.Handler, error) {
	return NewBrotliLevelAndMinSize(level, DefaultMinSize)
}

func NewBrotliLevelAndMinSize(level, minSize int) (func(http.Handler) http.Handler, error) {
	return BrotliHandlerWithOpts(CompressionLevel(level), MinSize(minSize))
}

func BrotliHandlerWithOpts(opts ...option) (func(http.Handler) http.Handler, error) {
	c := &config{
		level:   brotli.DefaultCompression,
		minSize: DefaultMinSize,
	}

	for _, o := range opts {
		o(c)
	}

	if err := c.validate(); err != nil {
		return nil, err
	}

	return func(h http.Handler) http.Handler {
		index := poolIndex(c.level)

		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Add(vary, acceptEncoding)
			if acceptsBrotli(r) {
				bw := &BrotliResponseWriter{
					ResponseWriter: w,
					index:          index,
					minSize:        c.minSize,
					contentTypes:   c.contentTypes,
				}
				defer bw.Close()
				h.ServeHTTP(bw, r)
			} else {
				h.ServeHTTP(w, r)
			}
		})
	}, nil
}

type parsedContentType struct {
	mediaType string
	params    map[string]string
}

func (pct parsedContentType) equals(mediaType string, params map[string]string) bool {
	if pct.mediaType != mediaType {
		return false
	}

	if len(pct.params) == 0 {
		return true
	}

	if len(pct.params) != len(params) {
		return false
	}

	for k, v := range pct.params {
		if w, ok := params[k]; !ok || v != w {
			return false
		}
	}

	return true
}

type config struct {
	minSize      int
	level        int
	contentTypes []parsedContentType
}

func (c *config) validate() error {
	if c.level < brotli.BestSpeed || c.level > brotli.BestCompression {
		return fmt.Errorf("invalid compression level requested: %d", c.level)
	}

	if c.minSize < 0 {
		return fmt.Errorf("minimum size must be more than zero")
	}

	return nil
}

type option func(c *config)

func MinSize(size int) option {
	return func(c *config) {
		c.minSize = size
	}
}

func CompressionLevel(level int) option {
	return func(c *config) {
		c.level = level
	}
}

func ContentTypes(types []string) option {
	return func(c *config) {
		c.contentTypes = []parsedContentType{}
		for _, v := range types {
			mediaType, params, err := mime.ParseMediaType(v)
			if err == nil {
				c.contentTypes = append(c.contentTypes, parsedContentType{mediaType, params})
			}
		}
	}
}

func BrotliHandler(h http.Handler) http.Handler {
	wrapper, _ := NewBrotliLevelHandler(brotli.DefaultCompression)
	return wrapper(h)
}

func acceptsBrotli(r *http.Request) bool {
	acceptedEncodings, _ := parseEncodings(r.Header.Get(acceptEncoding))
	return acceptedEncodings["br"] > 0.0
}

func handleContentType(contentTypes []parsedContentType, ct string) bool {
	if len(contentTypes) == 0 {
		return true
	}

	mediaType, params, err := mime.ParseMediaType(ct)
	if err != nil {
		return false
	}

	for _, c := range contentTypes {
		if c.equals(mediaType, params) {
			return true
		}
	}

	return false
}

func parseEncodings(s string) (codings, error) {
	c := make(codings)
	var e []string

	for _, ss := range strings.Split(s, ",") {
		coding, qvalue, err := parseCoding(ss)

		if err != nil {
			e = append(e, err.Error())
		} else {
			c[coding] = qvalue
		}
	}

	if len(e) > 0 {
		return c, fmt.Errorf("errors while parsing encodings: %s", strings.Join(e, ", "))
	}

	return c, nil
}

func parseCoding(s string) (coding string, qvalue float64, err error) {
	for n, part := range strings.Split(s, ";") {
		part = strings.TrimSpace(part)
		qvalue = DefaultQValue

		if n == 0 {
			coding = strings.ToLower(part)
		} else if strings.HasPrefix(part, "q=") {
			qvalue, err = strconv.ParseFloat(strings.TrimPrefix(part, "q="), 64)

			if qvalue < 0.0 {
				qvalue = 0.0
			} else if qvalue > 1.0 {
				qvalue = 1.0
			}
		}
	}

	if coding == "" {
		err = fmt.Errorf("empty content-coding")
	}

	return
}
