/*
Package httpmock builds on httptest, providing easier API mocking.

Essentially all httpmock does is implement a similar interface to httptest, but using a Handler that receives the HTTP
method, path, and body rather than a request object. This makes it very easy to use a featureful mock as the handler,
e.g. github.com/stretchr/testify/mock

Examples

	s := httpmock.NewServer(&httpmock.OKHandler{})
	defer s.Close()

	// Make any requests you want to s.URL(), using it as the mock downstream server

This example uses MockHandler, a Handler that is a github.com/stretchr/testify/mock object.

	downstream := &httpmock.MockHandler{}

	// A simple GET that returns some pre-canned content
	downstream.On("Handle", "GET", "/object/12345", mock.Anything).Return(httpmock.Response{
		Body: []byte(`{"status": "ok"}`),
	})

	s := httpmock.NewServer(downstream)
	defer s.Close()

	//
	// Make any requests you want to s.URL(), using it as the mock downstream server
	//

	downstream.AssertExpectations(t)

If instead you wish to match against headers as well, a slightly different httpmock object can be used
(please note the change in function name to be matched against):

    downstream := &httpmock.MockHandlerWithHeaders{}

    // A simple GET that returns some pre-canned content
    downstream.On("HandleWithHeaders", "GET", "/object/12345", MatchHeader("MOCK", "this"), mock.Anything).Return(httpmock.Response{
        Body: []byte(`{"status": "ok"}`),
    })

    // ... same as above


Httpmock also provides helpers for checking calls using json objects, like so:

	// This tests a hypothetical "echo" endpoint, which returns the body we pass to it.
	type Obj struct {
		A string `json:"a"`
		B string `json:"b"`
	}

	o := &Obj{A: "ay", B: "bee"}

	// JSONMatcher ensures that this mock is triggered only when the HTTP body, when deserialized, matches the given
	// object.
	downstream.On("Handle", "POST", "/echo", httpmock.JSONMatcher(o)).Return(httpmock.Response{
		Body: httpmock.ToJSON(o),
	})

*/
package httpmock

import (
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
)

// Handler is the interface used by httpmock instead of http.Handler so that it can be mocked very easily.
type Handler interface {
	Handle(method, path string, body []byte) Response
}

// HandlerWithHeaders is the interface used by httpmock instead of http.Handler so that it can be mocked very easily,
// it additionally allows matching on headers.
type HandlerWithHeaders interface {
	Handler
	HandleWithHeaders(method, path string, headers http.Header, body []byte) Response
}

// Response holds the response a handler wants to return to the client.
type Response struct {
	// The HTTP status code to write (default: 200)
	Status int
	// Headers to add to the response
	Header http.Header
	// The response body to write (default: no body)
	Body []byte
}

// Server listens for requests and interprets them into calls to your Handler.
type Server struct {
	httpServer *httptest.Server
}

// NewServer constructs a new server and starts it (compare to httptest.NewServer). It needs to be Closed()ed.
// If you pass a handler that conforms to the HandlerWithHeaders interface, when requests are received, the
// HandleWithHeaders method will be called rather than Handle.
func NewServer(handler Handler) *Server {
	s := NewUnstartedServer(handler)
	s.Start()
	return s
}

// NewUnstartedServer constructs a new server but doesn't start it (compare to httptest.NewUnstartedServer).
// If you pass a handler that conforms to the HandlerWithHeaders interface, when requests are received, the
// HandleWithHeaders method will be called rather than Handle.
func NewUnstartedServer(handler Handler) *Server {
	converter := &httpToHTTPMockHandler{}
	if hh, ok := handler.(HandlerWithHeaders); ok {
		converter.handlerWithHeaders = hh
	} else {
		converter.handler = handler
	}
	s := &Server{
		httpServer: httptest.NewUnstartedServer(converter),
	}

	return s
}

// Start starts an unstarted server.
func (s *Server) Start() {
	s.httpServer.Start()
}

// Close shuts down a started server.
func (s *Server) Close() {
	s.httpServer.Close()
}

// URL is the URL for the local test server, i.e. the value of httptest.Server.URL
func (s *Server) URL() string {
	return s.httpServer.URL
}

// httpToHTTPMockHandler is a normal http.Handler that converts the request into a httpmock.Handler call and calls the
// httmock handler.
type httpToHTTPMockHandler struct {
	handler            Handler
	handlerWithHeaders HandlerWithHeaders
}

// ServeHTTP makes this implement http.Handler
func (h *httpToHTTPMockHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Printf("Failed to read HTTP body in httpmock: %v", err)
	}
	var resp Response
	if h.handler != nil {
		resp = h.handler.Handle(r.Method, r.URL.RequestURI(), body)
	} else {
		resp = h.handlerWithHeaders.HandleWithHeaders(r.Method, r.URL.RequestURI(), r.Header, body)
	}

	for k, v := range resp.Header {
		for _, val := range v {
			w.Header().Add(k, val)
		}
	}

	status := resp.Status
	if status == 0 {
		status = 200
	}
	w.WriteHeader(status)
	_, err = w.Write(resp.Body)
	if err != nil {
		log.Printf("Failed to write response in httpmock: %v", err)
	}
}
