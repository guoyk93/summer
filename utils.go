package summer

import (
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

func respondInternal(rw http.ResponseWriter, s string, code int) {
	buf := []byte(s)
	rw.Header().Set("Content-Type", ContentTypeTextPlainUTF8)
	rw.Header().Set("Content-Length", strconv.Itoa(len(buf)))
	rw.Header().Set("X-Content-Type-Options", "nosniff")
	rw.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate")
	rw.WriteHeader(code)
	_, _ = rw.Write(buf)
}

func flattenSingleSlice[T any](s []T) any {
	if len(s) == 1 {
		return s[0]
	}
	return s
}

func extractRequest(m map[string]any, req *http.Request) (err error) {
	// header
	for k, vs := range req.Header {
		k = "header_" + strings.ToLower(strings.ReplaceAll(k, "-", "_"))
		m[k] = flattenSingleSlice(vs)
	}

	// query
	for k, vs := range req.URL.Query() {
		v := flattenSingleSlice(vs)
		m[k] = v
		m["query_"+k] = v
	}

	// body
	var buf []byte
	if buf, err = io.ReadAll(req.Body); err != nil {
		return
	}

	if len(buf) == 0 {
		return
	}

	var contentType string
	if contentType, _, err = mime.ParseMediaType(req.Header.Get("Content-Type")); err != nil {
		return
	}

	switch contentType {
	case ContentTypeTextPlain:
		m["text"] = string(buf)
	case ContentTypeApplicationJSON:
		var j map[string]any
		if err = json.Unmarshal(buf, &j); err != nil {
			return
		}
		for k, v := range j {
			m[k] = v
		}
	case ContentTypeFormURLEncoded:
		var q url.Values
		if q, err = url.ParseQuery(string(buf)); err != nil {
			return
		}
		for k, vs := range q {
			m[k] = flattenSingleSlice(vs)
		}
	default:
		err = errors.New("unsupported request body type")
		return
	}

	return
}
