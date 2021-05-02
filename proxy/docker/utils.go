package docker

import (
	"context"
	"net/http"

	dockerContainer "github.com/projecteru2/barrel/docker/container"
	barrelHttp "github.com/projecteru2/barrel/http"
	"github.com/projecteru2/barrel/utils"
	"github.com/projecteru2/barrel/utils/log"
	"github.com/projecteru2/barrel/vessel"
)

type containerUtil struct {
	c        *dockerContainer.Container
	client   barrelHttp.Client
	servResp http.ResponseWriter
	servReq  *http.Request
	res      vessel.ResourceManager
}

func (c *containerUtil) ID() string {
	return c.c.ID
}

func (c *containerUtil) operate(
	operation string,
	callback func(int, string),
) (err error) {
	var resp *http.Response
	if resp, err = c.client.Request(c.servReq); err != nil {
		log.WithError(err).WithField("Operation", operation).Error("Request docker failed")
		if err := utils.WriteBadGateWayResponse(
			c.servResp,
			utils.HTTPSimpleMessageResponseBody{
				Message: "send request to docker socket error",
			},
		); err != nil {
			log.WithError(err).WithField("Operation", operation).Error("write response failed")
		}
		return
	}

	defer resp.Body.Close()
	callback(resp.StatusCode, resp.Status)

	if err = utils.Forward(resp, c.servResp); err != nil {
		log.WithError(err).WithField("Operation", operation).Error("forward failed")
	}
	return
}

func (c *containerUtil) Resources(_ context.Context, volumes bool) (res []vessel.Resource) {
	if c.isFixedIPContainer() {
		res = append(res, c.res.ContainerIPResource(c.ID()))
	}
	if !volumes {
		return
	}
	for _, path := range c.parseMounts() {
		if r, ok := c.res.MountResource(path); ok {
			res = append(res, r)
		}
	}
	return
}

func (c *containerUtil) isFixedIPContainer() bool {
	value, ok := c.c.Config.Labels[FixedIPLabel]
	if !ok {
		return false
	}
	return isFixedIPEnableByStringValue(value)
}

func (c *containerUtil) parseMounts() []string {
	var root *utils.Node
	for _, mnt := range c.c.Mounts {
		if mnt.Source != "" {
			if root == nil {
				root, _ = utils.RootNode(mnt.Source)
			} else {
				root.Add(mnt.Source)
			}
		}
	}
	return root.FoldedPaths("")
}
