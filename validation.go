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
			if err := bodyValidation(in.Body); err != nil {
				return fmt.Errorf("request validation failed: custom body validation failure: %v", err.Error())
			}

			return nil
		}
	}

	bodyValidation = func(data []byte) RequestValidationFunc {
		return func(in *IncomingRequest) error {

			if bytes.Compare(data, in.Body) != 0 {
				return fmt.Errorf("request validation failed: body should be %v but was %v", string(data), string(in.Body))
			}

			return nil
		}
	}

	basicAuthValidation = func(user, password string) RequestValidationFunc {
		return func(in *IncomingRequest) error {
			_user, _password, ok := in.R.BasicAuth()
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
			if in.R.URL.Path != path {
				return fmt.Errorf("request validation failed: expected path %v but was %v", path, in.R.URL.Path)
			}

			return nil
		}
	}

	pathRegexValidation = func(pathRegex string) RequestValidationFunc {
		regex := regexp.MustCompile(pathRegex)
		return func(in *IncomingRequest) error {
			if !regex.MatchString(in.R.URL.Path) {
				return fmt.Errorf("request validation failed: pathRegex %v did not match %v", pathRegex, in.R.URL.Path)
			}

			return nil
		}
	}

	headerValidation = func(key, value string) RequestValidationFunc {
		return func(in *IncomingRequest) error {
			if in.R.Header.Get(key) == "" {
				return fmt.Errorf("request validation failed: header %v was missing", key)
			}

			if in.R.Header.Get(key) != value {
				return fmt.Errorf("request validation failed: expected header %v to be %v but was %v", key, value, in.R.Header.Get(key))
			}

			return nil
		}
	}

	formParameterValidation = func(key, value string) RequestValidationFunc {
		return func(in *IncomingRequest) error {
			if in.R.Form.Get(key) == "" {
				return fmt.Errorf("request validation failed: form parameter %v was missing", key)
			}

			if in.R.Form.Get(key) != value {
				return fmt.Errorf("request validation failed: expected form parameter %v to be %v but was %v", key, value, in.R.Form.Get(key))
			}

			return nil
		}
	}

	methodValidation = func(method string) RequestValidationFunc {
		return func(in *IncomingRequest) error {
			if strings.ToLower(in.R.Method) != strings.ToLower(method) {
				return fmt.Errorf("request validation failed: expected method %v but was %v", method, in.R.Method)
			}

			return nil
		}
	}
)
