package transport

import (
	"net/http"
)

// HttpCodeMap is a map of HTTP status codes that correlate to ServiceClientError's summaries
var HttpCodeMap = map[string]int{
	"Verification_InvalidFlow":               http.StatusNotFound,
	"Verification_InvalidContact":            http.StatusForbidden,
	"Verification_InvalidSessionWarnPayload": http.StatusBadRequest,
	"Verification_FailedCreate":              http.StatusInternalServerError,
}
