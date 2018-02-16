package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"strings"
)

//easyjson:json
type Person struct {
	Browsers []string `json:"browsers"`
	Email    string   `json:"email"`
	Name     string   `json:"name"`
}

// вам надо написать более быструю оптимальную этой функции
func FastSearch(out io.Writer) {
	file, err := os.Open(filePath)
	if err != nil {
		panic(err)
	}

	scanner := bufio.NewScanner(file)

	seenBrowsers := make(map[string]bool, 114)
	// foundUsers := ""

	var foundUsersBuffer bytes.Buffer

	var lineIndex = -1

	for scanner.Scan() {
		lineIndex++

		var person Person
		if err := person.UnmarshalJSON(scanner.Bytes()); err != nil {
			log.Fatal(err)
		}

		isAndroid := false
		isMSIE := false

		browsers := person.Browsers

		for _, browser := range browsers {
			if strings.Contains(browser, "Android") {
				isAndroid = true
				seenBrowsers[browser] = true
			}

			if strings.Contains(browser, "MSIE") {
				isMSIE = true
				seenBrowsers[browser] = true
			}
		}

		if !(isAndroid && isMSIE) {
			continue
		}

		foundUsersBuffer.WriteString("[")
		foundUsersBuffer.WriteString(strconv.Itoa(lineIndex))
		foundUsersBuffer.WriteString("] ")
		foundUsersBuffer.WriteString(person.Name)
		foundUsersBuffer.WriteString(" <")

		email := strings.Replace(person.Email, "@", " [at] ", 1)
		foundUsersBuffer.WriteString(email)

		foundUsersBuffer.WriteString(">\n")
	}

	fmt.Fprintln(out, "found users:\n"+foundUsersBuffer.String())
	fmt.Fprintln(out, "Total unique browsers", len(seenBrowsers))
}
