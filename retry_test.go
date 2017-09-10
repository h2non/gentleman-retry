package retry

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/nbio/st"
	"gopkg.in/h2non/gentleman.v2"
	"gopkg.in/h2non/gentleman.v2/plugins/timeout"
)

func TestRetryRequest(t *testing.T) {
	calls := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("foo", r.Header.Get("foo"))
		fmt.Fprintln(w, "Hello, world")
	}))
	defer ts.Close()

	req := gentleman.NewRequest()
	req.SetHeader("foo", "bar")
	req.URL(ts.URL)
	req.Use(New(nil))

	res, err := req.Send()
	st.Expect(t, err, nil)
	st.Expect(t, res.Ok, true)
	st.Expect(t, res.StatusCode, 200)
	st.Expect(t, res.Header.Get("foo"), "bar")
	st.Expect(t, calls, 3)
}

func TestRetryRequestWithPayload(t *testing.T) {
	calls := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		buf, _ := ioutil.ReadAll(r.Body)
		fmt.Fprintln(w, string(buf))
	}))
	defer ts.Close()

	req := gentleman.NewRequest()
	req.URL(ts.URL)
	req.Method("POST")
	req.BodyString("Hello, world")
	req.Use(New(nil))

	res, err := req.Send()
	st.Expect(t, err, nil)
	st.Expect(t, res.Ok, true)
	st.Expect(t, res.RawResponse.ContentLength, int64(13))
	st.Expect(t, res.StatusCode, 200)
	st.Expect(t, res.String(), "Hello, world\n")
	st.Expect(t, calls, 3)
}

func TestRetryServerError(t *testing.T) {
	calls := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer ts.Close()

	req := gentleman.NewRequest()
	req.URL(ts.URL)
	req.Use(New(nil))

	res, err := req.Send()
	st.Expect(t, err, nil)
	st.Expect(t, res.Ok, false)
	st.Expect(t, res.StatusCode, 503)
	st.Expect(t, calls, 4)
}

func TestRetryNetworkError(t *testing.T) {
	req := gentleman.NewRequest()
	req.URL("http://127.0.0.1:9123")
	req.Use(New(nil))

	res, err := req.Send()
	st.Reject(t, err, nil)
	st.Expect(t, strings.Contains(err.Error(), "connection refused"), true)
	st.Expect(t, res.Ok, false)
	st.Expect(t, res.StatusCode, 0)
}

// Timeout retry is not fully supported yet
func testRetryNetworkTimeout(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(1 * time.Second)
		w.WriteHeader(200)
	}))
	defer ts.Close()

	req := gentleman.NewRequest()
	req.URL(ts.URL)
	req.Use(timeout.Request(100 * time.Millisecond))
	req.Use(New(nil))

	res, err := req.Send()
	st.Reject(t, err, nil)
	st.Expect(t, strings.Contains(err.Error(), "request canceled"), true)
	st.Expect(t, res.Ok, false)
	st.Expect(t, res.StatusCode, 0)
}
