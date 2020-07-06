package utils

import (
	"encoding/json"
	"net/http"
)

// HTTPSimpleMessageResponseBody .
type HTTPSimpleMessageResponseBody struct {
	Message string `json:"message"`
}

func WriteHTTPInternalServerErrorResponse(writer http.ResponseWriter, body HTTPSimpleMessageResponseBody) error {
	return WriteHTTPJSONResponse(writer, http.StatusInternalServerError, nil, body)
}

func WriteHTTPJSONResponse(writer http.ResponseWriter, statusCode int, prevHeader http.Header, body interface{}) error {
	var (
		data []byte
		err  error
	)
	header := make(http.Header)
	for key, values := range prevHeader {
		for _, value := range values {
			header.Add(key, value)
		}
	}
	header.Add("Content-Type", "application/json")
	if data, err = json.Marshal(&body); err != nil {
		return err
	}
	return WriteToServerResponse(writer, statusCode, header, data)
}
