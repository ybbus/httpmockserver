package httpmockserver

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

type Opts struct {
	Port   string
	UseSSL bool
}

func (o *Opts) validate() error {
	return nil
}

func New(t *testing.T) *MockServer {
	return NewWithOpts(t, Opts{})
}

func NewWithOpts(t *testing.T, opts Opts) *MockServer {
	err := opts.validate()
	if err != nil {
		t.Fatal(err)
	}

	mockServer := &MockServer{
		t: t,
	}

	// if port is not set to random (0) close the listener and change the port
	mockServer.server = httptest.NewUnstartedServer(mockServer)
	mockServer.server.Config.SetKeepAlivesEnabled(false)

	if opts.Port != "0" && opts.Port != "" {
		mockServer.server.Listener.Close()
		l, err := net.Listen("tcp", "127.0.0.1:"+opts.Port)
		if err != nil {
			t.Fatalf("httpmock: failed to listen on 127.0.0.1:%v: %v", opts.Port, err)
		}
		mockServer.server.Listener = l
	}

	if opts.UseSSL {
		mockServer.server.StartTLS()
	} else {
		mockServer.server.Start()
	}

	return mockServer
}

type MockServer struct {
	server *httptest.Server

	t *testing.T

	handlerMutex sync.Mutex

	every        []*requestExpectation
	expectations []*requestExpectation
	defaults     []*requestExpectation
}

func (s *MockServer) URL() string {
	return s.server.URL
}

func (s *MockServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.handlerMutex.Lock()
	defer s.handlerMutex.Unlock()

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		s.t.Fatal("request validation failed: could not read incoming request body: ", err.Error())
	}

	incomingRequest := &IncomingRequest{
		r:    r,
		body: body,
	}

	// check EVERY expectations
	for _, every := range s.every {
		for _, everyExp := range every.requestValidations {
			if err := everyExp.validation(incomingRequest); err != nil {
				s.t.Errorf("expectation failed: %v", err)
			}
		}
	}

	var matchedExpectation *requestExpectation
	// check if call matches an expectation
outerExp:
	for _, exp := range s.expectations {
		if exp.count >= exp.max {
			continue
		}

		for _, reqVal := range exp.requestValidations {
			if err := reqVal.validation(incomingRequest); err != nil {
				continue outerExp
			}
		}

		matchedExpectation = exp
		matchedExpectation.count++
		break
	}

	// if not matched any of the expectations
	if matchedExpectation == nil {
		// check if call matches a default
	outerDefaults:
		for _, exp := range s.defaults {
			for _, reqVal := range exp.requestValidations {
				if err := reqVal.validation(incomingRequest); err != nil {
					continue outerDefaults
				}
			}

			matchedExpectation = exp
			break
		}
	}

	// if no default found log request and return default code
	if matchedExpectation == nil {
		s.t.Fatalf("Unexpected call:\nMethod: %v\nURL: %v\nHeaders: %v\nBody: %v", r.Method, r.URL.Path, r.Header, string(body))
	}

	// build response
	for key, value := range matchedExpectation.response.Headers {
		w.Header().Set(key, value)
	}

	w.WriteHeader(matchedExpectation.response.Code)

	if matchedExpectation.response.Body != nil {
		w.Write(matchedExpectation.response.Body)
	}
}

// TODO: response expectation makes no sense here
func (s *MockServer) EVERY() RequestExpectation {
	exp := new(requestExpectation)
	exp.t = s.t

	s.every = append(s.every, exp)
	return exp
}

func (s *MockServer) EXPECT() RequestExpectation {
	exp := &requestExpectation{
		t:     s.t,
		count: 0,
		min:   1,
		max:   1,
	}

	// TODO: default response
	exp.response = &MockResponse{
		Code:    404,
		Headers: make(map[string]string),
	}

	s.expectations = append(s.expectations, exp)
	return exp
}

func (s *MockServer) DEFAULT() RequestExpectation {
	exp := &requestExpectation{
		t: s.t,
	}

	// TODO: default response
	exp.response = &MockResponse{
		Code:    404,
		Headers: make(map[string]string),
	}

	s.defaults = append(s.defaults, exp)
	return exp
}

func (s *MockServer) Finish() {
	var buf bytes.Buffer

	unsatisfied := false
	for i, exp := range s.expectations {
		if exp.count < exp.min || exp.count > exp.max {
			unsatisfied = true
			buf.WriteString(fmt.Sprintf("%v. Expectation\n", i+1))
			for _, val := range exp.requestValidations {
				buf.WriteString(fmt.Sprintf("----- %v\n", val.description))
			}

		}
	}

	if unsatisfied {
		s.t.Fatalf("\nexpectation(s) not satisfied:\n%v", buf.String())
	}
}

func (s *MockServer) Shutdown() {
	s.handlerMutex.Lock()
	defer s.handlerMutex.Unlock()

	s.server.Close()
}

type request struct {
	Method  string
	Headers map[string][]string
	URL     string
	Body    []byte
}
