package httpmockserver

import (
	"encoding/json"
	"testing"
	"net/http"
)

type IncomingRequest struct {
	r    *http.Request
	body []byte
}

type RequestExpectation interface {
	AnyRequest() RequestExpectation

	Request(method string, path string) RequestExpectation
	Method(method string) RequestExpectation
	Path(path string) RequestExpectation
	PathRegex(pathRegex string) RequestExpectation

	GET() RequestExpectation
	POST() RequestExpectation
	PUT() RequestExpectation
	DELETE() RequestExpectation

	Get(path string) RequestExpectation
	Post(path string) RequestExpectation
	Put(path string) RequestExpectation
	Delete(path string) RequestExpectation

	Header(key, value string) RequestExpectation
	Headers(map[string]string) RequestExpectation

	BasicAuth(user, password string) RequestExpectation

	Custom(validation RequestValidationFunc, description string) RequestExpectation

	JsonBody(object interface{}) RequestExpectation
	StringBody(body string) RequestExpectation
	Body(body []byte) RequestExpectation
	BodyFunc(func(body []byte) error) RequestExpectation

	// switch to responseExpectations
	Response(code int) ResponseExpectation
}

type requestExpectation struct {
	t                  *testing.T
	requestValidations []*requestValidation
	response           *MockResponse
}

func (exp *requestExpectation) AnyRequest() RequestExpectation {
	return exp
}

func (exp *requestExpectation) Request(method string, path string) RequestExpectation {
	exp.Method(method)
	return exp.Path(path)
}

func (exp *requestExpectation) Method(method string) RequestExpectation {
	return exp.appendValidation(methodValidation(method), "Method: "+method)
}

func (exp *requestExpectation) Path(path string) RequestExpectation {
	return exp.appendValidation(pathValidation(path), "Path: "+path)
}

func (exp *requestExpectation) PathRegex(pathRegex string) RequestExpectation {
	return exp.appendValidation(pathRegexValidation(pathRegex), "PathRegex: "+pathRegex)
}

func (exp *requestExpectation) GET() RequestExpectation {
	return exp.appendValidation(methodValidation("GET"), "GET")
}

func (exp *requestExpectation) POST() RequestExpectation {
	return exp.appendValidation(methodValidation("POST"), "POST")
}

func (exp *requestExpectation) PUT() RequestExpectation {
	return exp.appendValidation(methodValidation("PUT"), "PUT")
}

func (exp *requestExpectation) DELETE() RequestExpectation {
	return exp.appendValidation(methodValidation("DELETE"), "DELETE")
}

func (exp *requestExpectation) Get(path string) RequestExpectation {
	return exp.Request("GET", path)
}

func (exp *requestExpectation) Post(path string) RequestExpectation {
	return exp.Request("POST", path)
}

func (exp *requestExpectation) Put(path string) RequestExpectation {
	return exp.Request("PUT", path)
}

func (exp *requestExpectation) Delete(path string) RequestExpectation {
	return exp.Request("DELETE", path)
}

func (exp *requestExpectation) Header(key, value string) RequestExpectation {
	return exp.appendValidation(headerValidation(key, value), "Header: "+key+":"+value)
}

func (exp *requestExpectation) Headers(headers map[string]string) RequestExpectation {
	for key, value := range headers {
		exp.Header(key, value)
	}
	return exp
}

func (exp *requestExpectation) BasicAuth(user, password string) RequestExpectation {
	return exp.appendValidation(basicAuthValidation(user, password), "Basic auth: "+user+":"+password)
}

func (exp *requestExpectation) Custom(validation RequestValidationFunc, description string) RequestExpectation {
	return exp.appendValidation(validation, description)
}

func (exp *requestExpectation) JsonBody(object interface{}) RequestExpectation {
	data, err := json.Marshal(object)
	if err != nil {
		exp.t.Fatalf("request validation failed: could not parse input body %+v", object)
	}

	return exp.Body(data)
}

func (exp *requestExpectation) StringBody(body string) RequestExpectation {
	return exp.Body([]byte(body))
}

func (exp *requestExpectation) Body(body []byte) RequestExpectation {
	return exp.appendValidation(bodyValidation(body), "Body: "+string(body))
}

func (exp *requestExpectation) BodyFunc(bodyValidation func(body []byte) error) RequestExpectation {
	return exp.appendValidation(bodyFuncValidation(bodyValidation), "custom body validation")
}

func (exp *requestExpectation) Response(code int) ResponseExpectation {
	responseExpectation := &responseExpectation{
		t:    exp.t,
		resp: exp.response,
	}

	responseExpectation.resp.Code = code

	return responseExpectation
}

func (exp *requestExpectation) appendValidation(validation RequestValidationFunc, description string) *requestExpectation {
	exp.requestValidations = append(exp.requestValidations, &requestValidation{validation, description})
	return exp
}