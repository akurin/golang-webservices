package main

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"
)

func Test_Client_Finds_Users(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(xmlDatasetSearchServer))
	defer server.Close()

	tests := []struct {
		name         string
		req          SearchRequest
		wantLen      int
		wantNextPage bool
	}{
		{
			name: "limit coerced to 25",
			req: SearchRequest{
				Limit: 100,
			},
			wantLen:      25,
			wantNextPage: true,
		},
		{
			name: "has next page",
			req: SearchRequest{
				Limit: 10,
			},
			wantLen:      10,
			wantNextPage: true,
		},
		{
			name: "does not have next page",
			req: SearchRequest{
				Limit:  1,
				Offset: 34,
			},
			wantLen:      1,
			wantNextPage: false,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			client := &SearchClient{
				URL: server.URL,
			}
			got, err := client.FindUsers(test.req)
			if err != nil {
				t.Errorf("Unexpected error: %s", err.Error())
			}
			if len(got.Users) != test.wantLen {
				t.Errorf("len = %v, want %v", len(got.Users), test.wantLen)
			}
			if got.NextPage != test.wantNextPage {
				t.Errorf("NextPage = %v, want %v", got.NextPage, test.wantNextPage)
			}
		})
	}
}

func xmlDatasetSearchServer(w http.ResponseWriter, r *http.Request) {
	usersBytes, err := searchInXMLDataSet(r)
	if err != nil {
		w.Write([]byte(err.Error()))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(usersBytes)
}

func searchInXMLDataSet(r *http.Request) ([]byte, error) {
	dataset, err := readDataSet()
	if err != nil {
		return nil, err
	}
	request, err := readSearchRequest(*r.URL)
	if err != nil {
		return nil, err
	}

	users, err := applyRequest(dataset, request)
	if err != nil {
		return nil, err
	}

	usersBytes, err := json.Marshal(users)
	if err != nil {
		return nil, err
	}

	return usersBytes, nil
}

func readDataSet() ([]User, error) {
	datasetBytes, err := ioutil.ReadFile("dataset.xml")
	if err != nil {
		return nil, err
	}

	var dataset Dataset
	if err = xml.Unmarshal(datasetBytes, &dataset); err != nil {
		return nil, err
	}

	result := make([]User, 0, len(dataset.Rows))
	for _, row := range dataset.Rows {
		user := User{
			Id:     row.Id,
			Name:   row.FirstName + " " + row.LastName,
			Age:    row.Age,
			About:  row.About,
			Gender: row.Gender,
		}
		result = append(result, user)
	}

	return result, nil
}

type Dataset struct {
	XMLName xml.Name `xml:"root"`
	Rows    []Row    `xml:"row"`
}

type Row struct {
	XMLName   xml.Name `xml:"row"̀`
	Id        int      `xml:"id"`
	Age       int      `xml:"age"`
	FirstName string   `xml:"first_name"`
	LastName  string   `xml:"last_name"`
	Gender    string   `xml:"gender"`
	About     string   `xml:"about"`
}

func readSearchRequest(u url.URL) (SearchRequest, error) {
	query := u.Query()

	limitString := query.Get("limit")
	limit, err := strconv.Atoi(limitString)
	if err != nil {
		return SearchRequest{}, err
	}

	offsetString := query.Get("offset")
	offset, err := strconv.Atoi(offsetString)
	if err != nil {
		return SearchRequest{}, err
	}

	q := query.Get("query")

	orderField := query.Get("order_field")

	orderByString := query.Get("order_by")
	orderBy, err := strconv.Atoi(orderByString)
	if err != nil {
		return SearchRequest{}, err
	}

	return SearchRequest{
		Limit:      limit,
		Offset:     offset,
		Query:      q,
		OrderField: orderField,
		OrderBy:    orderBy,
	}, nil
}

func applyRequest(users []User, request SearchRequest) ([]User, error) {
	sortFunc, err := sortFunc(users, request.OrderField)
	if err != nil {
		return nil, err
	}
	var result []User
	sort.Slice(users, sortFunc)

	usersMeetFilterCount := 0
	for _, user := range users {
		if len(result) == request.Limit {
			break
		}

		userMeetsFilter := strings.Contains(user.Name, request.Query) ||
			strings.Contains(user.About, request.Query)

		if userMeetsFilter {
			if usersMeetFilterCount >= request.Offset {
				result = append(result, user)
			}

			usersMeetFilterCount++
		}
	}

	return result, nil
}

func sortFunc(users []User, orderField string) (func(i, j int) bool, error) {
	switch orderField {
	case "Id":
		return func(i, j int) bool {
			return users[i].Id < users[j].Id
		}, nil
	case "Age":
		return func(i, j int) bool {
			return users[i].Age < users[j].Age
		}, nil
	case "Name":
		fallthrough
	case "":
		return func(i, j int) bool {
			return users[i].Name < users[j].Name
		}, nil
	default:
		return nil, fmt.Errorf("Unknown field '%s'", orderField)
	}
}

func Test_Client_Validates_Arguments(t *testing.T) {
	tests := []struct {
		name string
		req  SearchRequest
	}{
		{
			name: "limit must be >= 0",
			req: SearchRequest{
				Limit: -1,
			},
		},
		{
			name: "offset must be >= 0",
			req: SearchRequest{
				Offset: -1,
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			client := &SearchClient{
				URL: "http://some-url",
			}
			_, err := client.FindUsers(test.req)
			if err == nil {
				t.Errorf("Expected error")
			}
		})
	}
}

func SlowSearchServer(w http.ResponseWriter, r *http.Request) {
	time.Sleep(time.Second + 2)
}

func Test_Returns_Error_When_Slow_Server(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(SlowSearchServer))
	defer server.Close()

	srv := &SearchClient{
		URL: server.URL,
	}
	_, err := srv.FindUsers(SearchRequest{})

	if err == nil {
		t.Errorf("should be timeout when slow server")
	}
}

func Test_Returns_Error_When_Server_Returns_Not_OK_Status(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		body       string
	}{
		{
			name:       "unauthorized",
			statusCode: http.StatusUnauthorized,
		},
		{
			name:       "internal server error",
			statusCode: http.StatusInternalServerError,
		},
		{
			name:       "bad request 1",
			statusCode: http.StatusBadRequest,
			body:       "{invalid json}",
		},
		{
			name:       "bad request 2",
			statusCode: http.StatusBadRequest,
			body:       `{"Error": "ErrorBadOrderField"}`,
		},
		{
			name:       "bad request 3",
			statusCode: http.StatusBadRequest,
			body:       `{"Error": "SomeOtherError"}`,
		},
		{
			name:       "internal server error",
			statusCode: http.StatusInternalServerError,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(
				func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(test.statusCode)
					w.Write([]byte(test.body))
				}))

			defer server.Close()

			client := &SearchClient{
				URL: server.URL,
			}
			_, err := client.FindUsers(SearchRequest{})

			if err == nil {
				t.Errorf("Expected error")
			}
		})
	}
}

func Test_Returns_Error_When_Server_Returns_Invalid_Json(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("{invalid json}"))
		}))

	defer server.Close()

	client := &SearchClient{
		URL: server.URL,
	}
	_, err := client.FindUsers(SearchRequest{})

	if err == nil {
		t.Errorf("Expected error")
	}

	if !strings.HasPrefix(err.Error(), "cant unpack result json") {
		t.Errorf("Expected unpack error, got: '%s'", err.Error())
	}
}

func Test_Returns_Error_When_Invalid_Url(t *testing.T) {
	client := &SearchClient{
		URL: "invalid-url",
	}
	_, err := client.FindUsers(SearchRequest{})

	if err == nil {
		t.Errorf("Expected error")
	}
}
