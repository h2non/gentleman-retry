package retry

import (
	"bytes"
	"errors"
	retry "gopkg.in/eapache/go-resiliency.v1/retrier"
	"gopkg.in/h2non/gentleman.v0/context"
	"gopkg.in/h2non/gentleman.v0/plugin"
	"io/ioutil"
	"net/http"
	"time"
)

var (
	// RetryTimes defines the default max amount of times to retry a request.
	RetryTimes = 3

	// RetryWait defines the default amount of time to wait before each retry attempt.
	RetryWait = 100 * time.Millisecond
)

var (
	// ErrServer stores the error when a server error happens.
	ErrServer = errors.New("retry: server error")

	// ErrTransport stores the error for unsupported http transport.
	ErrTransport = errors.New("retry: cannot use plugin with a custom http transport")
)

// Retrier defines the required interface implemented by retry strategies.
type Retrier interface {
	Run(func() error) error
}

// New creates a new retry plugin based on the given retry strategy.
func New(retrier Retrier) plugin.Plugin {
	if retrier == nil {
		retrier = retry.New(retry.ConstantBackoff(RetryTimes, RetryWait), nil)
	}

	// Create retry new plugin
	plu := plugin.New()

	// Attach the middleware handler for before dial phase
	plu.SetHandler("before dial", func(ctx *context.Context, h context.Handler) {
		// Assert http.Transport to work with the instance
		transport, ok := ctx.Client.Transport.(*http.Transport)
		if !ok {
			// If using custom transport, fail with an error for now
			h.Error(ctx, ErrTransport)
			return
		}

		// Creates the retry transport
		newTransport := &Transport{transport, ctx, retrier}
		ctx.Client.Transport = newTransport

		h.Next(ctx)
	})

	return plu
}

// Transport provides a http.RoundTripper compatible transport who encapsulates
// the original http.Transport and provides transparent retry support.
type Transport struct {
	transport *http.Transport
	context   *context.Context
	retrier   Retrier
}

// RoundTrip implements the required method by http.RoundTripper interface.
// Performs the network transport over the original http.Transport but providing retry support.
func (t *Transport) RoundTrip(req *http.Request) (*http.Response, error) {
	var err error
	var res *http.Response

	// Cache all the body buffer
	buf, err := ioutil.ReadAll(req.Body)
	if err != nil {
		return res, err
	}
	req.Body.Close()

	t.retrier.Run(func() error {
		// Restore the cached body buffer
		req.Body = ioutil.NopCloser(bytes.NewBuffer(buf))

		// Proxy to the original tranport round tripper
		res, err = t.transport.RoundTrip(req)
		if err != nil {
			return err
		}
		if res.StatusCode >= 500 {
			return ErrServer
		}

		return nil
	})

	// Restore original http.Transport
	t.context.Client.Transport = t.transport

	return res, err
}
