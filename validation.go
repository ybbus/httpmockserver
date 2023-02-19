package httpmockserver

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/form3tech-oss/jwt-go"
	"github.com/oliveagle/jsonpath"
	"reflect"
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

	stringBodyContainsValidation = func(substring string) RequestValidationFunc {
		return func(in *IncomingRequest) error {
			stringBody := string(in.Body)

			if !strings.Contains(stringBody, substring) {
				return fmt.Errorf("request validation failed: body should contain %v but was %v", substring, stringBody)
			}

			return nil
		}
	}

	stringBodyMatchValidation = func(regex string) RequestValidationFunc {
		return func(in *IncomingRequest) error {
			stringBody := string(in.Body)

			if !regexp.MustCompile(regex).MatchString(stringBody) {
				return fmt.Errorf("request validation failed: body should match %v but was %v", regex, stringBody)
			}

			return nil
		}
	}

	jsonBodyValidation = func(expectedJson interface{}) RequestValidationFunc {
		return func(in *IncomingRequest) error {
			var jsExpected []byte
			var err error

			if str, ok := expectedJson.(string); ok {
				jsExpected = []byte(str)
			} else {
				jsExpected, err = json.Marshal(expectedJson)
				if err != nil {
					return fmt.Errorf("request validation failed: could not parse provided json body %+v: %v", expectedJson, err)
				}
			}

			var normJsExpected map[string]interface{}
			err = json.Unmarshal(jsExpected, &normJsExpected)
			if err != nil {
				return fmt.Errorf("request validation failed: could not parse expected json body %+v: %v", expectedJson, err)
			}

			var normJsActual map[string]interface{}
			err = json.Unmarshal(in.Body, &normJsActual)
			if err != nil {
				return fmt.Errorf("request validation failed: could not parse actual json body %+v: %v", in.Body, err)
			}

			normStringActual, err := json.Marshal(normJsActual)
			normStringExpected, err := json.Marshal(normJsExpected)

			if bytes.Compare(normStringActual, normStringExpected) != 0 {
				return fmt.Errorf("request validation failed: json body should be %+v but was %+v", normStringExpected, normStringActual)
			}

			return nil
		}
	}

	jsonPathContainsValidation = func(jsPath string, value interface{}) RequestValidationFunc {
		return func(in *IncomingRequest) error {
			var jsBodyObject map[string]interface{}
			err := json.Unmarshal(in.Body, &jsBodyObject)
			if err != nil {
				return fmt.Errorf("request validation failed: could not parse json body %+v: %v", in.Body, err)
			}

			res, err := jsonpath.JsonPathLookup(jsBodyObject, jsPath)
			if err != nil {
				return fmt.Errorf("request validation failed: could not find json path %v in body %+v: %v", jsPath, in.Body, err)
			}

			if reflect.DeepEqual(res, value) {
				return nil
			}

			return fmt.Errorf("request validation failed: json path %v should be %+v but was %+v", jsPath, value, res)
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

	basicAuthExistsValidation = func() RequestValidationFunc {
		return func(in *IncomingRequest) error {
			_, _, ok := in.R.BasicAuth()
			if !ok {
				return fmt.Errorf("request validation failed: expected authHeader was missing")
			}

			return nil
		}
	}

	jwtTokenExistsValidation = func() RequestValidationFunc {
		return func(in *IncomingRequest) error {
			authHeader := in.R.Header.Get("Authorization")
			if authHeader == "" {
				return fmt.Errorf("request validation failed: expected authHeader was missing")
			}
			if !strings.HasPrefix(authHeader, "Bearer ") {
				return fmt.Errorf("request validation failed: Bearer prefix was missing")
			}

			token := strings.TrimPrefix(authHeader, "Bearer ")

			if token == "" {
				return fmt.Errorf("request validation failed: bearer token was empty")
			}

			return nil
		}
	}

	jwtTokenClaimPathValidation = func(jsPath string, value interface{}) RequestValidationFunc {
		return func(in *IncomingRequest) error {
			if err := jwtTokenExistsValidation()(in); err != nil {
				return err
			}

			token := strings.TrimPrefix(in.R.Header.Get("Authorization"), "Bearer ")

			parsedToken, _ := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
				return nil, nil
			})

			claims, ok := parsedToken.Claims.(jwt.MapClaims)
			if !ok {
				return fmt.Errorf("request validation failed: could not retrieve claims from token")
			}

			res, err := jsonpath.JsonPathLookup(claims, jsPath)
			if err != nil {
				return fmt.Errorf("request validation failed: could not retrieve claim with path %s from token", jsPath)
			}

			if !reflect.DeepEqual(res, value) {
				return fmt.Errorf("request validation failed: expected claim on path %s to be %v but was %v", jsPath, value, res)
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

	headerExistsValidation = func(key string) RequestValidationFunc {
		return func(in *IncomingRequest) error {
			if in.R.Header.Get(key) == "" {
				return fmt.Errorf("request validation failed: header %v was missing", key)
			}

			return nil
		}
	}

	headerMatchesValidation = func(key, regex string) RequestValidationFunc {
		return func(in *IncomingRequest) error {
			if in.R.Header.Get(key) == "" {
				return fmt.Errorf("request validation failed: header %v was missing", key)
			}

			if !regexp.MustCompile(regex).MatchString(in.R.Header.Get(key)) {
				return fmt.Errorf("request validation failed: header %v did not match regex %v", key, regex)
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

	formParameterExistsValidation = func(name string) RequestValidationFunc {
		return func(in *IncomingRequest) error {
			if in.R.Form.Get(name) == "" {
				return fmt.Errorf("request validation failed: form parameter %v was missing", name)
			}

			return nil
		}
	}

	formParameterMatchesValidation = func(key, regex string) RequestValidationFunc {
		return func(in *IncomingRequest) error {
			if in.R.Form.Get(key) == "" {
				return fmt.Errorf("request validation failed: form parameter %v was missing", key)
			}

			if !regexp.MustCompile(regex).MatchString(in.R.Form.Get(key)) {
				return fmt.Errorf("request validation failed: form parameter %v did not match regex %v", key, regex)
			}

			return nil
		}
	}

	queryParameterValidation = func(key, value string) RequestValidationFunc {
		return func(in *IncomingRequest) error {
			if in.R.URL.Query().Get(key) == "" {
				return fmt.Errorf("request validation failed: query parameter %v was missing", key)
			}

			if in.R.URL.Query().Get(key) != value {
				return fmt.Errorf("request validation failed: expected query parameter %v to be %v but was %v", key, value, in.R.Form.Get(key))
			}

			return nil
		}
	}

	queryParameterExistsValidation = func(name string) RequestValidationFunc {
		return func(in *IncomingRequest) error {
			if in.R.URL.Query().Get(name) == "" {
				return fmt.Errorf("request validation failed: query parameter %v was missing", name)
			}

			return nil
		}
	}

	queryParameterMatchesValidation = func(key, regex string) RequestValidationFunc {
		return func(in *IncomingRequest) error {
			if in.R.URL.Query().Get(key) == "" {
				return fmt.Errorf("request validation failed: query parameter %v was missing", key)
			}

			if !regexp.MustCompile(regex).MatchString(in.R.Form.Get(key)) {
				return fmt.Errorf("request validation failed: query parameter %v did not match regex %v", key, regex)
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
