package main

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"log"
	"os"
	"reflect"
	"strconv"
	"strings"
)

const topDeclarations = `
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
`

func main() {
	inputPath := os.Args[1]
	outputPath := os.Args[2]

	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, inputPath, nil, parser.ParseComments)
	if err != nil {
		log.Fatal(err)
	}

	funcsToServe := selectFuncsToServe(*node)
	groupedFuncsByReceiver := groupFuncsByReceiver(funcsToServe)

	structs := selectStructs(*node)

	out, err := os.Create(outputPath)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Fprintln(out, `package `+node.Name.Name)
	fmt.Fprintln(out)
	fmt.Fprintln(out, topDeclarations)

	generateServeHTTP(out, groupedFuncsByReceiver, structs)
}

func selectFuncsToServe(node ast.File) []ast.FuncDecl {
	var result []ast.FuncDecl

	for _, decl := range node.Decls {
		funcDecl, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}

		if funcDecl.Doc == nil {
			continue
		}

		hasComment, _ := getFuncComment(*funcDecl)
		if hasComment {
			result = append(result, *funcDecl)
		}
	}

	return result
}

func getFuncComment(funcDecl ast.FuncDecl) (hasComment bool, comment string) {
	const apiGenPrefix = "// apigen:api "

	for _, comment := range funcDecl.Doc.List {
		if strings.HasPrefix(comment.Text, apiGenPrefix) {
			return true, comment.Text[len(apiGenPrefix):]
		}
	}
	return false, ""
}

func groupFuncsByReceiver(funcs []ast.FuncDecl) map[string][]ast.FuncDecl {
	result := make(map[string][]ast.FuncDecl)

	for _, f := range funcs {
		receiverName := f.Recv.List[0].Type.(*ast.StarExpr).X.(*ast.Ident).Name
		funcsByReciver := result[receiverName]
		result[receiverName] = append(funcsByReciver, f)
	}

	return result
}

func selectStructs(node ast.File) map[string]ast.TypeSpec {
	var result = make(map[string]ast.TypeSpec)

	for _, decl := range node.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}

		for _, spec := range genDecl.Specs {
			currType, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}

			_, ok = currType.Type.(*ast.StructType)
			if !ok {
				continue
			}

			result[currType.Name.Name] = *currType
		}
	}

	return result
}

func generateServeHTTP(out *os.File, groupedFuncsByReceiver map[string][]ast.FuncDecl, structs map[string]ast.TypeSpec) {
	for receiver, funcDecls := range groupedFuncsByReceiver {
		fmt.Fprintf(out, `
func (api *%s) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var handleResult handleResult
	switch r.URL.Path {
`, receiver)

		for _, funcDecl := range funcDecls {
			_, comment := getFuncComment(funcDecl)

			var funcTag FuncTag
			_ = json.Unmarshal([]byte(comment), &funcTag)

			fmt.Fprintf(out, `	case "%s":
`, funcTag.Url)

			if funcTag.Method != "" {
				fmt.Fprintf(out, `		if r.Method != "%s" {
			handleResult.err = ApiError{
				HTTPStatus: 406,
				Err:        fmt.Errorf("bad method"),
			}
			break
		}
`, funcTag.Method)
			}

			if funcTag.Auth {
				fmt.Fprintf(out, `		if r.Header.Get("X-Auth") != "100500" {
					handleResult.err = ApiError{
						HTTPStatus: 403,
						Err:        fmt.Errorf("unauthorized"),
					}
					break
				}
`)
			}

			paramStructName := funcDecl.Type.Params.List[1].Type.(*ast.Ident).Name
			s := structs[paramStructName]

			fmt.Fprintf(out, "		var param %s\n", paramStructName)

			generateParamInit(out, s)

			fmt.Fprintf(out, "		response, err := api.%s(r.Context(), param)\n", funcDecl.Name.Name)
			fmt.Fprintf(out, `
		if err != nil {
			handleResult.err = err
			break
		}
`)

			fmt.Fprintln(out, `		handleResult.response = response`)
		}

		fmt.Fprintln(out, `
	default:
		handleResult.err = ApiError{
			HTTPStatus: 404,
			Err:        fmt.Errorf("unknown method"),
		}
	}

	writeHeader(w, handleResult)
	writeBody(w, handleResult)
}`)
	}
}

type FuncTag struct {
	Url    string `"json:"url"`
	Auth   bool   `"json:"auth"`
	Method string `"json:"method"`
}

func generateParamInit(out *os.File, s ast.TypeSpec) {
	for _, field := range s.Type.(*ast.StructType).Fields.List {
		fieldName := field.Names[0].Name

		formValueVarName := "formValue" + fieldName
		parsedVarName := "parsed" + fieldName
		assignToVar := formValueVarName

		fieldTag := parseTag(*field)

		var paramName string
		if fieldTag.paramname == "" {
			paramName = strings.ToLower(fieldName)
		} else {
			paramName = fieldTag.paramname
		}

		fmt.Fprintf(out, `		%s := r.FormValue("%s")
`, formValueVarName, paramName)

		if fieldTag.dflt != "" {
			fmt.Fprintf(out, `		if %s == "" {
			%s = "%s"
		}
`, formValueVarName, formValueVarName, fieldTag.dflt)
		}

		if fieldTag.required {
			fmt.Fprintf(out, `		if %s == "" {
			handleResult.err = ApiError{
				HTTPStatus: http.StatusBadRequest,
				Err:        fmt.Errorf("%s must me not empty"),
			}
			break
		}
`, formValueVarName, paramName)
		}

		filedIsInt := field.Type.(*ast.Ident).Name == "int"

		if filedIsInt {
			fmt.Fprintf(out, `		%s, err := strconv.Atoi(%s)
		if err != nil {
			handleResult.err = ApiError{
				HTTPStatus: 400,
				Err:        fmt.Errorf("%s must be int"),
			}
			break
		}
`, parsedVarName, formValueVarName, paramName)
			assignToVar = parsedVarName
		}

		if fieldTag.hasMin {
			if filedIsInt {
				fmt.Fprintf(out, `		if %s < %d {
			handleResult.err = ApiError{
				HTTPStatus: 400,
				Err:        fmt.Errorf("%s must be >= %d"),
			}
			break
		}
`, parsedVarName, fieldTag.min, paramName, fieldTag.min)
			} else {
				fmt.Fprintf(out, `		if len(%s) < %d {
			handleResult.err = ApiError{
				HTTPStatus: 400,
				Err:        fmt.Errorf("%s len must be >= %d"),
			}
			break
		}
`, formValueVarName, fieldTag.min, paramName, fieldTag.min)
			}
		}

		if fieldTag.hasMax {
			if filedIsInt {
				fmt.Fprintf(out, `		if %s > %d {
			handleResult.err = ApiError{
				HTTPStatus: 400,
				Err:        fmt.Errorf("%s must be <= %d"),
			}
			break
		}
`, parsedVarName, fieldTag.max, paramName, fieldTag.max)
			} else {
				fmt.Fprintf(out, `		if len(%s) > %d {
			handleResult.err = ApiError{
				HTTPStatus: 400,
				Err:        fmt.Errorf("%s len must be <= %d"),
			}
			break
		}
`, formValueVarName, fieldTag.max, paramName, fieldTag.max)
			}
		}

		validEnumValuesVarName := paramName + "ValidEnumValues"

		if len(fieldTag.enum) > 0 {
			fmt.Fprintf(out, "		%s := map[string]bool{", validEnumValuesVarName)
			fmt.Fprintln(out)
			for _, enumValue := range fieldTag.enum {
				fmt.Fprintf(out, `			"%s": true,
`, enumValue)
			}
			fmt.Fprintln(out, "}")

			fmt.Fprintf(out, `		if !%s[%s] {
			handleResult.err = ApiError{
				HTTPStatus: 400,
				Err:        fmt.Errorf("%s must be one of [%s]"),
			}
			break
		}
`, validEnumValuesVarName, formValueVarName, paramName, strings.Join(fieldTag.enum, ", "))
		}

		fmt.Fprintf(out, "		param.%s = %s\n\n", fieldName, assignToVar)
	}

}

func parseTag(field ast.Field) tagParseResult {
	fieldTag := reflect.StructTag(field.Tag.Value[1 : len(field.Tag.Value)-1])
	apivalidator := fieldTag.Get("apivalidator")

	result := tagParseResult{}

	for _, item := range strings.Split(apivalidator, ",") {
		if item == "required" {
			result.required = true
			continue
		}

		keyValue := strings.Split(item, "=")
		key := keyValue[0]
		value := keyValue[1]

		switch key {
		case "paramname":
			result.paramname = value
		case "min":
			result.min, _ = strconv.Atoi(value)
			result.hasMin = true
		case "max":
			result.max, _ = strconv.Atoi(value)
			result.hasMax = true
		case "enum":
			result.enum = strings.Split(value, "|")
		case "default":
			result.dflt = value
		}
	}
	return result
}

type tagParseResult struct {
	required  bool
	min       int
	hasMin    bool
	max       int
	hasMax    bool
	paramname string
	enum      []string
	dflt      string
}
