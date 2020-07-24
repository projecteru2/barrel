package utils

import (
	"encoding/json"
	"net/http"
	"strconv"

	log "github.com/sirupsen/logrus"
)

// HTTPSimpleMessageResponseBody .
type HTTPSimpleMessageResponseBody struct {
	Message string `json:"message"`
}

// WriteBadGateWayResponse .
func WriteBadGateWayResponse(writer http.ResponseWriter, body HTTPSimpleMessageResponseBody) error {
	return WriteHTTPJSONResponse(writer, http.StatusBadGateway, nil, body)
}

// WriteHTTPJSONResponse .
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

// WriteToServerResponse .
func WriteToServerResponse(response http.ResponseWriter, statusCode int, header http.Header, body []byte) error {
	log.Debugf("[WriteToServerResponse] Write ServerResponse, statusCode = %v", statusCode)
	PrintHeaders("ServerResponse", header)
	responseHeader := response.Header()
	for key, values := range header {
		for _, value := range values {
			responseHeader.Add(key, value)
		}
	}
	responseHeader.Add("Content-Length", strconv.FormatInt(int64(len(body)), 10))
	response.WriteHeader(statusCode)
	_, err := response.Write(body)
	if flusher, ok := response.(http.Flusher); ok {
		flusher.Flush()
	} else {
		log.Error("[WriteToServerResponse] Can't make flush to http.flusher")
	}
	return err
}
