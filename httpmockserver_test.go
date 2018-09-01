package httpmockserver_test

import (
	"bytes"
	"github.com/stretchr/testify/assert"
	"github.com/ybbus/httpmockserver"
	"net/http"
	"testing"
)

func TestMockServer_EXPECT(t *testing.T) {
	check := assert.New(t)

	tests := []struct {
		Description string
		Run         func(mockServer *httpmockserver.MockServer, url string)
	}{
		{
			Description: "simple get on /hello",
			Run: func(mockServer *httpmockserver.MockServer, url string) {
				mockServer.EXPECT().Get("/hello").Response(200)

				res, err := Get(url+"/hello", nil)
				check.NoError(err)
				check.Equal(200, res.StatusCode)
			},
		},
		{
			Description: "simple get on /hello with header",
			Run: func(mockServer *httpmockserver.MockServer, url string) {
				mockServer.EXPECT().Get("/hello").Header("Test", "123").Response(200)

				res, err := Get(url+"/hello", map[string]string{"Test": "123"})
				check.NoError(err)
				check.Equal(200, res.StatusCode)
			},
		},
		{
			Description: "default calls",
			Run: func(mockServer *httpmockserver.MockServer, url string) {
				mockServer.DEFAULT().AnyRequest().Response(201)
				mockServer.EXPECT().Get("/hello").Header("Test", "123").Response(200)

				res, err := Get(url+"/hello", map[string]string{"Test": "123"})
				check.NoError(err)
				check.Equal(200, res.StatusCode)

				res, err = Get(url+"/hello", map[string]string{"Test": "123"})
				check.NoError(err)
				check.Equal(201, res.StatusCode)

				res, err = Post(url+"/test", map[string]string{"Test": "123"}, []byte("Hello World"))
				check.NoError(err)
				check.Equal(201, res.StatusCode)
			},
		},
	}

	for _, test := range tests {
		func() {
			server := httpmockserver.New(t)
			defer server.Shutdown()

			test.Run(server, server.URL())

			server.Finish()
		}()
	}

}

func Get(url string, headers map[string]string) (*http.Response, error) {
	c := http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		panic(err)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	return c.Do(req)
}

func Post(url string, headers map[string]string, body []byte) (*http.Response, error) {
	c := http.Client{}
	req, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		panic(err)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	return c.Do(req)
}
