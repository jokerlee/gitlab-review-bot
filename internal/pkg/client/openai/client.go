package openai

import (
	"context"
	"github.com/sashabaranov/go-openai"
	"go.uber.org/ratelimit"
	"net/http"
	"net/url"
)

type Client struct {
	ctx    context.Context
	openai *openai.Client
	rl     ratelimit.Limiter
	closer chan struct{}
}

func (c *Client) Close() {
	c.closer <- struct{}{}
}

func New(rootCtx context.Context, openaiToken, openaiProxyUrl string) (*Client, error) {
	config := openai.DefaultConfig(openaiToken)
	if len(openaiProxyUrl) != 0 {
		proxyUrl, _ := url.Parse(openaiProxyUrl)
		config.HTTPClient = &http.Client{
			Transport: &http.Transport{
				Proxy: http.ProxyURL(proxyUrl),
			},
		}
	}
	c := &Client{
		ctx:    rootCtx,
		openai: openai.NewClientWithConfig(config),
		rl:     ratelimit.New(1),
		closer: make(chan struct{}),
	}

	return c, nil
}
