package transport

import "net/http"

// HttpCodeMap is a map of HTTP status codes that correlate to ServiceClientError's summaries
var HttpCodeMap = map[string]int{
	"Login_InvalidFlow":    http.StatusNotFound,
	"Login_InvalidPayload": http.StatusBadRequest,
}
