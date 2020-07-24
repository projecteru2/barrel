package utils

import (
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"

	log "github.com/sirupsen/logrus"
)

var (
	bufferSize int
	debug      bool
)

type writerWrapper struct {
	writer    io.Writer
	writeable bool
}

// Initialize .
func Initialize(_bufferSize int, _debug bool) {
	bufferSize = _bufferSize
	debug = _debug
}

// Forward .
func Forward(src *http.Response, dst http.ResponseWriter) error {
	CopyHeader(src, dst)
	return copyBody(src.Body, dst)
}

// Link .
func Link(client io.ReadWriteCloser, server io.ReadWriteCloser) {
	wg := sync.WaitGroup{}
	wg.Add(2)
	go forwardByteStream(client, server, &wg, "from client")
	go forwardByteStream(server, client, &wg, "from server")
	wg.Wait()
}

func forwardByteStream(src io.Reader, dst io.WriteCloser, wg *sync.WaitGroup, label string) {
	defer wg.Done()
	defer dst.Close()

	log.Print("Starting forward byte stream ", label)
	if _, err := io.Copy(dst, src); err != nil {
		if err == io.EOF {
			log.Print("ForwardByteStream end")
			return
		}
		log.Error("ForwardByteStream end with error", err)
	}
	log.Print("End forward byte stream ", label)
}

func copyBody(reader io.ReadCloser, dst http.ResponseWriter) (err error) {
	defer reader.Close()
	if debug {
		log.Info("Starting copy body")
	}
	// _, err = io.Copy(dst, reader) infact we could use io.Copy here
	// but sometimes we need to inspect byte streams so keep readAndWriteToDst for now
	_, err = readAndWriteToDst(reader, dst, true)
	if debug {
		log.Info("Finish copy body")
	}
	return
}

// CopyHeader .
func CopyHeader(src *http.Response, dst http.ResponseWriter) {
	if debug {
		PrintHeaders("ClientResponse", src.Header)
		log.Info("ContentLength = ", src.ContentLength)
		log.Info("TransferEncoding = ", src.TransferEncoding)
	}
	var chunked bool
	header := dst.Header()
	for key, values := range src.Header {
		for _, value := range values {
			lower := strings.ToLower(key)
			if lower != "content-length" && lower != "transfer-encoding" {
				header.Add(key, value)
			}
		}
	}
	for _, value := range src.TransferEncoding {
		if lower := strings.ToLower(strings.Trim(value, " ")); lower == "chunked" {
			chunked = true
			break
		}
	}
	if !chunked && src.ContentLength != -1 && src.Request.Method != http.MethodHead {
		header.Add("Content-Length", strconv.FormatInt(src.ContentLength, 10))
	}
	if len(src.TransferEncoding) > 0 {
		header.Add("Transfer-Encoding", strings.Join(src.TransferEncoding, ","))
	}
	dst.WriteHeader(src.StatusCode)
	log.Infof("copy response Header, status code = %d\n", src.StatusCode)
	// flush header here, otherwise we can't send header out 'cause the copy body may be blocking
	if f, ok := dst.(http.Flusher); ok {
		log.Info("Flush header")
		f.Flush()
	} else {
		log.Warn("Can't flush header, may cause cli block")
	}
}

// PrintHeaders .
func PrintHeaders(label string, header http.Header) {
	var headers []string
	for key, values := range header {
		for _, value := range values {
			headers = append(headers, fmt.Sprintf("%s: %s;", key, value))
		}
	}
	log.Info(label, " ", strings.Join(headers, " "))
}

// ReadAndForward .
func ReadAndForward(src *http.Response, dst http.ResponseWriter) ([]byte, error) {
	CopyHeader(src, dst)
	return readAndWriteToDst(src.Body, dst, false)
}

func readAndWriteToDst(reader io.Reader, dst io.Writer, writeToDstOnly bool) ([]byte, error) {
	var (
		err   error
		data  []byte
		total int
	)
	buffer := make([]byte, bufferSize)
	wrapper := writerWrapper{dst, true}
	for {
		var count int
		if count, err = readToBuffer(reader, buffer, bufferSize); err != nil && err != io.EOF {
			if debug {
				log.Infof("total bytes read %d\n", total)
			}
			return nil, err
		}
		total += count
		readed := buffer[:count]
		if debug {
			log.Infof("read bytes = %d, total = %d\n", count, total)
			log.Infoln(readed)
		}
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

// WriteToServerResponse .
func WriteToServerResponse(response http.ResponseWriter, statusCode int, header http.Header, body []byte) error {
	if debug {
		log.Info("Write ServerResponse, statusCode = ", statusCode)
		PrintHeaders("ServerResponse", header)
	}
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
		log.Info("Flush")
		flusher.Flush()
	} else {
		log.Info("Can't flush")
	}
	return err
}

// when error is encountered, mark the wrapper not to write in the future
func (wrapper *writerWrapper) write(data []byte) {
	if debug {
		log.Infof("bytes written count: %d", len(data))
	}
	if wrapper.writeable {
		if _, err := wrapper.writer.Write(data); err != nil {
			log.Errorln("write to writer error", err)
			wrapper.writeable = false
		}
	}
}

func readToBuffer(reader io.Reader, buffer []byte, size int) (int, error) {
	return reader.Read(buffer)
}
