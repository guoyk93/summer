package summer

import (
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"net/url"
	"strings"
)

func flattenSlice[T any](s []T) any {
	if len(s) == 1 {
		return s[0]
	}
	return s
}

func flattenRequest(req *http.Request) (m map[string]any, err error) {
	m = map[string]any{}

	// header
	for k, vs := range req.Header {
		k = "header_" + strings.ToLower(strings.ReplaceAll(k, "-", "_"))
		m[k] = flattenSlice(vs)
	}

	// query
	for k, vs := range req.URL.Query() {
		v := flattenSlice(vs)
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
			m[k] = flattenSlice(vs)
		}
	default:
		err = errors.New("unsupported request body type")
		return
	}

	return
}
