package utils

import (
	"io"
	"net/http"
	"strconv"

	log "github.com/sirupsen/logrus"
)

type NetUtil struct {
	BufferSize int
}

type writerWrapper struct {
	writer    io.Writer
	writeable bool
}

func (netutil NetUtil) Forward(src *http.Response, dst http.ResponseWriter) error {
	CopyHeader(src, dst)
	_, err := netutil.readAndWriteToDst(src.Body, dst, true)
	return err
}

func CopyHeader(src *http.Response, dst http.ResponseWriter) {
	header := dst.Header()
	for key, values := range src.Header {
		for _, value := range values {
			header.Add(key, value)
		}
	}
	dst.WriteHeader(src.StatusCode)
}

func (netutil NetUtil) ReadAndForward(src *http.Response, dst http.ResponseWriter) ([]byte, error) {
	CopyHeader(src, dst)
	return netutil.readAndWriteToDst(src.Body, dst, false)
}

func (netutil NetUtil) readAndWriteToDst(reader io.Reader, dst io.Writer, writeToDstOnly bool) ([]byte, error) {
	var (
		err  error
		data []byte
	)
	buffer := make([]byte, netutil.BufferSize)
	wrapper := writerWrapper{dst, true}
	for {
		var count int
		if count, err = readToBuffer(reader, buffer, netutil.BufferSize); err != nil && err != io.EOF {
			return nil, err
		}
		readed := buffer[:count]
		if !writeToDstOnly {
			data = append(data, readed...)
		}
		wrapper.write(readed)
		if writeToDstOnly && !wrapper.writeable {
			return nil, nil
		}
		if err == io.EOF {
			return data, nil
		}
	}
}

func WriteToServerResponse(response http.ResponseWriter, statusCode int, header http.Header, body []byte) error {
	responseHeader := response.Header()
	for key, values := range header {
		for _, value := range values {
			responseHeader.Add(key, value)
		}
	}
	responseHeader.Add("Content-Length", strconv.FormatInt(int64(len(body)), 10))
	response.WriteHeader(statusCode)
	_, err := response.Write(body)
	return err
}

// when error is encountered, mark the wrapper not to write in the future
func (wrapper *writerWrapper) write(data []byte) {
	if wrapper.writeable {
		if _, err := wrapper.writer.Write(data); err != nil {
			log.Errorln("write to writer error", err)
			wrapper.writeable = false
		}
	}
}

func readToBuffer(reader io.Reader, buffer []byte, size int) (int, error) {
	var err error
	var total int = 0
	for total < size {
		var count int
		if count, err = reader.Read(buffer); err != nil {
			return total + count, err
		}
		buffer = buffer[count:]
		total += count
	}
	return total, nil
}
