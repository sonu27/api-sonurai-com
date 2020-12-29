package server

import (
	"bytes"
	"crypto/md5"
	"fmt"
	"net/http"
	"time"
)

func WrapResponseWriter(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ww := NewWrapperResponseWriter(w)
		next.ServeHTTP(ww, r)
		_, _ = ww.Flush(r.Header.Get("If-None-Match"))
	})
}

type wrapperResponseWriter struct {
	http.ResponseWriter
	buf        *bytes.Buffer
	statusCode int
}

func NewWrapperResponseWriter(w http.ResponseWriter) *wrapperResponseWriter {
	return &wrapperResponseWriter{w, new(bytes.Buffer), http.StatusOK}
}

func (w *wrapperResponseWriter) Write(b []byte) (int, error) {
	return w.buf.Write(b)
}

func (w *wrapperResponseWriter) Flush(ifNoneMatch string) (int64, error) {
	if 200 <= w.statusCode && w.statusCode < 300 {
		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
		w.Header().Set("Cache-Control", fmt.Sprintf("max-age=%d", secondsExpiresIn()))
	}
	etag := fmt.Sprintf("\"%x\"", md5.Sum(w.buf.Bytes()))
	w.Header().Set("ETag", etag)
	if ifNoneMatch == etag {
		w.WriteHeader(304)
		w.buf.Reset()
	}

	return w.buf.WriteTo(w.ResponseWriter)
}

func (w *wrapperResponseWriter) WriteHeader(code int) {
	w.statusCode = code
	w.ResponseWriter.WriteHeader(code)
}

func secondsExpiresIn() int {
	now := time.Now()
	expireTime := time.Date(now.Year(), now.Month(), now.Day(), 8, 5, 0, 0, time.UTC)
	secsInDay := 86400

	var secondsExpiresIn int
	if now.Before(expireTime) {
		diff := expireTime.Sub(now)
		secondsExpiresIn = int(diff.Seconds())
	} else {
		diff := now.Sub(expireTime)
		secondsExpiresIn = secsInDay - int(diff.Seconds())
	}

	return secondsExpiresIn
}
