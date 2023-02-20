package httpmockserver

import (
	"encoding/json"
)

type MockResponse struct {
	Code    int
	Headers map[string]string
	Body    []byte
}

// ResponseExpectation is a builder for a MockResponse
// you may set Headers, Body, and Code on the response
// this response is returned to the caller when the corresponding request is matched
type ResponseExpectation interface {
	ContentType(contentType string) ResponseExpectation
	Header(key, value string) ResponseExpectation
	Headers(headers map[string]string) ResponseExpectation
	StringBody(body string) ResponseExpectation
	JsonBody(object interface{}) ResponseExpectation
	Body(data []byte) ResponseExpectation
}

type responseExpectation struct {
	resp *MockResponse
	t    T
}

// ContentType sets the content type header on the response
func (exp *responseExpectation) ContentType(contentType string) ResponseExpectation {
	exp.resp.Headers["Content-Type"] = contentType
	return exp
}

// Header sets a header on the response
func (exp *responseExpectation) Header(key, value string) ResponseExpectation {
	exp.resp.Headers[key] = value
	return exp
}

// Headers sets multiple headers on the response
func (exp *responseExpectation) Headers(headers map[string]string) ResponseExpectation {
	for key, value := range headers {
		exp.resp.Headers[key] = value
	}
	return exp
}

// StringBody sets the body of the response to the given string (e.g. "Hello World" or `{"foo":"bar"}`)
func (exp *responseExpectation) StringBody(body string) ResponseExpectation {
	return exp.Body([]byte(body))
}

// JsonBody sets the body of the response to the given object (e.g. `{"foo":"bar"}` or map[string]string{"foo":"bar"})
// you may provide a go object or a valid json string
// automatically sets the content type to application/json if ContentType is not set yet
func (exp *responseExpectation) JsonBody(object interface{}) ResponseExpectation {
	exp.t.Helper()

	// check if ContentType is set, if not set it to application/json
	if _, ok := exp.resp.Headers["Content-Type"]; !ok {
		exp.resp.Headers["Content-Type"] = "application/json"
	}

	if object == nil {
		return exp.Body(nil)
	}

	jsonBody, err := json.Marshal(object)
	if err != nil {
		exp.t.Fatalf("response expectation failed: could not parse to json: %+v", object)
	}

	return exp.Body(jsonBody)
}

// Body sets the body of the response to the given byte array (e.g. []byte("Hello World") or []byte(`{"foo":"bar"}`))
func (exp *responseExpectation) Body(data []byte) ResponseExpectation {
	exp.resp.Body = data
	return exp
}
