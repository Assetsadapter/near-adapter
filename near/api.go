package near

import (
	"fmt"
	"github.com/blocktree/openwallet/log"
	"github.com/imroc/req"
	"github.com/tidwall/gjson"
)

type Client struct {
	BaseURL string
	Debug   bool
}

func (c *Client) Get(path string, queryparams []interface{}) (*gjson.Result, error) {
	authHeader := req.Header{
		"Accept":       "application/json",
		"Content-Type": "application/json",
	}

	if c.Debug {
		log.Debug("Start Request API...")
	}
	requestPath := fmt.Sprintf("%s%s", c.BaseURL, path)
	r, err := req.Get(requestPath, authHeader)

	if c.Debug {
		log.Debug("Request API Completed")
	}

	if c.Debug {
		log.Debugf("%+v\n", r)
	}

	if err != nil {
		return nil, err
	}

	resp := gjson.ParseBytes(r.Bytes())

	return &resp, nil
}

////return txId
//func (c *Client) submitTransaction(txBase64 string) (*gjson.Result, error) {
//
//	if c.Debug {
//		log.Debug("Request API Completed")
//	}
//	requestPath :=  fmt.Sprintf("%s%s",c.BaseURL,transactions)
//
//	r, err := req.requestPath,authHeader)
//
//	if c.Debug {
//		log.Debugf("%+v\n", r)
//	}
//
//	if err != nil {
//		return nil, err
//	}
//
//	resp := gjson.ParseBytes(r.Bytes())
//
//	return &resp, nil
//
//}
