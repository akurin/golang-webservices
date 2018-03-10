package main


import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
)

var _ = strconv.Atoi // build fails if strconv is imported and not used

func writeHeader(w http.ResponseWriter, result handleResult) {
	if result.err == nil {
		return
	}

	if apiErr, ok := result.err.(ApiError); ok {
		w.WriteHeader(apiErr.HTTPStatus)
		return
	}

	w.WriteHeader(http.StatusInternalServerError)
}

func writeBody(w http.ResponseWriter, result handleResult) {
	body := Body{
		Response: result.response,
	}
	if result.err != nil {
		body.Error = result.err.Error()
	}
	bodyBytes, _ := json.Marshal(body)
	w.Write(bodyBytes)
}

type handleResult struct {
	err      error
	response interface{}
}

type Body struct {
	Error    string      "json:\"error\""
	Response interface{} "json:\"response,omitempty\""
}


func (api *MyApi) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var handleResult handleResult
	switch r.URL.Path {
	case "/user/profile":
		var param ProfileParams
		formValueLogin := r.FormValue("login")
		if formValueLogin == "" {
			handleResult.err = ApiError{
				HTTPStatus: http.StatusBadRequest,
				Err:        fmt.Errorf("login must me not empty"),
			}
			break
		}
		param.Login = formValueLogin

		response, err := api.Profile(r.Context(), param)

		if err != nil {
			handleResult.err = err
			break
		}
		handleResult.response = response
	case "/user/create":
		if r.Method != "POST" {
			handleResult.err = ApiError{
				HTTPStatus: 406,
				Err:        fmt.Errorf("bad method"),
			}
			break
		}
		if r.Header.Get("X-Auth") != "100500" {
					handleResult.err = ApiError{
						HTTPStatus: 403,
						Err:        fmt.Errorf("unauthorized"),
					}
					break
				}
		var param CreateParams
		formValueLogin := r.FormValue("login")
		if formValueLogin == "" {
			handleResult.err = ApiError{
				HTTPStatus: http.StatusBadRequest,
				Err:        fmt.Errorf("login must me not empty"),
			}
			break
		}
		if len(formValueLogin) < 10 {
			handleResult.err = ApiError{
				HTTPStatus: 400,
				Err:        fmt.Errorf("login len must be >= 10"),
			}
			break
		}
		param.Login = formValueLogin

		formValueName := r.FormValue("full_name")
		param.Name = formValueName

		formValueStatus := r.FormValue("status")
		if formValueStatus == "" {
			formValueStatus = "user"
		}
		statusValidEnumValues := map[string]bool{
			"user": true,
			"moderator": true,
			"admin": true,
}
		if !statusValidEnumValues[formValueStatus] {
			handleResult.err = ApiError{
				HTTPStatus: 400,
				Err:        fmt.Errorf("status must be one of [user, moderator, admin]"),
			}
			break
		}
		param.Status = formValueStatus

		formValueAge := r.FormValue("age")
		parsedAge, err := strconv.Atoi(formValueAge)
		if err != nil {
			handleResult.err = ApiError{
				HTTPStatus: 400,
				Err:        fmt.Errorf("age must be int"),
			}
			break
		}
		if parsedAge < 0 {
			handleResult.err = ApiError{
				HTTPStatus: 400,
				Err:        fmt.Errorf("age must be >= 0"),
			}
			break
		}
		if parsedAge > 128 {
			handleResult.err = ApiError{
				HTTPStatus: 400,
				Err:        fmt.Errorf("age must be <= 128"),
			}
			break
		}
		param.Age = parsedAge

		response, err := api.Create(r.Context(), param)

		if err != nil {
			handleResult.err = err
			break
		}
		handleResult.response = response

	default:
		handleResult.err = ApiError{
			HTTPStatus: 404,
			Err:        fmt.Errorf("unknown method"),
		}
	}

	writeHeader(w, handleResult)
	writeBody(w, handleResult)
}

func (api *OtherApi) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var handleResult handleResult
	switch r.URL.Path {
	case "/user/create":
		if r.Method != "POST" {
			handleResult.err = ApiError{
				HTTPStatus: 406,
				Err:        fmt.Errorf("bad method"),
			}
			break
		}
		if r.Header.Get("X-Auth") != "100500" {
					handleResult.err = ApiError{
						HTTPStatus: 403,
						Err:        fmt.Errorf("unauthorized"),
					}
					break
				}
		var param OtherCreateParams
		formValueUsername := r.FormValue("username")
		if formValueUsername == "" {
			handleResult.err = ApiError{
				HTTPStatus: http.StatusBadRequest,
				Err:        fmt.Errorf("username must me not empty"),
			}
			break
		}
		if len(formValueUsername) < 3 {
			handleResult.err = ApiError{
				HTTPStatus: 400,
				Err:        fmt.Errorf("username len must be >= 3"),
			}
			break
		}
		param.Username = formValueUsername

		formValueName := r.FormValue("account_name")
		param.Name = formValueName

		formValueClass := r.FormValue("class")
		if formValueClass == "" {
			formValueClass = "warrior"
		}
		classValidEnumValues := map[string]bool{
			"warrior": true,
			"sorcerer": true,
			"rouge": true,
}
		if !classValidEnumValues[formValueClass] {
			handleResult.err = ApiError{
				HTTPStatus: 400,
				Err:        fmt.Errorf("class must be one of [warrior, sorcerer, rouge]"),
			}
			break
		}
		param.Class = formValueClass

		formValueLevel := r.FormValue("level")
		parsedLevel, err := strconv.Atoi(formValueLevel)
		if err != nil {
			handleResult.err = ApiError{
				HTTPStatus: 400,
				Err:        fmt.Errorf("level must be int"),
			}
			break
		}
		if parsedLevel < 1 {
			handleResult.err = ApiError{
				HTTPStatus: 400,
				Err:        fmt.Errorf("level must be >= 1"),
			}
			break
		}
		if parsedLevel > 50 {
			handleResult.err = ApiError{
				HTTPStatus: 400,
				Err:        fmt.Errorf("level must be <= 50"),
			}
			break
		}
		param.Level = parsedLevel

		response, err := api.Create(r.Context(), param)

		if err != nil {
			handleResult.err = err
			break
		}
		handleResult.response = response

	default:
		handleResult.err = ApiError{
			HTTPStatus: 404,
			Err:        fmt.Errorf("unknown method"),
		}
	}

	writeHeader(w, handleResult)
	writeBody(w, handleResult)
}
