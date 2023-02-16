package summer

const (
	ContentTypeApplicationJSON = "application/json"
	ContentTypeTextPlain       = "text/plain"
	ContentTypeFormURLEncoded  = "application/x-www-form-urlencoded"

	ContentTypeApplicationJSONUTF8 = "application/json; charset=utf-8"
	ContentTypeTextPlainUTF8       = "text/plain; charset=utf-8"
	ContentTypeFormURLEncodedUTF8  = "application/x-www-form-urlencoded; charset=utf-8"

	DefaultReadinessPath = "/debug/ready"
	DefaultLivenessPath  = "/debug/alive"
	DefaultMetricsPath   = "/debug/metrics"
)
