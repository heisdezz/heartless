package http

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	nethttp "net/http"
	"net/url"
	"strings"
	"time"

	"heartless/internal/domain/id"
	domainreq "heartless/internal/domain/request"
	domainres "heartless/internal/domain/response"
)

type Client struct {
	base *nethttp.Client
}

// NewClient creates an HTTP transport client with default baseline settings.
func NewClient() *Client {
	return &Client{base: &nethttp.Client{}}
}

// Execute sends an HTTP request using provided settings and returns a captured response.
func (c *Client) Execute(ctx context.Context, req *domainreq.Request, settings domainreq.HTTPSettings) (*domainres.Response, error) {
	if req == nil {
		return nil, errors.New("request is required")
	}
	if c == nil || c.base == nil {
		c = NewClient()
	}
	if settings.Timeout <= 0 {
		settings.Timeout = domainreq.DefaultHTTPSettings().Timeout
	}

	u, err := buildURL(req.URL, req.Params)
	if err != nil {
		return nil, err
	}

	body := strings.NewReader(req.Body)
	hreq, err := nethttp.NewRequestWithContext(ctx, strings.ToUpper(req.Method), u, body)
	if err != nil {
		return nil, err
	}
	for k, v := range req.Headers {
		hreq.Header.Set(k, v)
	}

	client := c.clientFor(settings)
	start := time.Now()
	hres, err := client.Do(hreq)
	dur := time.Since(start)
	if err != nil {
		return nil, err
	}
	defer hres.Body.Close()

	data, err := io.ReadAll(hres.Body)
	if err != nil {
		return nil, err
	}

	res := domainres.NewHTTPResponse(req.ID, hres.StatusCode, hres.Status)
	res.ID = id.New("res")
	res.Duration = dur
	res.Size = int64(len(data))
	res.Body = data
	res.ContentType = hres.Header.Get("Content-Type")
	res.EffectiveURL = hres.Request.URL.String()
	if res.Headers == nil {
		res.Headers = map[string]string{}
	}
	for k, vals := range hres.Header {
		if len(vals) > 0 {
			res.Headers[k] = vals[len(vals)-1]
		}
	}
	return res, nil
}

func (c *Client) clientFor(settings domainreq.HTTPSettings) *nethttp.Client {
	transport := &nethttp.Transport{
		Proxy: nethttp.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   settings.Timeout,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		TLSClientConfig:     &tls.Config{InsecureSkipVerify: !settings.VerifyTLS}, //nolint:gosec
		ForceAttemptHTTP2:   true,
		MaxIdleConns:        100,
		IdleConnTimeout:     90 * time.Second,
		TLSHandshakeTimeout: 10 * time.Second,
	}
	if settings.ProxyURL != "" {
		if pu, err := url.Parse(settings.ProxyURL); err == nil {
			transport.Proxy = nethttp.ProxyURL(pu)
		}
	}

	client := *c.base
	client.Transport = transport
	client.Timeout = settings.Timeout
	if settings.FollowRedirects {
		max := settings.MaxRedirects
		if max <= 0 {
			max = 10
		}
		client.CheckRedirect = func(req *nethttp.Request, via []*nethttp.Request) error {
			if len(via) >= max {
				return fmt.Errorf("stopped after %d redirects", max)
			}
			return nil
		}
	} else {
		client.CheckRedirect = func(req *nethttp.Request, via []*nethttp.Request) error {
			return nethttp.ErrUseLastResponse
		}
	}
	return &client
}

func buildURL(raw string, params map[string]string) (string, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return "", err
	}
	q := u.Query()
	for k, v := range params {
		q.Set(k, v)
	}
	u.RawQuery = q.Encode()
	return u.String(), nil
}
