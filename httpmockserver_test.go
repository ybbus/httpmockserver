package httpmockserver_test

import (
	"bytes"
	"errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/ybbus/httpmockserver"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

func TestMockServer_New(t *testing.T) {
	check := assert.New(t)

	t.Run("New should create a listening mockserver", func(t *testing.T) {
		mockServer := httpmockserver.New(t)
		defer mockServer.Shutdown()

		baseURL := mockServer.BaseURL()
		check.Regexp("^http://127.0.0.1:\\d+$", baseURL)

		mockServer.AssertExpectations()
	})

	t.Run("New should create a listening mockserver with custom options", func(t *testing.T) {
		mockServer := httpmockserver.NewWithOpts(t, httpmockserver.Opts{
			Port: "8080",
		})
		defer mockServer.Shutdown()

		baseURL := mockServer.BaseURL()
		check.Equal("http://127.0.0.1:8080", baseURL)

		mockServer.AssertExpectations()
	})

	t.Run("New should fail on invalid options", func(t *testing.T) {
		tMock := new(TMock)
		tMock.On("Fatalf", mock.Anything, mock.Anything)

		httpmockserver.NewWithOpts(tMock, httpmockserver.Opts{
			Port: "ABC",
		})
	})

	t.Run("New should fail on invalid options", func(t *testing.T) {
		tMock := new(TMock)
		tMock.On("Fatalf", mock.Anything, mock.Anything)

		httpmockserver.NewWithOpts(tMock, httpmockserver.Opts{
			UseSSL: true,
		})
	})

	t.Run("should fail if AssertExpectations()() was not called before Shutdown", func(t *testing.T) {
		tMock := new(TMock)
		tMock.On("Fatalf", mock.Anything, mock.Anything)

		mockServer := httpmockserver.New(tMock)
		defer mockServer.Shutdown()
	})
}

func TestMockServer_EVERY(t *testing.T) {
	check := assert.New(t)

	t.Run("EVERY should check expectations for all calls", func(t *testing.T) {
		tMock := new(TMock)

		mockServer := httpmockserver.New(tMock)
		defer mockServer.Shutdown()

		mockServer.EVERY().GET()

		mockServer.EXPECT().GET().AnyTimes().Response(200)
		res := get(mockServer.BaseURL(), "/test", nil)
		check.Equal(200, res.status)

		res = get(mockServer.BaseURL(), "/123/abc", nil)
		check.Equal(200, res.status)

		mockServer.AssertExpectations()
		tMock.AssertExpectations(t)
	})

	t.Run("EVERY should fail on missing expectation", func(t *testing.T) {
		tMock := new(TMock)
		tMock.On("Errorf", mock.Anything, mock.Anything).Twice()

		mockServer := httpmockserver.New(tMock)
		defer mockServer.Shutdown()

		mockServer.EVERY().Path("/test").Header("Content-Type", "application/json")
		mockServer.EXPECT().GET().AnyTimes().Response(200)

		res := get(mockServer.BaseURL(), "/test", Headers{"Content-Type": "application/json"})
		check.Equal(200, res.status)

		res = get(mockServer.BaseURL(), "/test", nil)
		check.Equal(200, res.status)

		// this should fail because the path is not /test
		res = get(mockServer.BaseURL(), "/123/abc", Headers{"Content-Type": "application/json"})
		check.Equal(200, res.status)

		mockServer.AssertExpectations()
		tMock.AssertExpectations(t)
	})

	t.Run("EVERY should fatal if tried to use for response", func(t *testing.T) {
		tMock := new(TMock)
		tMock.On("Fatalf", mock.Anything, mock.Anything)

		mockServer := httpmockserver.New(tMock)
		defer mockServer.Shutdown()

		mockServer.EVERY().GET().Path("/test").Response(200)

		mockServer.AssertExpectations()

		tMock.AssertExpectations(t)
	})
}

func TestMockServer_DEFAULT(t *testing.T) {
	check := assert.New(t)

	t.Run("DEFAULT should set default expectation", func(t *testing.T) {
		tMock := new(TMock)

		mockServer := httpmockserver.New(tMock)
		defer mockServer.Shutdown()

		mockServer.DEFAULT().GET().Response(201)
		mockServer.EXPECT().GET().Path("/abc").Response(200)

		res := get(mockServer.BaseURL(), "/abc", nil)
		check.Equal(200, res.status)

		res = get(mockServer.BaseURL(), "/default", nil)
		check.Equal(201, res.status)

		mockServer.AssertExpectations()
		tMock.AssertExpectations(t)
	})

	t.Run("DEFAULT should not match", func(t *testing.T) {
		tMock := new(TMock)
		// if nothing is found, Fatalf is called
		tMock.On("Fatalf", mock.Anything, mock.Anything)

		mockServer := httpmockserver.New(tMock)
		defer mockServer.Shutdown()

		mockServer.DEFAULT().Path("/default").Response(201)
		mockServer.EXPECT().GET().Path("/abc").Response(200)

		res := get(mockServer.BaseURL(), "/abc", nil)
		check.Equal(200, res.status)

		// default catches all methods
		res = get(mockServer.BaseURL(), "/default", nil)
		check.Equal(201, res.status)
		res = post(mockServer.BaseURL(), "/default", "", nil)
		check.Equal(201, res.status)

		// wrong path even for default: Fatalf is called
		res = get(mockServer.BaseURL(), "/wrong", nil)

		mockServer.AssertExpectations()
		tMock.AssertExpectations(t)
	})
}

func TestMockServer_EXPECT(t *testing.T) {
	check := assert.New(t)

	t.Run("EXPECT should check expectations for each call", func(t *testing.T) {
		tMock := new(TMock)

		mockServer := httpmockserver.New(tMock)
		defer mockServer.Shutdown()

		mockServer.EXPECT().GET().Path("/test").MinTimes(1).MaxTimes(1).Response(200).Header("Content-Type", "application/json").StringBody("test")
		mockServer.EXPECT().POST().Path("/123/abc").Once().Response(201)

		res := get(mockServer.BaseURL(), "/test", nil)
		check.Equal(200, res.status)
		check.Equal("application/json", res.header["Content-Type"][0])
		check.Equal("test", res.body)

		res = post(mockServer.BaseURL(), "/123/abc", "", nil)
		check.Equal(201, res.status)

		mockServer.AssertExpectations()
	})

	t.Run("EXPECT should fail on wrong number called", func(t *testing.T) {
		tMock := new(TMock)
		tMock.On("Fatalf", mock.Anything, mock.Anything)

		mockServer := httpmockserver.New(tMock)
		defer mockServer.Shutdown()

		mockServer.EXPECT().GET().Path("/test").Times(2).Response(200)

		res := get(mockServer.BaseURL(), "/test", nil)
		check.Equal(200, res.status)

		res = get(mockServer.BaseURL(), "/test", nil)
		check.Equal(200, res.status)

		res = get(mockServer.BaseURL(), "/test", nil)
		check.Equal(200, res.status)

		mockServer.AssertExpectations()
		tMock.AssertExpectations(t)
	})

	t.Run("EXPECT should fail on missing response", func(t *testing.T) {
		tMock := new(TMock)
		tMock.On("Fatalf", mock.Anything, mock.Anything)

		mockServer := httpmockserver.New(tMock)
		defer mockServer.Shutdown()

		mockServer.EXPECT().GET().Path("/test")

		res := get(mockServer.BaseURL(), "/test", nil)
		check.Equal(200, res.status)

		mockServer.AssertExpectations()
		tMock.AssertExpectations(t)
	})

	t.Run("EXPECT should fail on empty expectation", func(t *testing.T) {
		tMock := new(TMock)
		tMock.On("Fatalf", mock.Anything, mock.Anything)

		mockServer := httpmockserver.New(tMock)
		defer mockServer.Shutdown()

		mockServer.EXPECT().Response(200)

		res := get(mockServer.BaseURL(), "/test", nil)
		check.Equal(200, res.status)

		mockServer.AssertExpectations()
		tMock.AssertExpectations(t)
	})

	t.Run("EXPECT show potential unsatisfied expectation", func(t *testing.T) {
		tMock := new(TMock)
		// called when request did not match anything
		tMock.On("Fatalf", mock.Anything, mock.Anything).Once()

		// called for unmet expectations at the end of the test
		tMock.On("Fatalf", mock.Anything, mock.Anything).Once().Run(func(args mock.Arguments) {
			check.Contains(args[1].([]interface{})[0], "Header: Test:123 (never matched)")
		})

		mockServer := httpmockserver.New(tMock)
		defer mockServer.Shutdown()

		mockServer.EXPECT().GET().Path("/test").Header("Test", "123").Body([]byte("Hello World")).Response(200)

		get(mockServer.BaseURL(), "/test", nil)

		mockServer.AssertExpectations()
	})
}

func TestMockServer_PATHS(t *testing.T) {
	check := assert.New(t)

	t.Run("EXPECT should match path regex", func(t *testing.T) {
		tMock := new(TMock)

		mockServer := httpmockserver.New(tMock)
		defer mockServer.Shutdown()

		mockServer.EXPECT().GET().PathMatches(`^/test/\d+$`).Times(1).Response(201)
		mockServer.EXPECT().GetMatches(`/abc`).Times(4).Response(202)
		mockServer.DEFAULT().GET().Response(400)

		res := get(mockServer.BaseURL(), "/test/123", nil)
		check.Equal(201, res.status)

		res = get(mockServer.BaseURL(), "/abc", nil)
		check.Equal(202, res.status)

		res = get(mockServer.BaseURL(), "/abc/123", nil)
		check.Equal(202, res.status)

		res = get(mockServer.BaseURL(), "/test/abc/123", nil)
		check.Equal(202, res.status)

		res = get(mockServer.BaseURL(), "/test/abcde", nil)
		check.Equal(202, res.status)

		res = get(mockServer.BaseURL(), "/test", nil)
		check.Equal(400, res.status)

		res = get(mockServer.BaseURL(), "/test/wrong", nil)
		check.Equal(400, res.status)

		res = get(mockServer.BaseURL(), "/wrong/test/123", nil)
		check.Equal(400, res.status)

		res = get(mockServer.BaseURL(), "/not/xabc", nil)
		check.Equal(400, res.status)

		mockServer.AssertExpectations()
	})

	t.Run("EXPECT should fail on wrong number called", func(t *testing.T) {
		tMock := new(TMock)
		tMock.On("Fatalf", mock.Anything, mock.Anything)

		mockServer := httpmockserver.New(tMock)
		defer mockServer.Shutdown()

		mockServer.EXPECT().GET().Path("/test").Times(2).Response(200)

		res := get(mockServer.BaseURL(), "/test", nil)
		check.Equal(200, res.status)

		res = get(mockServer.BaseURL(), "/test", nil)
		check.Equal(200, res.status)

		res = get(mockServer.BaseURL(), "/test", nil)
		check.Equal(200, res.status)

		mockServer.AssertExpectations()
		tMock.AssertExpectations(t)
	})

	t.Run("EXPECT should fail on missing response", func(t *testing.T) {
		tMock := new(TMock)
		tMock.On("Fatalf", mock.Anything, mock.Anything)

		mockServer := httpmockserver.New(tMock)
		defer mockServer.Shutdown()

		mockServer.EXPECT().GET().Path("/test")

		res := get(mockServer.BaseURL(), "/test", nil)
		check.Equal(200, res.status)

		mockServer.AssertExpectations()
		tMock.AssertExpectations(t)
	})

	t.Run("EXPECT should fail on empty expectation", func(t *testing.T) {
		tMock := new(TMock)
		tMock.On("Fatalf", mock.Anything, mock.Anything)

		mockServer := httpmockserver.New(tMock)
		defer mockServer.Shutdown()

		mockServer.EXPECT().Response(200)

		res := get(mockServer.BaseURL(), "/test", nil)
		check.Equal(200, res.status)

		mockServer.AssertExpectations()
		tMock.AssertExpectations(t)
	})
}

func TestMockServer_Headers(t *testing.T) {
	check := assert.New(t)

	t.Run("EXPECT should match headers", func(t *testing.T) {
		tMock := new(TMock)

		mockServer := httpmockserver.New(tMock)
		defer mockServer.Shutdown()

		mockServer.EXPECT().Get("/test").Header("X-Test", "123").Times(1).Response(201)
		mockServer.EXPECT().Get("/test2").HeaderExists("X-Test2").Times(1).Response(202)
		mockServer.EXPECT().Get("/test3").Headers(map[string]string{"X-Test3": "123", "X-Test4": "456"}).Times(1).Response(203)
		mockServer.EXPECT().Get("/test4").HeaderMatches("X-Test5", `^abc`).Times(1).Response(204)
		mockServer.DEFAULT().GET().Response(400)

		res := get(mockServer.BaseURL(), "/test", map[string]string{"X-Test": "123"})
		check.Equal(201, res.status)

		res = get(mockServer.BaseURL(), "/test2", map[string]string{"X-Test2": "123"})
		check.Equal(202, res.status)

		res = get(mockServer.BaseURL(), "/test3", map[string]string{"X-Test3": "123", "X-Test4": "456"})
		check.Equal(203, res.status)

		res = get(mockServer.BaseURL(), "/test4", map[string]string{"X-Test5": "abc123"})
		check.Equal(204, res.status)

		res = get(mockServer.BaseURL(), "/test", map[string]string{"X-Test": "456"})
		check.Equal(400, res.status)

		mockServer.AssertExpectations()
	})

}

func TestMockServer_Forms(t *testing.T) {
	check := assert.New(t)

	t.Run("EXPECT should match forms", func(t *testing.T) {
		tMock := new(TMock)

		mockServer := httpmockserver.New(tMock)
		defer mockServer.Shutdown()

		mockServer.EXPECT().Path("/test").FormParameter("XTest", "123").Times(3).Response(201)

		req, err := http.PostForm(mockServer.BaseURL()+"/test", url.Values{"XTest": {"123"}})
		check.NoError(err)
		check.Equal(201, req.StatusCode)

		req, err = http.Post(mockServer.BaseURL()+"/test?XTest=123", "application/json", bytes.NewBuffer([]byte{}))
		check.NoError(err)
		check.Equal(201, req.StatusCode)

		req, err = http.Get(mockServer.BaseURL() + "/test?XTest=123")
		check.NoError(err)
		check.Equal(201, req.StatusCode)

		mockServer.AssertExpectations()
	})
}

func TestMockServer_Query(t *testing.T) {
	check := assert.New(t)

	t.Run("EXPECT should match url query parameters", func(t *testing.T) {
		tMock := new(TMock)

		mockServer := httpmockserver.New(tMock)
		defer mockServer.Shutdown()

		mockServer.EXPECT().Path("/test").QueryParameter("test", "123").Times(1).Response(201)
		mockServer.EXPECT().Path("/test2").QueryParameterMatches("test", "abc").Times(1).Response(202)
		mockServer.DEFAULT().Response(400)

		req, err := http.Get(mockServer.BaseURL() + "/test?test=123")
		check.NoError(err)
		check.Equal(201, req.StatusCode)

		req, err = http.Get(mockServer.BaseURL() + "/test2?test=xabcx")
		check.NoError(err)
		check.Equal(202, req.StatusCode)

		req, err = http.Get(mockServer.BaseURL() + "/test2?test=xxx")
		check.NoError(err)
		check.Equal(400, req.StatusCode)

		req, err = http.PostForm(mockServer.BaseURL()+"/test", url.Values{"test": {"123"}})
		check.NoError(err)
		check.Equal(400, req.StatusCode)

		mockServer.AssertExpectations()
	})
}

func TestMockServer_Auth(t *testing.T) {
	check := assert.New(t)

	t.Run("should match basic auth", func(t *testing.T) {
		tMock := new(TMock)

		mockServer := httpmockserver.New(tMock)
		defer mockServer.Shutdown()

		mockServer.EXPECT().Get("/test").BasicAuth("alice", "secret").Times(1).Response(201)
		mockServer.EXPECT().Get("/test2").BasicAuthExists().Times(1).Response(202)
		mockServer.DEFAULT().Response(400)

		req, _ := http.NewRequest("GET", mockServer.BaseURL()+"/test", nil)
		req.SetBasicAuth("alice", "secret")
		resp, err := http.DefaultClient.Do(req)
		check.NoError(err)
		check.Equal(201, resp.StatusCode)

		// wrong password
		req, _ = http.NewRequest("GET", mockServer.BaseURL()+"/test", nil)
		req.SetBasicAuth("alice", "wrong")
		resp, err = http.DefaultClient.Do(req)
		check.NoError(err)
		check.Equal(400, resp.StatusCode)

		// no basic auth
		req, _ = http.NewRequest("GET", mockServer.BaseURL()+"/test", nil)
		resp, err = http.DefaultClient.Do(req)
		check.NoError(err)
		check.Equal(400, resp.StatusCode)

		// basic auth exists
		req, _ = http.NewRequest("GET", mockServer.BaseURL()+"/test2", nil)
		req.SetBasicAuth("some", "thing")
		resp, err = http.DefaultClient.Do(req)
		check.NoError(err)
		check.Equal(202, resp.StatusCode)

		// no basic auth
		req, _ = http.NewRequest("GET", mockServer.BaseURL()+"/test2", nil)
		resp, err = http.DefaultClient.Do(req)
		check.NoError(err)
		check.Equal(400, resp.StatusCode)

		mockServer.AssertExpectations()
	})

	t.Run("should match jwt token auth", func(t *testing.T) {
		token := `eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c`
		tMock := new(TMock)

		mockServer := httpmockserver.New(tMock)
		defer mockServer.Shutdown()

		mockServer.EXPECT().Get("/test").JWTTokenExists().Times(1).Response(201)
		mockServer.DEFAULT().Response(400)

		req, _ := http.NewRequest("GET", mockServer.BaseURL()+"/test", nil)
		req.Header.Add("Authorization", "Bearer "+token)
		resp, err := http.DefaultClient.Do(req)
		check.NoError(err)
		check.Equal(201, resp.StatusCode)

		// missing token
		req, _ = http.NewRequest("GET", mockServer.BaseURL()+"/test", nil)
		resp, err = http.DefaultClient.Do(req)
		check.NoError(err)
		check.Equal(400, resp.StatusCode)

		mockServer.AssertExpectations()
	})

	t.Run("should contain value in claim", func(t *testing.T) {
		// contains: "name": "John Doe"
		token := `eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c`
		// contains: "name": "James"
		tokenJames := `eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkphbWVzIiwiaWF0IjoxNTE2MjM5MDIyfQ.kQ7pRlDsIMgWIu3jhQIEFqmiTJ3eBSc_3Jwl_58tK7Y`
		tMock := new(TMock)

		mockServer := httpmockserver.New(tMock)
		defer mockServer.Shutdown()

		mockServer.EXPECT().Get("/test").JWTTokenClaimPath("$.name", "John Doe").Times(1).Response(201)
		mockServer.DEFAULT().Response(400)

		req, _ := http.NewRequest("GET", mockServer.BaseURL()+"/test", nil)
		req.Header.Add("Authorization", "Bearer "+token)
		resp, err := http.DefaultClient.Do(req)
		check.NoError(err)
		check.Equal(201, resp.StatusCode)

		req, _ = http.NewRequest("GET", mockServer.BaseURL()+"/test", nil)
		req.Header.Add("Authorization", "Bearer "+tokenJames)
		resp, err = http.DefaultClient.Do(req)
		check.NoError(err)
		check.Equal(400, resp.StatusCode)

		// missing token
		req, _ = http.NewRequest("GET", mockServer.BaseURL()+"/test", nil)
		resp, err = http.DefaultClient.Do(req)
		check.NoError(err)
		check.Equal(400, resp.StatusCode)

		mockServer.AssertExpectations()
	})
}

func TestMockServer_Body(t *testing.T) {
	check := assert.New(t)

	t.Run("should match byte body", func(t *testing.T) {
		tMock := new(TMock)

		mockServer := httpmockserver.New(tMock)
		defer mockServer.Shutdown()

		mockServer.EXPECT().Post("/test").Body([]byte("Hello World!")).Times(1).Response(201)
		mockServer.DEFAULT().Response(400)

		res := post(mockServer.BaseURL(), "/test", "Hello World!", nil)
		check.Equal(201, res.status)

		res = post(mockServer.BaseURL(), "/test", "Something else", nil)
		check.Equal(400, res.status)

		res = post(mockServer.BaseURL(), "/test", "", nil)
		check.Equal(400, res.status)

		res = post(mockServer.BaseURL(), "/test", "nil", nil)
		check.Equal(400, res.status)

		mockServer.AssertExpectations()
	})

	t.Run("should match string body", func(t *testing.T) {
		tMock := new(TMock)

		mockServer := httpmockserver.New(tMock)
		defer mockServer.Shutdown()

		mockServer.EXPECT().Post("/test").StringBody("Hello World!").Times(1).Response(201)
		mockServer.DEFAULT().Response(400)

		res := post(mockServer.BaseURL(), "/test", "Hello World!", nil)
		check.Equal(201, res.status)

		res = post(mockServer.BaseURL(), "/test", "Something else", nil)
		check.Equal(400, res.status)

		res = post(mockServer.BaseURL(), "/test", "", nil)
		check.Equal(400, res.status)

		res = post(mockServer.BaseURL(), "/test", "nil", nil)
		check.Equal(400, res.status)

		mockServer.AssertExpectations()
	})

	t.Run("should match string body regex", func(t *testing.T) {
		tMock := new(TMock)

		mockServer := httpmockserver.New(tMock)
		defer mockServer.Shutdown()

		mockServer.EXPECT().Post("/test").StringBodyMatches(`^abcd\d+`).Times(1).Response(201)
		mockServer.DEFAULT().Response(400)

		res := post(mockServer.BaseURL(), "/test", "abcd1234", nil)
		check.Equal(201, res.status)

		res = post(mockServer.BaseURL(), "/test", "Test abcd123", nil)
		check.Equal(400, res.status)

		res = post(mockServer.BaseURL(), "/test", "abcd", nil)
		check.Equal(400, res.status)

		res = post(mockServer.BaseURL(), "/test", "", nil)
		check.Equal(400, res.status)

		res = post(mockServer.BaseURL(), "/test", "nil", nil)
		check.Equal(400, res.status)

		mockServer.AssertExpectations()
	})

	t.Run("should match contained substring", func(t *testing.T) {
		tMock := new(TMock)

		mockServer := httpmockserver.New(tMock)
		defer mockServer.Shutdown()

		mockServer.EXPECT().Post("/test").StringBodyContains(`something`).Times(1).Response(201)
		mockServer.DEFAULT().Response(400)

		res := post(mockServer.BaseURL(), "/test", "testsomethingtest", nil)
		check.Equal(201, res.status)

		res = post(mockServer.BaseURL(), "/test", "some thing", nil)
		check.Equal(400, res.status)

		res = post(mockServer.BaseURL(), "/test", "", nil)
		check.Equal(400, res.status)

		res = post(mockServer.BaseURL(), "/test", "nil", nil)
		check.Equal(400, res.status)

		mockServer.AssertExpectations()
	})

	t.Run("should match JSON body", func(t *testing.T) {
		tMock := new(TMock)

		mockServer := httpmockserver.New(tMock)
		defer mockServer.Shutdown()

		mockServer.EXPECT().Post("/test").JSONBody(map[string]interface{}{"name": "John", "age": 123}).Times(1).Response(201)
		mockServer.EXPECT().Post("/test2").JSONBody(`{ "name"   : "John",   "age":   123}`).Times(1).Response(201)
		mockServer.DEFAULT().Response(400)

		jsonBody := `{"age": 123, "name": "John"}`
		jsonBodyWrong := `{"age": 123, "name": "John", "extra": "field"}`
		jsonBodyInvalid := `{"age": 123, "name": "John"`

		res := post(mockServer.BaseURL(), "/test", jsonBody, nil)
		check.Equal(201, res.status)

		res = post(mockServer.BaseURL(), "/test2", jsonBody, nil)
		check.Equal(201, res.status)

		res = post(mockServer.BaseURL(), "/test", jsonBodyWrong, nil)
		check.Equal(400, res.status)

		res = post(mockServer.BaseURL(), "/test", jsonBodyInvalid, nil)
		check.Equal(400, res.status)

		res = post(mockServer.BaseURL(), "/test", "", nil)
		check.Equal(400, res.status)

		res = post(mockServer.BaseURL(), "/test", "nil", nil)
		check.Equal(400, res.status)

		mockServer.AssertExpectations()
	})

	t.Run("should match JSON path", func(t *testing.T) {
		tMock := new(TMock)

		mockServer := httpmockserver.New(tMock)
		defer mockServer.Shutdown()

		mockServer.EXPECT().Post("/test").JSONPathContains(`$.person.name`, "John").Times(1).Response(201)
		mockServer.DEFAULT().Response(400)

		jsonBody := `{"person": {"age": 123, "name": "John"}}`
		jsonBodyWrong := `{"age": 123, "name": "John", "extra": "field"}`
		jsonBodyInvalid := `{"age": 123, "name": "John"`

		res := post(mockServer.BaseURL(), "/test", jsonBody, nil)
		check.Equal(201, res.status)

		res = post(mockServer.BaseURL(), "/test", jsonBodyWrong, nil)
		check.Equal(400, res.status)

		res = post(mockServer.BaseURL(), "/test", jsonBodyInvalid, nil)
		check.Equal(400, res.status)

		res = post(mockServer.BaseURL(), "/test", "", nil)
		check.Equal(400, res.status)

		res = post(mockServer.BaseURL(), "/test", "nil", nil)
		check.Equal(400, res.status)

		mockServer.AssertExpectations()
	})

	t.Run("should execute bodyfunc", func(t *testing.T) {
		tMock := new(TMock)
		tMock.On("Fatalf", mock.Anything, mock.Anything)

		mockServer := httpmockserver.New(tMock)
		defer mockServer.Shutdown()

		called := false
		mockServer.EXPECT().Post("/test").BodyFunc(func(body []byte) error {
			check.Equal("test123", string(body))
			called = true
			return nil
		}).Times(1).Response(201)
		mockServer.EXPECT().Post("/test2").BodyFunc(func(body []byte) error {
			return errors.New("some error")
		}).Times(1).Response(201)
		mockServer.DEFAULT().Response(400)

		res := post(mockServer.BaseURL(), "/test", "test123", nil)
		check.Equal(201, res.status)

		res = post(mockServer.BaseURL(), "/test2", "test123", nil)

		check.True(called)

		mockServer.AssertExpectations()
	})
}

func TestMockServer_CustomRequestValidation(t *testing.T) {
	check := assert.New(t)

	t.Run("should execute custom validator", func(t *testing.T) {
		tMock := new(TMock)
		tMock.On("Fatalf", mock.Anything, mock.Anything)

		mockServer := httpmockserver.New(tMock)
		defer mockServer.Shutdown()

		called := false
		mockServer.EXPECT().Post("/test").Custom(func(r *httpmockserver.IncomingRequest) error {
			check.Equal("Hello World!", string(r.Body))
			check.Equal("POST", r.R.Method)
			called = true
			return nil
		}, "custom validator").Times(1).Response(201)
		mockServer.EXPECT().Post("/test2").Custom(func(r *httpmockserver.IncomingRequest) error {
			return errors.New("some error")
		}, "custom validator").Times(1).Response(202)
		mockServer.DEFAULT().Response(400)

		res := post(mockServer.BaseURL(), "/test", "Hello World!", nil)
		check.Equal(201, res.status)

		res = post(mockServer.BaseURL(), "/test2", "Hello World!", nil)
		check.Equal(400, res.status)

		check.True(called)

		mockServer.AssertExpectations()
	})
}

func TestMockServer_ResponseExpectation(t *testing.T) {
	check := assert.New(t)

	t.Run("should return body", func(t *testing.T) {
		tMock := new(TMock)
		tMock.On("Fatalf", mock.Anything, mock.Anything)

		mockServer := httpmockserver.New(tMock)
		defer mockServer.Shutdown()

		mockServer.EXPECT().Get("/test").Times(1).Response(201).Body([]byte("Hello World!")).Header("Content-Type", "text/plain")
		mockServer.EXPECT().Get("/test2").Times(1).Response(202).StringBody("Hello World!").Headers(Headers{"Content-Type": "text/plain"})
		mockServer.EXPECT().Get("/test3").Times(1).Response(203).JsonBody(map[string]string{"hello": "world"})
		mockServer.EXPECT().Get("/test4").Times(1).Response(204).JsonBody(nil)
		mockServer.EXPECT().Get("/test4").Times(1).Response(205).JsonBody("wrong json body")
		mockServer.DEFAULT().Response(400)

		res := get(mockServer.BaseURL(), "/test", nil)
		check.Equal(201, res.status)
		check.Equal("Hello World!", res.body)
		check.Equal("text/plain", res.header["Content-Type"][0])

		res = get(mockServer.BaseURL(), "/test2", nil)
		check.Equal(202, res.status)
		check.Equal("Hello World!", res.body)

		res = get(mockServer.BaseURL(), "/test3", nil)
		check.Equal(203, res.status)
		check.Equal(`{"hello":"world"}`, res.body)

		res = get(mockServer.BaseURL(), "/test4", nil)
		check.Equal(204, res.status)
		check.Equal("", res.body)

		mockServer.AssertExpectations()
	})
}

type Headers map[string]string

func get(baseUrl string, path string, header Headers) response {
	req, _ := http.NewRequest("GET", baseUrl+path, nil)
	for key, value := range header {
		req.Header.Add(key, value)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return response{
			status: 0,
			header: nil,
			body:   "",
			err:    err,
		}
	}
	body, _ := io.ReadAll(resp.Body)
	if body == nil {
		body = []byte{}
	}
	return response{
		status: resp.StatusCode,
		header: resp.Header,
		body:   string(body),
		err:    err,
	}
}

func post(baseUrl string, path string, body string, headers Headers) response {
	var bodyData io.Reader
	if body != "nil" {
		bodyData = strings.NewReader(body)
	}

	req, _ := http.NewRequest("POST", baseUrl+path, bodyData)
	for key, value := range headers {
		req.Header.Add(key, value)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return response{
			status: 0,
			header: nil,
			body:   "",
			err:    err,
		}
	}
	resBody, _ := io.ReadAll(resp.Body)
	return response{
		status: resp.StatusCode,
		header: resp.Header,
		body:   string(resBody),
		err:    err,
	}
}

type response struct {
	status int
	header map[string][]string
	body   string
	err    error
}

type TMock struct {
	mock.Mock
}

func (t *TMock) Fatal(args ...interface{}) {
	t.Called(args)
}

func (t *TMock) Fatalf(format string, args ...interface{}) {
	t.Called(format, args)
}

func (t *TMock) Errorf(format string, args ...interface{}) {
	t.Called(format, args)
}
