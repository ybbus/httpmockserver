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

### TODO