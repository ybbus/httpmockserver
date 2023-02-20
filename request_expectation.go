package httpmockserver

import (
	"fmt"
	"math"
	"net/http"
)

type IncomingRequest struct {
	R    *http.Request
	Body []byte
}

// RequestExpectation is used to set expectations on incoming requests
// EVERY(): used to set expectations that are checked on every request
// EXPECT(): used to set expectations that are checked on a specific request (specified number of times)
// DEFAULT(): used to set expectations that are checked on a request if no other expectation matches
type RequestExpectation interface {
	// AnyTimes expects a given request any number of times (same as MinTimes(0).MaxTimes(∞))
	AnyTimes() RequestExpectation
	// Once expects a given request exactly once (same as Times(1))
	Once() RequestExpectation
	// Twice expects a given request exactly twice (same as Times(2))
	Twice() RequestExpectation
	// AtMostOnce expects a given request at most once (same as MinTimes(0).MaxTimes(1))
	AtMostOnce() RequestExpectation
	// AtLeastOnce expects a given request at least once (same as MinTimes(1).MaxTimes(∞))
	AtLeastOnce() RequestExpectation
	// Times expects a given request exactly n times (same as MinTimes(n).MaxTimes(n))
	Times(n int) RequestExpectation
	// MinTimes expects a given request at least n times (increases MaxTimes to at least n)
	MinTimes(n int) RequestExpectation
	// MaxTimes expects a given request at most n times (decreases MinTimes to at least n)
	MaxTimes(n int) RequestExpectation

	// Request expects a given request with a specific method and path
	Request(method string, path string) RequestExpectation
	// RequestMatches expects a given request with a specific method and path matching a regex (e.g. `^/foo/bar/\d+$`)
	RequestMatches(method string, pathRegex string) RequestExpectation
	// Method expects a given request with a specific method (e.g. GET, POST, PUT, DELETE)
	Method(method string) RequestExpectation
	// Path expects a given request with a specific path (e.g. /foo/bar)
	Path(path string) RequestExpectation
	// PathMatches expects a given request with a path matching a regex (e.g. `^/foo/bar/\d+$`)
	PathMatches(pathRegex string) RequestExpectation

	// GET expects a given request with a GET method
	// use if no path should be matched (otherwise use Get(path))
	GET() RequestExpectation
	// POST expects a given request with a POST method
	// use if no path should be matched (otherwise use Post(path))
	POST() RequestExpectation
	// PUT expects a given request with a PUT method
	// use if no path should be matched (otherwise use Put(path))
	PUT() RequestExpectation
	// PATCH expects a given request with a PATCH method
	// use if no path should be matched (otherwise use Patch(path))
	PATCH() RequestExpectation
	// DELETE expects a given request with a DELETE method
	// use if no path should be matched (otherwise use Delete(path))
	DELETE() RequestExpectation
	// HEAD expects a given request with a HEAD method
	// use if no path should be matched (otherwise use Head(path))
	HEAD() RequestExpectation

	// Get expects a given request with a GET method and a specific path (e.g. /foo/bar)
	Get(path string) RequestExpectation
	// GetMatches expects a given request with a GET method and a path matching a regex (e.g. `^/foo/bar/\d+$`)
	GetMatches(pathRegex string) RequestExpectation
	// Post expects a given request with a POST method and a specific path (e.g. /foo/bar)
	Post(path string) RequestExpectation
	// PostMatches expects a given request with a POST method and a path matching a regex (e.g. `^/foo/bar/\d+$`)
	PostMatches(pathRegex string) RequestExpectation
	// Put expects a given request with a PUT method and a specific path (e.g. /foo/bar)
	Put(path string) RequestExpectation
	// PutMatches expects a given request with a PUT method and a path matching a regex (e.g. `^/foo/bar/\d+$`)
	PutMatches(pathRegex string) RequestExpectation
	// Patch expects a given request with a PATCH method and a specific path (e.g. /foo/bar)
	Patch(path string) RequestExpectation
	// PatchMatches expects a given request with a PATCH method and a path matching a regex (e.g. `^/foo/bar/\d+$`)
	PatchMatches(pathRegex string) RequestExpectation
	// Delete expects a given request with a DELETE method and a specific path (e.g. /foo/bar)
	Delete(path string) RequestExpectation
	// DeleteMatches expects a given request with a DELETE method and a path matching a regex (e.g. `^/foo/bar/\d+$`)
	DeleteMatches(pathRegex string) RequestExpectation
	// Head expects a given request with a HEAD method and a specific path (e.g. /foo/bar)
	Head(path string) RequestExpectation
	// HeadMatches expects a given request with a HEAD method and a path matching a regex (e.g. `^/foo/bar/\d+$`)
	HeadMatches(pathRegex string) RequestExpectation

	// Header expects a given request with a specific header (e.g. "Content-Type", "application/json")
	Header(name, value string) RequestExpectation
	// HeaderMatches expects a given request with a header matching a regex (e.g. "Content-Type", `^application/(json|xml)$`)
	HeaderMatches(name, valueRegex string) RequestExpectation
	// HeaderExists expects a given request with a specific header (e.g. "Authorization")
	HeaderExists(name string) RequestExpectation
	// Headers expects a given request with specific list of headers
	Headers(map[string]string) RequestExpectation

	// FormParameter expects a given request with a specific form parameter (e.g. "foo", "bar")
	FormParameter(name, value string) RequestExpectation
	// FormParameterMatches expects a given request with a form parameter matching a regex (e.g. "foo", `^bar\d+$`)
	FormParameterMatches(name string, regex string) RequestExpectation
	// FormParameterExists expects a given request with a specific form parameter (e.g. "foo")
	FormParameterExists(name string) RequestExpectation
	// FormParameters expects a given request with specific list of form parameters
	FormParameters(map[string]string) RequestExpectation

	// QueryParameter expects a given request with a specific query parameter (e.g. "?foo=bar")
	QueryParameter(name, value string) RequestExpectation
	// QueryParameterMatches expects a given request with a query parameter matching a regex (e.g. "?foo=`^bar\d+$`)
	QueryParameterMatches(name string, regex string) RequestExpectation
	// QueryParameterExists expects a given request with a specific query parameter (e.g. "foo")
	QueryParameterExists(name string) RequestExpectation
	// QueryParameters expects a given request with specific list of query parameters
	QueryParameters(map[string]string) RequestExpectation

	// BasicAuth expects a given request with a specific basic auth username and password
	BasicAuth(user, password string) RequestExpectation
	// BasicAuthExists expects a given request with basic auth
	BasicAuthExists() RequestExpectation

	// JWTTokenExists expects a given request with a jwt auth token
	JWTTokenExists() RequestExpectation
	// JWTTokenClaimPath expects a given request with a jwt auth token containing a specific claim using jsonPath notation
	// (e.g. `$.foo.bar` for `{"foo":{"bar":"baz"}}`)
	// see: https://github.com/oliveagle/jsonpath
	JWTTokenClaimPath(jsonPath string, value interface{}) RequestExpectation

	// Body expects a given request with a specific body in bytes (e.g. []byte(`{"foo":"bar"}`))
	Body(body []byte) RequestExpectation
	// StringBody expects a given request with a specific body as string (e.g. `{"foo":"bar"}`)
	StringBody(body string) RequestExpectation
	// StringBodyContains expects a given request with a body containing a specific substring (e.g. `foo`)
	StringBodyContains(substring string) RequestExpectation
	// StringBodyMatches expects a given request with a body matching a regex (e.g. `^abcd\d+$`)
	StringBodyMatches(regex string) RequestExpectation
	// JSONBody expects a given request with a specific body.
	// The body can be either a go object that wil be parsed to a json string (e.g. `map[string]string{"foo":"bar"}`)
	// or a json string (e.g. `{"foo":"bar"}`).
	// The body will be normalized (e.g. whitespace will be removed, fields will be sorted) and compared by string equality.
	JSONBody(object interface{}) RequestExpectation
	// JSONPathContains expects a given request with a body containing a specific json value using jsonPath notation
	// see: https://github.com/oliveagle/jsonpath
	JSONPathContains(jsonPath string, value interface{}) RequestExpectation

	// BodyFunc expects a given request with a custom validation function
	// you can use the provided body to do arbitrary validation
	// return nil if the request matched the given requirements
	// if an error is returned, another expectation is tried (or the default expectation is used, if any)
	BodyFunc(func(body []byte) error) RequestExpectation

	// Custom expects a given request with a custom validation function
	// return nil if the request matched the given requirements
	// if an error is returned, another expectation is tried (or the default expectation is used, if any)
	Custom(validation RequestValidationFunc, description string) RequestExpectation

	// Response returns the given status code and switches to response expectation mode
	// where you can specify the response body and headers
	Response(code int) ResponseExpectation
}

type requestExpectation struct {
	t                  T
	count              int
	min                int
	max                int
	requestValidations []*requestValidation
	response           *MockResponse
	every              bool
	defaultExp         bool
}

func (exp *requestExpectation) Times(n int) RequestExpectation {
	exp.min = n
	exp.max = n
	return exp
}

func (exp *requestExpectation) MinTimes(n int) RequestExpectation {
	exp.min = n
	if exp.max == 1 {
		exp.max = math.MaxInt32
	}
	if exp.max < exp.min {
		exp.max = exp.min
	}
	return exp
}

func (exp *requestExpectation) MaxTimes(n int) RequestExpectation {
	exp.max = n
	if exp.min > exp.max {
		exp.min = exp.max
	}
	return exp
}

func (exp *requestExpectation) AnyTimes() RequestExpectation {
	exp.min = 0
	exp.max = math.MaxInt32
	return exp
}

func (exp *requestExpectation) Once() RequestExpectation {
	return exp.Times(1)
}

func (exp *requestExpectation) Twice() RequestExpectation {
	return exp.Times(2)
}

func (exp *requestExpectation) AtMostOnce() RequestExpectation {
	exp.min = 0
	exp.max = 1
	return exp
}

func (exp *requestExpectation) AtLeastOnce() RequestExpectation {
	exp.min = 1
	exp.max = math.MaxInt32
	return exp
}

func (exp *requestExpectation) Request(method string, path string) RequestExpectation {
	exp.Method(method)
	return exp.Path(path)
}

func (exp *requestExpectation) RequestMatches(method string, pathRegex string) RequestExpectation {
	exp.Method(method)
	return exp.PathMatches(pathRegex)
}

func (exp *requestExpectation) Method(method string) RequestExpectation {
	return exp.appendValidation(methodValidation(method), "Method: "+method)
}

func (exp *requestExpectation) Path(path string) RequestExpectation {
	return exp.appendValidation(pathValidation(path), "Path: "+path)
}

func (exp *requestExpectation) PathMatches(regex string) RequestExpectation {
	return exp.appendValidation(pathRegexValidation(regex), "PathMatches: "+regex)
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

func (exp *requestExpectation) PATCH() RequestExpectation {
	return exp.appendValidation(methodValidation("PATCH"), "PATCH")
}

func (exp *requestExpectation) DELETE() RequestExpectation {
	return exp.appendValidation(methodValidation("DELETE"), "DELETE")
}

func (exp *requestExpectation) HEAD() RequestExpectation {
	return exp.appendValidation(methodValidation("HEAD"), "HEAD")
}

func (exp *requestExpectation) Get(path string) RequestExpectation {
	return exp.Request("GET", path)
}

func (exp *requestExpectation) GetMatches(regex string) RequestExpectation {
	return exp.RequestMatches("GET", regex)
}

func (exp *requestExpectation) Post(path string) RequestExpectation {
	return exp.Request("POST", path)
}

func (exp *requestExpectation) PostMatches(regex string) RequestExpectation {
	return exp.RequestMatches("POST", regex)
}

func (exp *requestExpectation) Put(path string) RequestExpectation {
	return exp.Request("PUT", path)
}

func (exp *requestExpectation) PutMatches(regex string) RequestExpectation {
	return exp.RequestMatches("PUT", regex)
}

func (exp *requestExpectation) Patch(path string) RequestExpectation {
	return exp.Request("PATCH", path)
}

func (exp *requestExpectation) PatchMatches(regex string) RequestExpectation {
	return exp.RequestMatches("PATCH", regex)
}

func (exp *requestExpectation) Delete(path string) RequestExpectation {
	return exp.Request("DELETE", path)
}

func (exp *requestExpectation) DeleteMatches(regex string) RequestExpectation {
	return exp.RequestMatches("DELETE", regex)
}

func (exp *requestExpectation) Head(path string) RequestExpectation {
	return exp.Request("HEAD", path)
}

func (exp *requestExpectation) HeadMatches(regex string) RequestExpectation {
	return exp.RequestMatches("HEAD", regex)
}

func (exp *requestExpectation) Header(name, value string) RequestExpectation {
	return exp.appendValidation(headerValidation(name, value), "Header: "+name+":"+value)
}

func (exp *requestExpectation) HeaderExists(name string) RequestExpectation {
	return exp.appendValidation(headerExistsValidation(name), "HeaderExists: "+name)
}

func (exp *requestExpectation) HeaderMatches(name, regex string) RequestExpectation {
	return exp.appendValidation(headerMatchesValidation(name, regex), "HeaderMatches: "+name+":"+regex)
}

func (exp *requestExpectation) Headers(headers map[string]string) RequestExpectation {
	for name, value := range headers {
		exp.Header(name, value)
	}
	return exp
}

func (exp *requestExpectation) FormParameter(name, value string) RequestExpectation {
	return exp.appendValidation(formParameterValidation(name, value), "FormParameter: "+name+":"+value)
}

func (exp *requestExpectation) FormParameterExists(name string) RequestExpectation {
	return exp.appendValidation(formParameterExistsValidation(name), "FormParameterExists: "+name)
}

func (exp *requestExpectation) FormParameterMatches(name string, regex string) RequestExpectation {
	return exp.appendValidation(formParameterMatchesValidation(name, regex), "FormParameterMatches: "+name+":"+regex)
}

func (exp *requestExpectation) FormParameters(formParameters map[string]string) RequestExpectation {
	for key, value := range formParameters {
		exp.FormParameter(key, value)
	}
	return exp
}

func (exp *requestExpectation) QueryParameter(name, value string) RequestExpectation {
	return exp.appendValidation(queryParameterValidation(name, value), "QueryParameter: "+name+":"+value)
}

func (exp *requestExpectation) QueryParameterExists(name string) RequestExpectation {
	return exp.appendValidation(queryParameterExistsValidation(name), "QueryParameterExists: "+name)
}

func (exp *requestExpectation) QueryParameterMatches(name string, regex string) RequestExpectation {
	return exp.appendValidation(queryParameterMatchesValidation(name, regex), "QueryParameterMatches: "+name+":"+regex)
}

func (exp *requestExpectation) QueryParameters(queryParameters map[string]string) RequestExpectation {
	for key, value := range queryParameters {
		exp.QueryParameter(key, value)
	}
	return exp
}

func (exp *requestExpectation) BasicAuth(user, password string) RequestExpectation {
	return exp.appendValidation(basicAuthValidation(user, password), "Basic auth: "+user+":"+password)
}

func (exp *requestExpectation) BasicAuthExists() RequestExpectation {
	return exp.appendValidation(basicAuthExistsValidation(), "Basic auth exists")
}

func (exp *requestExpectation) JWTTokenExists() RequestExpectation {
	return exp.appendValidation(jwtTokenExistsValidation(), "JWT token exists")
}

func (exp *requestExpectation) JWTTokenClaimPath(jsonPath string, value interface{}) RequestExpectation {
	return exp.appendValidation(jwtTokenClaimPathValidation(jsonPath, value), "JWT token claim path: "+jsonPath)
}

func (exp *requestExpectation) JSONBody(expected interface{}) RequestExpectation {
	return exp.appendValidation(jsonBodyValidation(expected), "JSONBody: "+fmt.Sprintf("%+v", expected))
}

func (exp *requestExpectation) JSONPathContains(jsonPath string, value interface{}) RequestExpectation {
	return exp.appendValidation(jsonPathContainsValidation(jsonPath, value), "JSONPathContains: "+jsonPath)
}

func (exp *requestExpectation) StringBody(body string) RequestExpectation {
	return exp.Body([]byte(body))
}

func (exp *requestExpectation) StringBodyContains(substring string) RequestExpectation {
	return exp.appendValidation(stringBodyContainsValidation(substring), "StringBodyContains: "+substring)
}

func (exp *requestExpectation) StringBodyMatches(regex string) RequestExpectation {
	return exp.appendValidation(stringBodyMatchValidation(regex), "StringBodyMatches: "+regex)
}

func (exp *requestExpectation) Body(body []byte) RequestExpectation {
	return exp.appendValidation(bodyValidation(body), "Body: "+string(body))
}

func (exp *requestExpectation) BodyFunc(bodyValidation func(body []byte) error) RequestExpectation {
	return exp.appendValidation(bodyFuncValidation(bodyValidation), "BodyFunc")
}

func (exp *requestExpectation) Custom(validation RequestValidationFunc, description string) RequestExpectation {
	return exp.appendValidation(validation, description)
}

func (exp *requestExpectation) Response(code int) ResponseExpectation {
	exp.t.Helper()
	if exp.every {
		exp.t.Fatalf("Every is used to check conditions on every request, therefore it cannot be used with Response()")
		return nil
	}

	if len(exp.requestValidations) == 0 && !exp.defaultExp {
		exp.t.Fatalf("no request validation specified")
	}

	exp.response = &MockResponse{
		Code:    code,
		Headers: make(map[string]string),
	}

	responseExpectation := &responseExpectation{
		t:    exp.t,
		resp: exp.response,
	}

	responseExpectation.resp.Code = code

	return responseExpectation
}

func (exp *requestExpectation) appendValidation(validation RequestValidationFunc, description string) *requestExpectation {
	exp.requestValidations = append(exp.requestValidations, &requestValidation{validation, description, false})
	return exp
}
