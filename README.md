[![Go Report Card](https://goreportcard.com/badge/github.com/ybbus/httpmockserver)](https://goreportcard.com/report/github.com/ybbus/httpmockserver)
[![GitHub license](https://img.shields.io/github/license/mashape/apistatus.svg)]()

# HTTP Mock Server for golang
HTTP Mock Server provides a way to easily mock externel http systems in your integration tests.
Often the code under test access external resources that are not available during test or should not be used.

Sometimes it is not enough to mock the client that access these external resources because one is also interested in the validation of the correct http call to this external resources.

You are able to verify the correctness of the HTTP request that arrives at the external (mocked) system and you are able to set a response for every request.

That could be:
- check if the auth header is set
- check if the request method was GET / POST ...
- check if path was correct
- check if body was provided correctly

## Installation

```sh
go get -u github.com/ybbus/httpmockserver
```

# Getting started
Let's say we want to test (a part of) our application with an integration or system test.
We know that the application makes use of an external REST service API.
Of course we could mock the part that does the REST calls away, but we want to achieve two things with our test:
- test the system as-is, so do not mock any internals. as you will see this will be much easier
- check if the external resource would have been queried in the correct way (e.g. the authentication header is set correctly)

```go
func TestIntegration(t *testing.T) {
    // start our mock-server
    mockServer := httpmockserver.New(t)

    // set expectations (anywhere) in your tests, but of course before the actual call
    // first we expect a GET to path person /person
    mockServer.EXPECT().Get("/person")

    // second we expect a Post to any path
    mockServer.EXPECT().POST()

    // third we expect a GET to /persons/{id}
    mockServer.EXPECT().Get().PathRegex("/persons/.*").Response(200).StringBody(`{"name": "Alex"}`)

    // do your actual tests that internally should satisfy our expectations
    aClient := NewClient(mockServer.getURL())
    myTestComponent := NewComponent(aClient)

    // http calls should happen here internally
    // so we do not mock the client but check if the calls work as expected
    myTestComponent.DoSomething()

    // tell the mockserver we are done after the tests so it can check for missing calls
    mockServer.Finish

    // shutdown the mock-server after the tests
    defer mockServer.Shutdown()
}
```

## In detail
TODO

