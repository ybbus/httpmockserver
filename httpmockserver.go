package httpmockserver

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
)

// Opts is used to configure the mock server
// it has reasonable defaults
type Opts struct {
	// Port is the port the mock server will listen on (default: random port)
	Port string
	// UseSSL is used to enable SSL (default: false)
	UseSSL bool
	// Cert is the certificate used for SSL
	Cert io.Reader
	// Key is the key used for SSL
	Key io.Reader
}

func (o *Opts) validate() error {
	if o.UseSSL && (o.Cert == nil || o.Key == nil) {
		return fmt.Errorf("UseSSL is set to true but no certificate or key is provided")
	}
	if o.Port == "" {
		o.Port = "0"
	}
	// check if port can be parsed to an integer atoi
	i, err := strconv.Atoi(o.Port)
	if err != nil {
		return fmt.Errorf("port is not a valid integer")
	}
	if i < 0 || i > 65535 {
		return fmt.Errorf("port is not a valid port number")
	}
	return nil
}

type MockServer interface {
	// BaseURL returns the base url of the mock server (default: http://127.0.0.1:<random_port>)
	BaseURL() string
	// ServeHTTP provides direct access to the http handler, normally this is not required
	ServeHTTP(w http.ResponseWriter, r *http.Request)
	// EVERY returns a RequestExpectation that will match on any call
	// (e.g. all requests should have a specific header, or all requests use GET)
	EVERY() RequestExpectation
	// EXPECT returns a RequestExpectation that can be used to create expectations
	// the default number of calls is expected to be exactly one
	// this can be changed by calling a method like: Times, MinTimes, MaxTimes, etc.
	EXPECT() RequestExpectation
	// DEFAULT returns a RequestExpectation that will be executed if no other expectation matches
	DEFAULT() RequestExpectation
	// AssertExpectations should be called to check if all expectations have been met
	AssertExpectations()
	// Shutdown should be called to stop the mock server (should be deferred at the beginning of the test function)
	Shutdown()
}

// New creates a new mock server running on http://127.0.0.1:<random_port>
func New(t T) MockServer {
	return NewWithOpts(t, Opts{})
}

type T interface {
	Fatal(args ...interface{})
	Fatalf(format string, args ...interface{})
	Errorf(format string, args ...interface{})
}

// NewWithOpts can be used to create a mock server with custom options
func NewWithOpts(t T, opts Opts) MockServer {
	err := opts.validate()
	if err != nil {
		t.Fatalf("invalid options: %v", err)
		return nil
	}

	mockServerInst := &mockServer{
		t: t,
	}

	// if port is not set to random (0) close the listener and change the port
	mockServerInst.server = httptest.NewUnstartedServer(mockServerInst)
	mockServerInst.server.Config.SetKeepAlivesEnabled(false)

	if opts.Port != "0" {
		mockServerInst.server.Listener.Close()
		l, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%s", opts.Port))
		if err != nil {
			t.Fatalf("httpmock: failed to listen on 127.0.0.1:%v: %v", opts.Port, err)
		}
		mockServerInst.server.Listener = l
	}

	if opts.UseSSL {
		if opts.Cert != nil && opts.Key != nil {
			key, _ := io.ReadAll(opts.Key)
			cert, _ := io.ReadAll(opts.Cert)

			xCert, err := tls.X509KeyPair(cert, key)
			if err != nil {
				t.Fatal("could not load certificate: ", err.Error())
			}

			mockServerInst.server.TLS = &tls.Config{}
			mockServerInst.server.TLS.NextProtos = []string{"http/1.1", "h2"}
			mockServerInst.server.TLS.Certificates = []tls.Certificate{xCert}
		}

		mockServerInst.server.StartTLS()
	} else {
		mockServerInst.server.Start()
	}

	return mockServerInst
}

type mockServer struct {
	server        *httptest.Server
	finisheCalled bool

	t T

	handlerMutex sync.Mutex

	every        []*requestExpectation
	expectations []*requestExpectation
	defaults     []*requestExpectation
}

func (s *mockServer) BaseURL() string {
	return s.server.URL
}

func (s *mockServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.handlerMutex.Lock()
	defer s.handlerMutex.Unlock()

	err := r.ParseForm()
	if err != nil {
		s.t.Fatal("could not parse form parameters of http request")
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		s.t.Fatal("request validation failed: could not read incoming request body: ", err.Error())
	}

	incomingRequest := &IncomingRequest{
		R:    r,
		Body: body,
	}

	// check EVERY expectation
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
		for _, reqVal := range exp.requestValidations {
			if err := reqVal.validation(incomingRequest); err != nil {
				continue outerExp
			}
			reqVal.satisfied = true
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
		s.t.Fatalf("Unexpected call:\nMethod: %v\nPath: %v\nHeaders: %v\nBody: %v", r.Method, r.URL.Path, r.Header, string(body))
		return
	}

	if matchedExpectation.response == nil {
		buf := bytes.Buffer{}
		for _, val := range matchedExpectation.requestValidations {
			buf.WriteString(fmt.Sprintf("----- %v\n", val.description))
		}

		s.t.Fatalf("Response not defined for expectation:\n%v", buf.String())
		return
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

func (s *mockServer) EVERY() RequestExpectation {
	exp := new(requestExpectation)
	exp.t = s.t
	exp.every = true

	s.every = append(s.every, exp)
	return exp
}

func (s *mockServer) EXPECT() RequestExpectation {
	exp := &requestExpectation{
		t:     s.t,
		count: 0,
		min:   1,
		max:   1,
	}

	s.expectations = append(s.expectations, exp)
	return exp
}

func (s *mockServer) DEFAULT() RequestExpectation {
	exp := &requestExpectation{
		t:          s.t,
		defaultExp: true,
	}

	s.defaults = append(s.defaults, exp)
	return exp
}

func (s *mockServer) AssertExpectations() {
	s.finisheCalled = true
	var buf bytes.Buffer

	unsatisfied := false
	for i, exp := range s.expectations {
		showFirstUnmatched := false
		if len(exp.requestValidations) == 0 {
			unsatisfied = true
			buf.WriteString(fmt.Sprintf("%v. Expectation\n", i+1))
			buf.WriteString("----- no request validation defined\n")
		}
		if exp.count < exp.min || exp.count > exp.max {
			unsatisfied = true
			buf.WriteString(fmt.Sprintf("%v. Expectation\n", i+1))
			for _, val := range exp.requestValidations {
				buf.WriteString(fmt.Sprintf("----- %v", val.description))
				if !val.satisfied && !showFirstUnmatched {
					showFirstUnmatched = true
					buf.WriteString(" (never matched)")
				}
				buf.WriteString("\n")
			}
			if exp.count < exp.min {
				buf.WriteString(fmt.Sprintf("----- only %v calls but at least %v were expected\n", exp.count, exp.min))
			} else if exp.count > exp.max {
				buf.WriteString(fmt.Sprintf("----- %v calls but at most %v were expected\n", exp.count, exp.max))
			}

		}
	}

	if unsatisfied {
		s.t.Fatalf("\nexpectation(s) not satisfied:\n%v", buf.String())
		return
	}
}

func (s *mockServer) Shutdown() {
	if !s.finisheCalled {
		s.t.Fatalf("AssertExpectations() was not called, no expectations were checked")
		return
	}
	s.handlerMutex.Lock()
	defer s.handlerMutex.Unlock()

	s.server.Close()
}
