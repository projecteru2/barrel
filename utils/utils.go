package utils

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
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

// Initialize .
func Initialize(bufSize int) {
	bufferSize = bufSize
	debug = log.GetLevel() == log.DebugLevel
}

// Forward .
func Forward(src *http.Response, dst http.ResponseWriter) error {
	copyHeader(src, dst)
	return copyBody(src.Body, dst)
}

// ForwardAndRead .
func ForwardAndRead(src *http.Response, dst http.ResponseWriter) (body []byte, err error) {
	copyHeader(src, dst)
	buffer := bytes.NewBuffer(nil)
	reader := io.TeeReader(src.Body, buffer)
	if err = copyBody(ioutil.NopCloser(reader), dst); err != nil {
		return
	}
	body, err = ioutil.ReadAll(buffer)
	return
}

// Link .
func Link(client io.ReadWriteCloser, server io.ReadWriteCloser) {
	wg := sync.WaitGroup{}
	wg.Add(2)
	defer wg.Wait()
	go forwardByteStream(client, server, &wg, "from client")
	go forwardByteStream(server, client, &wg, "from server")
}

// PrintHeaders .
func PrintHeaders(label string, header http.Header) {
	if !debug {
		return
	}
	var headers []string
	for key, values := range header {
		for _, value := range values {
			headers = append(headers, fmt.Sprintf("%s: %s;", key, value))
		}
	}
	log.Debugf("[PrintHeaders] %v %s", label, strings.Join(headers, " "))
}

// ReadAndForward .
func ReadAndForward(src *http.Response, dst http.ResponseWriter) ([]byte, error) {
	copyHeader(src, dst)
	return readAndWriteToDst(src.Body, dst, false)
}

func readToBuffer(reader io.Reader, buffer []byte, _ int) (int, error) {
	return reader.Read(buffer)
}

func forwardByteStream(src io.Reader, dst io.WriteCloser, wg *sync.WaitGroup, label string) {
	defer wg.Done()
	defer dst.Close()

	log.Debugf("[forwardByteStream] Starting forward byte stream %v", label)
	if _, err := io.Copy(dst, src); err != nil {
		if err == io.EOF {
			log.Debug("[forwardByteStream] ForwardByteStream end")
			return
		}
		log.Errorf("[forwardByteStream] ForwardByteStream end with %v", err)
	}
	log.Debugf("[forwardByteStream] End forward byte stream %v", label)
}

func copyBody(reader io.ReadCloser, dst io.Writer) (err error) {
	defer reader.Close()
	log.Debug("[copyBody] Starting copy body")
	// _, err = io.Copy(dst, reader) infact we could use io.Copy here
	// but sometimes we need to inspect byte streams so keep readAndWriteToDst for now
	_, err = readAndWriteToDst(reader, dst, true)
	log.Debug("[copyBody] Finish copy body")
	return
}

func copyHeader(src *http.Response, dst http.ResponseWriter) {
	PrintHeaders("ClientResponse", src.Header)
	log.Debugf("[copyHeader] ContentLength = %v", src.ContentLength)
	log.Debugf("[copyHeader] TransferEncoding = %v", src.TransferEncoding)
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
	log.Infof("[copyHeader] copy response Header, status code = %d", src.StatusCode)
	// flush header here, otherwise we can't send header out 'cause the copy body may be blocking
	if f, ok := dst.(http.Flusher); ok {
		log.Debug("[copyHeader] Flush header")
		f.Flush()
	} else {
		log.Warn("[copyHeader] Can't flush header, may cause cli block")
	}
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
			log.Debugf("[readAndWriteToDst] total bytes read %d", total)
			return nil, err
		}
		total += count
		readed := buffer[:count]
		log.Debugf("[readAndWriteToDst] read bytes = %d, total = %d", count, total)
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

type writerWrapper struct {
	writer    io.Writer
	writeable bool
}

// when error is encountered, mark the wrapper not to write in the future
func (wrapper *writerWrapper) write(data []byte) {
	log.Debugf("[write] bytes written count: %d", len(data))
	if wrapper.writeable {
		if _, err := wrapper.writer.Write(data); err != nil {
			log.Errorf("[write] write to writer failed %v", err)
			wrapper.writeable = false
		}
	}
}
