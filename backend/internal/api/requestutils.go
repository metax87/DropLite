package api

import (
	"encoding/json"
	"net/http"
)

var jsonDecoder = json.NewDecoder

func decodeJSON(r *http.Request, v any) error {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	return dec.Decode(v)
}
