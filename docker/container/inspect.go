package container

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"

	"github.com/sirupsen/logrus"

	barrelHttp "github.com/projecteru2/barrel/http"
	"github.com/projecteru2/barrel/types"
	"github.com/projecteru2/barrel/utils/errors"
	"github.com/projecteru2/barrel/utils/log"
)

type InspectContainer struct {
	Client              barrelHttp.Client
	ContainerIdentifier string
	APIVersion          string
}

// Inspect .
func (i *InspectContainer) Inspect() (*Container, error) {
	if err := i.checkArgs(); err != nil {
		return nil, err
	}

	return i.inspectContainer()
}

func (i *InspectContainer) checkArgs() error {
	if i.ContainerIdentifier == "" {
		return types.ErrNoContainerIdent
	}
	if i.APIVersion == "" {
		return types.ErrWrongAPIVersion
	}
	return nil
}

func (i *InspectContainer) inspectContainer() (c *Container, err error) {
	var resp *http.Response
	if resp, err = i.requestInspect(); err != nil {
		return nil, err
	}
	defer close(resp.Body)

	switch resp.StatusCode {
	case http.StatusNotFound:
		return nil, types.ErrContainerNotExists
	case http.StatusOK:
		return i.parseContainer(resp.Body)
	}
	errorEntry := i.errorEntry().WithField("ResponseStatus", resp.Status)
	var data []byte
	if data, err = ioutil.ReadAll(resp.Body); err == nil {
		errorEntry = errorEntry.WithField("Body", string(data))
	} else {
		i.logEntry().WithField(
			"ResponseStatus", resp.Status,
		).WithError(err).Error("Read body error")
	}
	return nil, errorEntry.Error("Inspect Container Failed")
}

func (i *InspectContainer) requestInspect() (resp *http.Response, err error) {
	var req *http.Request
	if req, err = i.makeInspectRequest(); err != nil {
		return nil, err
	}
	if resp, err = i.Client.Request(req); err != nil {
		i.logEntry().WithError(err).Error("Inspect container error")
		return nil, err
	}
	i.logEntry().Info("Inspect container done")
	return
}

func (i *InspectContainer) parseContainer(body io.Reader) (*Container, error) {
	var (
		data []byte
		err  error
		c    Container
	)
	if data, err = ioutil.ReadAll(body); err != nil {
		i.logEntry().WithError(err).Error("Read container inspect data error")
		return nil, err
	}

	if err = json.Unmarshal(data, &c); err != nil {
		i.logEntry().WithError(err).Error("Unmarshal container inspect data error")
		return nil, err
	}

	return &c, nil
}

func (i *InspectContainer) makeInspectRequest() (req *http.Request, err error) {
	if req, err = http.NewRequest(
		http.MethodGet,
		i.makeInspectURL(),
		nil,
	); err != nil {
		i.logEntry().Error("Create container inspect request error")
		return nil, err
	}
	return req, nil
}

func (i *InspectContainer) makeInspectURL() string {
	return fmt.Sprintf("/%s/containers/%s/json", i.APIVersion, i.ContainerIdentifier)
}

func (i *InspectContainer) logEntry() *logrus.Entry {
	return log.WithField(
		"ContainerIdentifier", i.ContainerIdentifier,
	).WithField(
		"ApiVersion", i.APIVersion,
	)
}

func (i *InspectContainer) errorEntry() errors.ErrorBuilder {
	return errors.WithField(
		"ContainerIdentifier", i.ContainerIdentifier,
	).WithField(
		"ApiVersion", i.APIVersion,
	)
}

func close(body io.Closer) {
	if err := body.Close(); err != nil {
		log.WithError(err).Error("Close body error")
	}
}
