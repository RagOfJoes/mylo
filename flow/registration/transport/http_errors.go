package transport

import (
	"net/http"
)

// HttpCodeMap is a map of HTTP status codes that correlate to ServiceClientError's summaries
var HttpCodeMap = map[string]int{
	"Registration_InvalidFlow":    http.StatusNotFound,
	"Registration_InvalidPayload": http.StatusBadRequest,
	"Registration_FailedCreate":   http.StatusInternalServerError,
}
