package httpmockserver

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"
)

type RequestValidationFunc func(r *IncomingRequest) error

type requestValidation struct {
	validation  RequestValidationFunc
	description string
}

func (val *requestValidation) String() string {
	return val.description
}

var (
	bodyFuncValidation = func(bodyValidation func(body []byte) error) RequestValidationFunc {
		return func(in *IncomingRequest) error {
			if err := bodyValidation(in.body); err != nil {
				return fmt.Errorf("request validation failed: custom body validation failure: %v", err.Error())
			}

			return nil
		}
	}

	bodyValidation = func(data []byte) RequestValidationFunc {
		return func(in *IncomingRequest) error {

			if bytes.Compare(data, in.body) != 0 {
				return fmt.Errorf("request validation failed: body should be %v but was %v", string(data), string(in.body))
			}

			return nil
		}
	}

	basicAuthValidation = func(user, password string) RequestValidationFunc {
		return func(in *IncomingRequest) error {
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

	pathValidation = func(path string) RequestValidationFunc {
		return func(in *IncomingRequest) error {
			if in.r.URL.Path != path {
				return fmt.Errorf("request validation failed: expected path %v but was %v", path, in.r.URL.Path)
			}

			return nil
		}
	}

	pathRegexValidation = func(pathRegex string) RequestValidationFunc {
		regex := regexp.MustCompile(pathRegex)
		return func(in *IncomingRequest) error {
			if !regex.MatchString(in.r.URL.Path) {
				return fmt.Errorf("request validation failed: pathRegex %v did not match %v", pathRegex, in.r.URL.Path)
			}

			return nil
		}
	}

	headerValidation = func(key, value string) RequestValidationFunc {
		return func(in *IncomingRequest) error {
			if in.r.Header.Get(key) == "" {
				return fmt.Errorf("request validation failed: header %v was missing", key)
			}

			if in.r.Header.Get(key) != value {
				return fmt.Errorf("request validation failed: expected header %v to be %v but was %v", key, value, in.r.Header.Get(key))
			}

			return nil
		}
	}

	methodValidation = func(method string) RequestValidationFunc {
		return func(in *IncomingRequest) error {
			if strings.ToLower(in.r.Method) != strings.ToLower(method) {
				return fmt.Errorf("request validation failed: expected method %v but was %v", method, in.r.Method)
			}

			return nil
		}
	}
)
