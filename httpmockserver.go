package httpmockserver

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"regexp"
	"sync"
	"testing"
)

func New(port string, t *testing.T) *httpMock {
	if port == "" {
		port = "8081"
	}

	mock := &httpMock{}
	mock.t = t

	mutex := sync.Mutex{}

	var handler http.HandlerFunc = func(w http.ResponseWriter, r *http.Request) {
		// only one request at a time
		mutex.Lock()
		defer mutex.Unlock()

		// check if we have expectations
		if len(mock.expectations) == 0 {
			t.Fatalf("Missing expectation for %v %v", r.Method, r.URL.Path)
		}

		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			t.Fatal("request validation failed: could not read incoming request body")
		}

		incomingRequest := &incomingRequest{
			r:    r,
			body: body,
		}

		// check EVERY expectations
		for _, every := range mock.every {
			for _, everyExp := range every.requestValidations {
				if err := everyExp.val(incomingRequest); err != nil {
					t.Fatalf("EVERY expectation failed: %v", err.Error())
				}
			}
		}

		// check ONEOF expectations
		oneMatch := true
		for _, one := range mock.one {
			oneMatch = true
			for _, oneExp := range one.requestValidations {
				if oneExp.val(incomingRequest) != nil {
					oneMatch = false
					break
				}
			}
			if oneMatch {
				break
			}
		}

		if !oneMatch {
			t.Fatalf("request validation failed: no match in ONEOF constraint list for %v %v", r.Method, r.URL.Path)
		}

		exp := mock.expectations[0]
		mock.expectations = mock.expectations[1:]

		// check request validations
		for _, reqVal := range exp.requestValidations {
			if err := reqVal.val(incomingRequest); err != nil {
				t.Fatalf(err.Error())
			}
		}

		// build response
		w.WriteHeader(exp.response.code)

		for key, value := range exp.response.headers {
			w.Header().Set(key, value)
		}

		if exp.response.body != nil {
			w.Write(exp.response.body)
		}
	}

	server := &http.Server{
		Addr:    ":" + port,
		Handler: handler,
	}

	go func() {
		log.Fatal(server.ListenAndServe())
	}()

	mock.server = server
	return mock
}

type httpMock struct {
	every        []*expectation
	one          []*expectation
	expectations []*expectation
	server       *http.Server
	t            *testing.T
}

// TODO: response expectation makes no sense here
func (mock *httpMock) EVERY() RequestExpectation {
	exp := new(expectation)
	exp.t = mock.t

	mock.every = append(mock.every, exp)
	return exp
}

func (mock *httpMock) ONEOF() RequestExpectation {
	exp := new(expectation)
	exp.t = mock.t

	mock.one = append(mock.one, exp)
	return exp
}

func (mock *httpMock) Finish() {
	if len(mock.expectations) != 0 {
		var buf bytes.Buffer

		for i, exp := range mock.expectations {
			buf.WriteString(fmt.Sprintf("%v. Expectation\n", i+1))

			for _, val := range exp.requestValidations {
				buf.WriteString(fmt.Sprintf("----- %v\n", val.description))
			}
		}
		mock.t.Fatalf("\nexpectations not satisfied:\n%v", buf.String())
	}
}

func (mock *httpMock) Shutdown() {
	mock.server.Close()
}

func (mock *httpMock) EXPECT() RequestExpectation {
	exp := new(expectation)
	exp.t = mock.t

	// default response
	exp.response = &mockResponse{
		code: 404,
	}

	mock.expectations = append(mock.expectations, exp)
	return exp
}

type mockResponse struct {
	code    int
	headers map[string]string
	body    []byte
}

/*
REQUEST EXPECTATIONS
*/

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

	JsonBody(object interface{}) RequestExpectation
	StringBody(body string) RequestExpectation
	Body(body []byte) RequestExpectation
	BodyFunc(func(body []byte) error) RequestExpectation

	// switch to responseExpectations
	Response(code int) ResponseExpectation
}

func (exp *expectation) AnyRequest() RequestExpectation {
	return exp
}

func (exp *expectation) Request(method string, path string) RequestExpectation {
	exp.Method(method)
	return exp.Path(path)
}

func (exp *expectation) Method(method string) RequestExpectation {
	return exp.appendValidation(methodValidation(method), "Method: "+method)
}

func (exp *expectation) Path(path string) RequestExpectation {
	return exp.appendValidation(pathValidation(path), "Path: "+path)
}

func (exp *expectation) PathRegex(pathRegex string) RequestExpectation {
	return exp.appendValidation(pathRegexValidation(pathRegex), "PathRegex: "+pathRegex)
}

func (exp *expectation) GET() RequestExpectation {
	return exp.appendValidation(methodValidation("GET"), "GET")
}

func (exp *expectation) POST() RequestExpectation {
	return exp.appendValidation(methodValidation("POST"), "POST")
}

func (exp *expectation) PUT() RequestExpectation {
	return exp.appendValidation(methodValidation("PUT"), "PUT")
}

func (exp *expectation) DELETE() RequestExpectation {
	return exp.appendValidation(methodValidation("DELETE"), "DELETE")
}

func (exp *expectation) Get(path string) RequestExpectation {
	return exp.Request("GET", path)
}

func (exp *expectation) Post(path string) RequestExpectation {
	return exp.Request("POST", path)
}

func (exp *expectation) Put(path string) RequestExpectation {
	return exp.Request("PUT", path)
}

func (exp *expectation) Delete(path string) RequestExpectation {
	return exp.Request("DELETE", path)
}

func (exp *expectation) Header(key, value string) RequestExpectation {
	return exp.appendValidation(headerValidation(key, value), "Header: "+key+":"+value)
}

func (exp *expectation) Headers(headers map[string]string) RequestExpectation {
	for key, value := range headers {
		exp.Header(key, value)
	}
	return exp
}

func (exp *expectation) BasicAuth(user, password string) RequestExpectation {
	return exp.appendValidation(basicAuthValidation(user, password), "Basic auth: "+user+":"+password)
}

func (exp *expectation) JsonBody(object interface{}) RequestExpectation {
	data, err := json.Marshal(object)
	if err != nil {
		exp.t.Fatalf("request validation failed: could not parse input body %+v", object)
	}

	return exp.Body(data)
}

func (exp *expectation) StringBody(body string) RequestExpectation {
	return exp.Body([]byte(body))
}

func (exp *expectation) Body(body []byte) RequestExpectation {
	return exp.appendValidation(bodyValidation(body), "Body: "+string(body))
}

func (exp *expectation) BodyFunc(bodyValidation func(body []byte) error) RequestExpectation {
	return exp.appendValidation(bodyFuncValidation(bodyValidation), "custom body validation")
}

func (exp *expectation) Response(code int) ResponseExpectation {
	responseExpectation := &responseExpectation{
		t:    exp.t,
		resp: exp.response,
	}

	responseExpectation.resp.code = code

	return responseExpectation
}

func (exp *expectation) appendValidation(validation requestValidationFunc, description string) *expectation {
	exp.requestValidations = append(exp.requestValidations, &requestValidation{validation, description})
	return exp
}

type incomingRequest struct {
	r    *http.Request
	body []byte
}

type requestValidationFunc func(r *incomingRequest) error

var (
	bodyFuncValidation = func(bodyValidation func(body []byte) error) requestValidationFunc {
		return func(in *incomingRequest) error {
			if err := bodyValidation(in.body); err != nil {
				return fmt.Errorf("request validation failed: custom body validation failure: %v", err.Error())
			}

			return nil
		}
	}

	bodyValidation = func(data []byte) requestValidationFunc {
		return func(in *incomingRequest) error {

			if bytes.Compare(data, in.body) != 0 {
				return fmt.Errorf("request validation failed: body should be %v but was %v", string(data), string(in.body))
			}

			return nil
		}
	}

	basicAuthValidation = func(user, password string) requestValidationFunc {
		return func(in *incomingRequest) error {
			_user, _password, ok := in.r.BasicAuth()
			if !ok {
				return fmt.Errorf("request validation failed: expected authHeader was missing")
			}

			if user != _user || password != _password {
				return fmt.Errorf("request validation failed: expected authHeader user:password %v:%v but was %v:%v", user, password, _user, _password)
			}

			return nil
		}
	}

	pathValidation = func(path string) requestValidationFunc {
		return func(in *incomingRequest) error {
			if in.r.URL.Path != path {
				return fmt.Errorf("request validation failed: expected path %v but was %v", path, in.r.URL.Path)
			}

			return nil
		}
	}

	pathRegexValidation = func(pathRegex string) requestValidationFunc {
		regex := regexp.MustCompile(pathRegex)
		return func(in *incomingRequest) error {
			if !regex.MatchString(in.r.URL.Path) {
				return fmt.Errorf("request validation failed: expected pathRegex %v but was %v", pathRegex, in.r.URL.Path)
			}

			return nil
		}
	}

	headerValidation = func(key, value string) requestValidationFunc {
		return func(in *incomingRequest) error {
			if in.r.Header.Get(key) == "" {
				return fmt.Errorf("request validation failed: header %v was missing", key)
			}

			if in.r.Header.Get(key) != value {
				return fmt.Errorf("request validation failed: expected header %v to be %v but was %v", key, value, in.r.Header.Get(key))
			}

			return nil
		}
	}

	methodValidation = func(method string) requestValidationFunc {
		return func(in *incomingRequest) error {
			if in.r.Method != method {
				return fmt.Errorf("request validation failed: expected method %v but was %v", method, in.r.Method)
			}

			return nil
		}
	}
)

/*
RESPONSE EXPECTATIONS
*/

type ResponseExpectation interface {
	Header(key, value string) ResponseExpectation
	Headers(headers map[string]string) ResponseExpectation
	StringBody(body string) ResponseExpectation
	JsonBody(object interface{}) ResponseExpectation
	Body(data []byte) ResponseExpectation
}

type responseExpectation struct {
	resp *mockResponse
	t    *testing.T
}

func (exp *responseExpectation) Header(key, value string) ResponseExpectation {
	exp.resp.headers[key] = value
	return exp
}

func (exp *responseExpectation) Headers(headers map[string]string) ResponseExpectation {
	for key, value := range headers {
		exp.resp.headers[key] = value
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
	exp.resp.body = data
	return exp
}

type expectation struct {
	t                  *testing.T
	requestValidations []*requestValidation
	response           *mockResponse
}

type requestValidation struct {
	val         requestValidationFunc
	description string
}

func (val *requestValidation) String() string {
	return val.description
}
