package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/akurin/golang-webservices/hw6_db_explorer/dao"
	"github.com/akurin/golang-webservices/hw6_db_explorer/schema"
)

func NewDbExplorer(db *sql.DB) (http.Handler, error) {
	dbSchema, err := schema.Of(db)
	if err != nil {
		panic(err)
	}
	dao := dao.CreateDao(db, dbSchema)
	dbExplorer := dbExplorer{dao, dbSchema}
	router := router{dbExplorer}
	return router, nil
}

type router struct {
	explorer dbExplorer
}

func (router router) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	pathParts := getPathParts(r.URL)

	if len(pathParts) == 0 {
		router.explorer.listTables(w, r)
		return
	}

	if len(pathParts) == 1 {
		if r.Method == "GET" {
			offset, _ := strconv.Atoi(r.FormValue("offset"))
			limit, _ := strconv.Atoi(r.FormValue("limit"))
			router.explorer.getTable(w, r, pathParts[0], offset, limit)
			return
		}

		if r.Method == "PUT" {
			router.explorer.create(w, r, pathParts[0])
			return
		}
	}

	if len(pathParts) == 2 {
		if r.Method == "GET" {
			router.explorer.getRow(w, r, pathParts[0], pathParts[1])
			return
		}

		if r.Method == "POST" {
			router.explorer.update(w, r, pathParts[0], pathParts[1])
			return
		}

		if r.Method == "DELETE" {
			router.explorer.delete(w, r, pathParts[0], pathParts[1])
			return
		}
	}

	http.NotFound(w, r)
}

func getPathParts(url *url.URL) []string {
	var result []string
	split := strings.Split(url.Path, "/")
	for _, s := range split {
		if s != "" {
			result = append(result, s)
		}
	}
	return result
}

type dbExplorer struct {
	dao    dao.Dao
	schema schema.DbSchema
}

type Obj map[string]interface{}

func (explorer dbExplorer) listTables(w http.ResponseWriter, r *http.Request) {
	tables, err := explorer.dao.Tables()
	if err != nil {
		respondInternalServerError(w, err)
		return
	}

	response := Obj{
		"response": Obj{
			"tables": tables.Names(),
		},
	}

	responseBytes, _ := json.Marshal(response)
	w.Write([]byte(responseBytes))
}

func respondInternalServerError(w http.ResponseWriter, err error) {
	fmt.Println(err)
	w.WriteHeader(http.StatusInternalServerError)
}

func (explorer dbExplorer) getTable(
	w http.ResponseWriter, r *http.Request, table string, offset int, limit int) {

	tableSchema, ok := explorer.schema.TableSchemaOf(table)
	if !ok {
		respondTableNotFound(w)
		return
	}

	rows, err := explorer.dao.SelectAll(table, offset, limit)
	if err != nil {
		respondInternalServerError(w, err)
		return
	}

	records, err := explorer.toJSON(rows, tableSchema)

	result := Obj{
		"response": Obj{
			"records": records,
		},
	}

	bytes, _ := json.Marshal(result)
	w.Write(bytes)
}

func respondTableNotFound(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNotFound)
	w.Write([]byte(`{"error": "unknown table"}`))
}

func (explorer dbExplorer) toJSON(rows [][]interface{}, tableSchema schema.TableSchema) ([]Obj, error) {
	var records []Obj
	for _, row := range rows {
		record := Obj{}
		for columnIndex, fieldSchema := range tableSchema {
			sqlValue := row[columnIndex]
			var jsonValue interface{}

			switch sqlValue.(type) {
			case *sql.NullInt64:
				sqlValue := sqlValue.(*sql.NullInt64)
				if sqlValue.Valid {
					jsonValue = sqlValue.Int64
				}
			case *sql.NullBool:
				sqlValue := sqlValue.(*sql.NullBool)
				if sqlValue.Valid {
					jsonValue = sqlValue.Bool
				}
			case *sql.NullFloat64:
				sqlValue := sqlValue.(*sql.NullFloat64)
				if sqlValue.Valid {
					jsonValue = sqlValue.Float64
				}
			case *sql.NullString:
				sqlValue := sqlValue.(*sql.NullString)
				if sqlValue.Valid {
					jsonValue = sqlValue.String
				}
			default:
				return nil, fmt.Errorf("unexpected type %T", sqlValue)
			}

			record[fieldSchema.Name] = jsonValue
		}
		records = append(records, record)
	}
	return records, nil
}

func (explorer dbExplorer) getRow(
	w http.ResponseWriter, r *http.Request, table string, id string) {

	tableSchema, ok := explorer.schema.TableSchemaOf(table)
	if !ok {
		respondTableNotFound(w)
		return
	}

	row, err := explorer.dao.SelectRow(table, id)
	if err != nil {
		respondInternalServerError(w, err)
		return
	}

	if row == nil {
		w.WriteHeader(404)
		w.Write([]byte(`{"error": "record not found"}`))
		return
	}

	records, err := explorer.toJSON([][]interface{}{row}, tableSchema)

	result := Obj{
		"response": Obj{
			"record": records[0],
		},
	}

	bytes, _ := json.Marshal(result)
	w.Write(bytes)
}

func (explorer dbExplorer) create(w http.ResponseWriter, r *http.Request, table string) {
	_, ok := explorer.schema.TableSchemaOf(table)
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	bodyBytes, err := ioutil.ReadAll(r.Body)
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	bodyJSON := make(map[string]interface{})
	err = json.Unmarshal(bodyBytes, &bodyJSON)
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	primaryKey, id, err := explorer.dao.Insert(table, bodyJSON)
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	result := Obj{
		"response": Obj{
			primaryKey: id,
		},
	}

	bytes, _ := json.Marshal(result)
	w.Write(bytes)
}

func (explorer dbExplorer) update(w http.ResponseWriter, r *http.Request, table string, id string) {
	_, ok := explorer.schema.TableSchemaOf(table)
	if !ok {
		respondTableNotFound(w)
		return
	}

	bodyBytes, err := ioutil.ReadAll(r.Body)
	if err != nil {
		respondInternalServerError(w, err)
		return
	}

	bodyJSON := make(map[string]interface{})
	err = json.Unmarshal(bodyBytes, &bodyJSON)
	if err != nil {
		respondInternalServerError(w, err)
		return
	}

	rowsAffected, err := explorer.dao.UpdateRow(table, id, bodyJSON)
	if err != nil {
		switch err.(type) {
		case dao.BadValues:
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(fmt.Sprintf(`{"error": "%s"}`, err.Error())))
			return
		}

		fmt.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	result := Obj{
		"response": Obj{
			"updated": rowsAffected,
		},
	}

	bytes, _ := json.Marshal(result)
	w.Write(bytes)
}

func (explorer dbExplorer) delete(w http.ResponseWriter, r *http.Request, table string, id string) {
	_, ok := explorer.schema.TableSchemaOf(table)
	if !ok {
		respondTableNotFound(w)
		return
	}

	affected, err := explorer.dao.Delete(table, id)
	if err != nil {
		respondInternalServerError(w, err)
		return
	}

	result := Obj{
		"response": Obj{
			"deleted": affected,
		},
	}

	bytes, _ := json.Marshal(result)
	w.Write(bytes)
}
