package transport

import (
	"fmt"
	"net/http"
)

// RequestURL retrieves entry path of request
func RequestURL(req *http.Request) string {
	path := req.URL.Path
	query := req.URL.Query().Encode()
	url := path
	if len(query) > 0 {
		url = fmt.Sprintf("%s?%s", path, query)
	}
	return url
}
