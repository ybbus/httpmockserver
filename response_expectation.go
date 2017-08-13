package httpmockserver

import (
	"testing"
	"encoding/json"
)

type MockResponse struct {
	Code    int
	Headers map[string]string
	Body    []byte
}

type ResponseExpectation interface {
	Header(key, value string) ResponseExpectation
	Headers(headers map[string]string) ResponseExpectation
	StringBody(body string) ResponseExpectation
	JsonBody(object interface{}) ResponseExpectation
	Body(data []byte) ResponseExpectation
}

type responseExpectation struct {
	resp *MockResponse
	t    *testing.T
}

func (exp *responseExpectation) Header(key, value string) ResponseExpectation {
	exp.resp.Headers[key] = value
	return exp
}

func (exp *responseExpectation) Headers(headers map[string]string) ResponseExpectation {
	for key, value := range headers {
		exp.resp.Headers[key] = value
	}
	return exp
}

func (exp *responseExpectation) StringBody(body string) ResponseExpectation {
	return exp.Body([]byte(body))
}

func (exp *responseExpectation) JsonBody(object interface{}) ResponseExpectation {
	if object == nil {
		exp.Body(nil)
	}

	jsonBody, err := json.Marshal(object)
	if err != nil {
		exp.t.Fatalf("response expectation failed: could not parse to json: %+v", object)
	}

	return exp.Body(jsonBody)
}

func (exp *responseExpectation) Body(data []byte) ResponseExpectation {
	exp.resp.Body = data
	return exp
}
