[![Go Report Card](https://goreportcard.com/badge/github.com/ybbus/httpmockserver)](https://goreportcard.com/report/github.com/ybbus/httpmockserver)
[![Codecov](https://codecov.io/github/ybbus/httpmockserver/branch/master/graph/badge.svg?token=ARYOQ8R1DT)](https://codecov.io/github/ybbus/httpmockserver)
[![GoDoc](https://godoc.org/github.com/ybbus/httpmockserver?status.svg)](https://godoc.org/github.com/ybbus/httpmockserver)
[![GitHub license](https://img.shields.io/github/license/mashape/apistatus.svg)]()

# HTTP Mock Server for golang
HTTP Mock Server provides an easy way to mock external http resources instead of mocking the whole application.

Supports:
- expectations of resources that should be requested during the test
    - request method (GET, POST, PUT, DELETE, ...)
    - request path (e.g. /api/v1/users)
    - request forms / query parameters
    - request headers
    - request auth headers (e.g. basic auth, jwt token)
    - request body
    - custom expectations (if none of the above is sufficient)
- response mocking
    - response status code
    - response headers
    - response body
- verification of expectations
    - verify that all expectations were met
    - verify that no unexpected requests were made
- default expectations as catch all (if no other expectation matches)
- "every" expectations, that matches on every request (e.g. Content-Type header for all must be application/json)
- starts a http server on a random port (you can also specify a port)
- supports http and https
- integrates into t *testing.T and returns helpful error messages

## Installation

```sh
go get -u github.com/ybbus/httpmockserver
```

## Getting started
To start simple, let's assume our application uses an external http service to retrieve user information.
We want to test our application, but we don't want to call the external service during the test.
We also don't want to mock the http client since we want to be sure that the resulting http request is correct.

The application retrieves a user information from the external service: GET /api/v1/users

So in our tests we want to expect a GET request to /api/v1/users and return a mocked response.

This can be easily done with httpmockserver:

```go
package main

import (
	"testing"
	"github.com/ybbus/httpmockserver"
)

func Test_Application(t *testing.T) {
	// create a new mock server
	server := httpmockserver.New(t)
	defer server.Shutdown()

	// set expectation(s)
	server.EXPECT().
		Get("/api/v1/users").
		Response(200).
		StringBody(`[{"id": 1, "name": "John Doe"}]`)

	// test application and use the base url of the mock server to initialize the application client
	baseUrl := server.BaseURL() // default: http://127.0.0.1:<random_port>

	server.AssertExpectations()
}
```

You will now have a http server running on a random port.
The server returns the mocked response when GET /api/v1/users is called.
If the call was missing, the test will fail with a helpful error message.

**Note:** The New call expects a *testing.T as parameter. This is required to integrate into the testing framework.
Don't be confused that it does not actually require a *testing.T, since it only implements the required methods.
This was done for better unit testing (mocking *testing.T), since the library depends heavily on *testing.T.

If you want to use the server without *testing.T, at all, you may just provide your own implementation of T interface.
```go
type T interface {
	Fatal(args ...interface{})
	Fatalf(format string, args ...interface{})
	Errorf(format string, args ...interface{})
}
```

## In detail

**Note:** Most of the examples just show the method calls, but you can also chain them together.
The examples don't make much sense without chaining them together but it makes them easier to read.

Also some combinations are quite useless like: `server.EXPECT().GET().POST()`.
A request cannot be both a GET and a POST request at the same time.

### EVERY() expectations

EVERY() expectations are matched on every request.
Normally you only need them if you expect a lot of calls and don't want to specify the same expectation for every call.

#### EVERY() for request expectations

A valid example would be to expect that all requests have the header "Content-Type: application/json" set:

```go
// expect all incoming requests to have the header "Content-Type: application/json" set
server.EVERY().Header("Content-Type", "application/json")
```

Or if you only expect GET requests to be called:
```go
// expect all incoming requests to be GET requests
server.EVERY().GET()
```

**Note:** EVERY() cannot set Responses, since it matches on every request and that would not make sense.

### DEFAULT() matcher

If you want to catch requests that do not match any expectation, you can use DEFAULT() as fallback.

The request is checked against the DEFAULT() expectations, only if all intended expectations do not match.

For example, if you would like to return 404 on all requests that do not match any expectation:

```go
// return 404 on all GET requests that do not match any expectation
server.DEFAULT().Response(404).StringBody(`{"error": "not found"}`)
```

This will also prevent the test to fail on additional requests that do not match any expectation.

### EXPECT() matcher

Use EXPECT() to set the actual expectations of the mock server.

First you define the validators that are used to match the incoming request.
Then you define the response that should be returned.

There are a lot of helper methods to set the validators.

**Note:** Each EXPECT() call must at least contain one validator (e.g. Path("/api/v1/users")). Otherwise the test will fail.

#### Number of times an expectation should be met

You can specify how often an expectation should be met.

**Note:** The default is Once() which means that the expectation should be met exactly once.

This can be done by just using the following methods:
```go
MinTimes(2) // should at least be called 2 times
MaxTimes(4) // should at most be called 2 times
```

There are some shortcuts for the most common cases:
```go
AnyTimes() // should be called any number of times
Once() // should be called exactly once
Twice() // should be called exactly twice
AtMostOnce() // should be called at most once
AtLeastOnce() // should be called at least once
Times(3) // should be called exactly 3 times
```

#### Request method and path

The following validation helpers are available for matching the request method and path:

```go
// expect a method without specifying a path
GET()
POST()
PUT()
DELETE()

// expect an exact path to match without specifying a method
Path("/api/v1/users")

// expect a method and an exact path
Get("/api/v1/users")
Post("/api/v1/users")

// expect a custom method and a path
Request("TRACE", "/api/v1/users")
or
Method("TRACE").Path("/api/v1/users")
```

For the path you may also use a regular expression:
```go
GetMatches(`^/abc/\d+$`) // to match /abc/123 etc.
PathMatches(`^/abc/\d+$`)
RequestMatches("POST", `^/abc/\d+$`)
```

**Note**:
- if no method expectation is set, the expectation will match on every method
- if no path expectation is set, the expectation will match on every path

#### Request headers

To validate if the request has a specific header set, you can use the following helpers:

```go
Header("Content-Type", "application/json") // to match the exact header value
HeaderMatches("Content-Type", `^application/(json|xml)$`) // to match application/json or application/xml
HeaderExists("Content-Type") // to check if the header exists

Headers(map[string]string{"Content-Type": "application/json", "Accept": "application/json"}) // to check multiple headers
//same as
Header("Content-Type", "application/json")
Header("Accept", "application/json")
```

**Note:** There may be additional headers in the request that are not specified in the expectation.
This won't cause the test to fail.

#### Request query / form parameters

```go
QueryParameter("page", "1")
QueryParameterMatches("page", `^\d+$`)
QueryParameterExists("page")
QueryParameters(map[string]string{"page": "1", "limit": "10"})

FormParameter("client_id", "abc")
FormParameterMatches("client_id", `user_.*`)
FormParameterExists("client_id")
FormParameters(map[string]string{"client_id": "user", "client_secret": "secret"})
```

**Note:** There may be additional query parameters in the request that are not specified in the expectation.
This won't cause the test to fail.

#### Authentication

You may want to check authentication headers. The following helpers are available:

```go
// Basic authentication
BasicAuth("alice", "secret") // to match the exact username and password
BasicAuthExists() // to check if Authorization header exists

// JWT token (bearer)
JWTTokenExists() // check if Authorization header with Bearer exists

// JWT Token has a specific claim using json path (see: see: https://github.com/oliveagle/jsonpath) 
JWTTokenClaimPath("$.name", "Jack") // check if token has a claim "name" the value "Jack"
```

#### Request body validators

The following validators are available to check the request body:

```go
Body([]byte("Hello World")) // to match the exact body "Hello World"
StringBody("Hello World") // same as Body([]byte("Hello World")), let you provide a string instead of a byte array
StringBodyContains("Hello") // to check if the body contains the string "Hello"
StringBodyMatches(`^Hello.*$`) // to check if the body matches the regular expression
JSONBody(object interface{}) // to check if the body is a valid json and matches the given object
JSONPathContains("$.name", "Jack") // to check if the json body contains the given json path (see: https://github.com/oliveagle/jsonpath)

BodyFunc(func(body []byte) error {
	// check if the body matches your custom logic
    return nil // or return an error if the body does not match
})
```

**Note:**

JSONBody expects a given request with a specific body. The body can be either be a go object that wil be parsed to a json string (e.g. `map[string]string{"foo":"bar"}`) or a json string (e.g. `{"foo":"bar"}`).
The body will be normalized (e.g. whitespace will be removed, fields will be sorted) and compared with the body by string equality.


### Response()

When you are done with the expectations, you can set the response that should be returned when the expectation is met.

The following methods are available to set the response:

**Note:** To switch from the expectation to the response, you must call Response(int) as first call.

```go
Response(200) // to set the status code
Header("Content-Type", "application/json") // to set a response header
Headers(map[string]string{"Content-Type": "application/json", "Accept": "application/json"}) // to set multiple response headers
StringBody("Hello World") // to set the response body as string
Body([]byte("Hello World")) // same as StringBody("Hello World"), let you provide a byte array instead of a string
JsonBody(object interface{}) // to set the response body as json (may provide a go object or a string that is valid json)
```

Example:
```go
server.EXPECT().
	  Post("/api/v1/users").
	  Header("Content-Type", "application/json").
	  JSONPathContains("$.name", "Jack").
	  Times(2).
	Response(201).
	  Header("Content-Type", "application/json").
	  StringBody(`{"id": 123, "name": "Jack"}`)
```