package httpmock

import (
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gotest.tools/v3/assert"
)

func TestBasicRequestResponse(t *testing.T) {

	downstream := NewMockHandler(t)

	downstream.On("Handle", "GET", "/object/12345", mock.Anything).Return(Response{
		Body: []byte(`{"status": "ok"}`),
	})

	s := NewServer(downstream)
	defer s.Close()

	req, err := http.NewRequest("GET", fmt.Sprintf("%s/object/12345", s.URL()), nil)
	require.NoError(t, err)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.DeepEqual(t, []byte(`{"status": "ok"}`), body)

	downstream.AssertExpectations(t)
}

func TestBasicRequestResponseWithHeaders(t *testing.T) {
	headerKey := "HTTPMOCK-TEST"
	headerVal := "its here"
	downstream := NewMockHandlerWithHeaders(t)

	downstream.On(
		"HandleWithHeaders",
		"GET",
		"/object/12345",
		HeaderMatcher(headerKey, headerVal),
		mock.Anything,
	).
		Return(Response{
			Body: []byte(`{"status": "ok"}`),
		})

	s := NewServer(downstream)
	defer s.Close()

	req, err := http.NewRequest("GET", fmt.Sprintf("%s/object/12345", s.URL()), nil)
	require.NoError(t, err)

	req.Header.Set(headerKey, headerVal)
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	assert.DeepEqual(t, []byte(`{"status": "ok"}`), body)

	downstream.AssertExpectations(t)
}

func TestMultiHeaderMatcher(t *testing.T) {
	headerKey := "HTTPMOCK-TEST"
	headerVal := "its here"
	headerKey2 := "HTTPMOCK-TEST-2"
	headerVal2 := "its here too!"
	downstream := NewMockHandlerWithHeaders(t)

	downstream.On(
		"HandleWithHeaders",
		"GET",
		"/object/12345",
		MultiHeaderMatcher(http.Header{
			headerKey:  []string{headerVal},
			headerKey2: []string{headerVal2},
		}),
		mock.Anything,
	).
		Return(Response{
			Body: []byte(`{"status": "ok"}`),
		})

	s := NewServer(downstream)
	defer s.Close()

	req, err := http.NewRequest("GET", fmt.Sprintf("%s/object/12345", s.URL()), nil)
	require.NoError(t, err)

	req.Header.Set(headerKey, headerVal)
	req.Header.Set(headerKey2, headerVal2)
	_, err = http.DefaultClient.Do(req)
	require.NoError(t, err)

	downstream.AssertExpectations(t)
}
