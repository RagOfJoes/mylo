package transport

import (
	"net/http"
)

// HttpCodeMap is a map of HTTP status codes that correlate to ServiceClientError's summaries
var HttpCodeMap = map[string]int{
	"Recovery_InvalidFlow":              http.StatusNotFound,
	"Recovery_InvalidSubmitPayload":     http.StatusBadRequest,
	"Recovery_InvalidIdentifierPayload": http.StatusBadRequest,
	"Recovery_FailedCreate":             http.StatusInternalServerError,
}
