package httpmockserver

import (
	"bytes"
	"fmt"
	"testing"
	"io/ioutil"
	"net/http"
	"sync"
	"net/http/httptest"
	"net"
)

func New(ssl bool, t *testing.T) *MockServer {
	return NewWithPort("0", ssl, t)
}

func NewWithPort(port string, ssl bool, t *testing.T) *MockServer {

	mockServer := &MockServer{
		t: t,
	}

	// if port is not set to random (0) close the listener and change the port
	mockServer.server = httptest.NewUnstartedServer(mockServer)
	mockServer.server.Config.SetKeepAlivesEnabled(false)

	if port != "0" {
		mockServer.server.Listener.Close()
		l, err := net.Listen("tcp", "127.0.0.1:"+port)
		if err != nil {
			panic(fmt.Sprintf("httptest: failed to listen on 127.0.0.1:%v: %v", port, err))
		}
		mockServer.server.Listener = l
	}

	if (ssl) {
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
	one          []*requestExpectation
	expectations []*requestExpectation
}

func (s *MockServer) GetURL() string {
	return s.server.URL
}

// TODO: should not be public
func (s *MockServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// only one request at a time
	s.handlerMutex.Lock()
	defer s.handlerMutex.Unlock()

	// check if we have expectations
	if len(s.expectations) == 0 {
		s.t.Fatalf("Missing expectation for %v %v", r.Method, r.URL.Path)
	}

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		s.t.Fatal("request validation failed: could not read incoming request body")
	}

	incomingRequest := &IncomingRequest{
		r:    r,
		body: body,
	}

	// check EVERY expectations
	for _, every := range s.every {
		for _, everyExp := range every.requestValidations {
			if err := everyExp.validation(incomingRequest); err != nil {
				s.t.Fatalf("EVERY expectation failed: %v", err.Error())
			}
		}
	}

	// check ONEOF expectations
	oneMatch := true
	for _, one := range s.one {
		oneMatch = true
		for _, oneExp := range one.requestValidations {
			if oneExp.validation(incomingRequest) != nil {
				oneMatch = false
				break
			}
		}
		if oneMatch {
			break
		}
	}

	if !oneMatch {
		s.t.Fatalf("request validation failed: no match in ONEOF constraint list for %v %v", r.Method, r.URL.Path)
	}

	exp := s.expectations[0]
	s.expectations = s.expectations[1:]

	// check request validations
	for _, reqVal := range exp.requestValidations {
		if err := reqVal.validation(incomingRequest); err != nil {
			s.t.Fatalf(err.Error())
		}
	}

	// build response
	for key, value := range exp.response.Headers {
		w.Header().Set(key, value)
	}

	w.WriteHeader(exp.response.Code)

	if exp.response.Body != nil {
		w.Write(exp.response.Body)
	}
}

// TODO: response expectation makes no sense here
func (s *MockServer) EVERY() RequestExpectation {
	exp := new(requestExpectation)
	exp.t = s.t

	s.every = append(s.every, exp)
	return exp
}

func (s *MockServer) ONEOF() RequestExpectation {
	exp := new(requestExpectation)
	exp.t = s.t

	s.one = append(s.one, exp)
	return exp
}

func (s *MockServer) EXPECT() RequestExpectation {
	exp := new(requestExpectation)
	exp.t = s.t

	// default response
	exp.response = &MockResponse{
		Code:    404,
		Headers: make(map[string]string),
	}

	s.expectations = append(s.expectations, exp)
	return exp
}

func (s *MockServer) Finish() {
	if len(s.expectations) != 0 {
		var buf bytes.Buffer

		for i, exp := range s.expectations {
			buf.WriteString(fmt.Sprintf("%v. Expectation\n", i+1))

			for _, val := range exp.requestValidations {
				buf.WriteString(fmt.Sprintf("----- %v\n", val.description))
			}
		}
		s.t.Fatalf("\nexpectations not satisfied:\n%v", buf.String())
	}
}

func (s *MockServer) Shutdown() {
	s.handlerMutex.Lock()
	defer s.handlerMutex.Unlock()

	s.server.Close()
}
