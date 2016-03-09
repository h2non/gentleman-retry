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

const (
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

var (
	// ConstantBackoff provides a built-in retry strategy based on constant back off.
	ConstantBackoff = retry.New(retry.ConstantBackoff(RetryTimes, RetryWait), nil)

	// ExponentialBackoff provides a built-int retry strategy based on exponential back off.
	ExponentialBackoff = retry.New(retry.ExponentialBackoff(RetryTimes, RetryWait), nil)
)

// Retrier defines the required interface implemented by retry strategies.
type Retrier interface {
	Run(func() error) error
}

// New creates a new retry plugin based on the given retry strategy.
func New(retrier Retrier) plugin.Plugin {
	if retrier == nil {
		retrier = ConstantBackoff
	}

	// Create retry new plugin
	plu := plugin.New()

	// Attach the middleware handler for before dial phase
	plu.SetHandler("before dial", func(ctx *context.Context, h context.Handler) {
		err := InterceptTransport(ctx, retrier)
		if err != nil {
			// If using custom transport, fail with an error for now
			h.Error(ctx, ErrTransport)
			return
		}
		h.Next(ctx)
	})

	return plu
}

// InterceptTransport is a middleware function handler that intercepts
// the HTTP transport based on the given HTTP retrier and context.
func InterceptTransport(ctx *context.Context, retrier Retrier) error {
	// Assert http.Transport to work with the instance
	transport, ok := ctx.Client.Transport.(*http.Transport)
	if !ok {
		// If using custom transport, fail with an error for now
		return ErrTransport
	}

	// Creates the retry transport
	newTransport := &Transport{retrier, transport, ctx}
	ctx.Client.Transport = newTransport
	return nil
}

// Transport provides a http.RoundTripper compatible transport who encapsulates
// the original http.Transport and provides transparent retry support.
type Transport struct {
	retrier   Retrier
	transport *http.Transport
	context   *context.Context
}

// RoundTrip implements the required method by http.RoundTripper interface.
// Performs the network transport over the original http.Transport but providing retry support.
func (t *Transport) RoundTrip(req *http.Request) (*http.Response, error) {
	res := t.context.Response

	// Cache all the body buffer
	buf, err := ioutil.ReadAll(req.Body)
	if err != nil {
		return res, err
	}
	req.Body.Close()

	// Transport request via retrier
	t.retrier.Run(func() error {
		// Clone the http.Request for side effects free
		reqCopy := &http.Request{}
		*reqCopy = *req

		// Restore the cached body buffer
		reqCopy.Body = ioutil.NopCloser(bytes.NewBuffer(buf))

		// Proxy to the original tranport round tripper
		res, err = t.transport.RoundTrip(reqCopy)
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
