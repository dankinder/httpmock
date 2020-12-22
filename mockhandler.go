package httpmock

import (
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"

	"github.com/stretchr/testify/mock"
)

// MockHandler is a httpmock.Handler that uses github.com/stretchr/testify/mock.
type MockHandler struct {
	mock.Mock
}

// Handle makes this implement the Handler interface.
func (m *MockHandler) Handle(method, path string, body []byte) Response {
	args := m.Called(method, path, body)
	return args.Get(0).(Response)
}

// MockHandlerWithHeaders is a httpmock.Handler that uses github.com/stretchr/testify/mock.
type MockHandlerWithHeaders struct {
	mock.Mock
}

// Handle makes this implement the Handler interface.
func (m *MockHandlerWithHeaders) Handle(method, path string, body []byte) Response {
	args := m.Called(method, path, body)
	return args.Get(0).(Response)
}

// HandleWithHeaders makes this implement the HandlerWithHeaders interface.
func (m *MockHandlerWithHeaders) HandleWithHeaders(method, path string, headers http.Header, body []byte) Response {
	args := m.Called(method, path, headers, body)
	return args.Get(0).(Response)
}

// JSONMatcher returns a mock.MatchedBy func to check if the argument is the json form of the provided object.
// See the github.com/stretchr/testify/mock documentation and example in httpmock.go.
func JSONMatcher(o1 interface{}) interface{} {
	return mock.MatchedBy(func(arg []byte) bool {
		// Just using reflect.New on the TypeOf(o1) does not work here, since o1 is an interface. We have to grab the
		// underlying type (Indirect) and create a pointer to that type instead. If you do it the former way, the values
		// LOOK equal, but DeepEqual will always return false, since the pointer type is different.
		o2 := reflect.New(reflect.Indirect(reflect.ValueOf(o1)).Type()).Interface()
		err := json.Unmarshal(arg, o2)
		if err != nil {
			// Assume that this call doesn't match us since we couldn't parse the json
			return false
		}
		return reflect.DeepEqual(o1, o2)
	})
}

// ToJSON is a convenience function for converting an object to JSON inline. It panics on failure, so should be used
// only in test code.
func ToJSON(obj interface{}) []byte {
	data, err := json.Marshal(obj)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal object %v: %v", obj, err))
	}
	return data
}

// HeaderMatcher matches the presence of a header named key that has a given value. Other headers
// are allowed to exist and are not checked.
func HeaderMatcher(key, value string) interface{} {
	headers := make(http.Header)
	headers.Set(key, value)
	return MultiHeaderMatcher(headers)
}

// MultiHeaderMatcher matches the presence and content of multiple headers. Other headers besides those
// within desiredHeaders are allowed to exist and are not checked.
func MultiHeaderMatcher(desiredHeaders http.Header) interface{} {
	return mock.MatchedBy(func(headers http.Header) bool {
		for key, val := range desiredHeaders {
			if headers.Get(key) != val[0] {
				return false
			}
		}
		return true
	})
}
