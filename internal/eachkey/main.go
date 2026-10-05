// Command eachkey generates the two callback variants from a shared scanner.
package main

import (
	"bytes"
	"go/format"
	"io/ioutil"
	"log"
	"text/template"
)

type callbackVariant struct {
	Name          string
	ResultType    string
	MissingResult string
	MatchedResult string
	ErrorCallback bool
	ArrayCall     string
}

func main() {
	source, err := ioutil.ReadFile("internal/eachkey/eachkey.go.tmpl")
	if err != nil {
		log.Fatal(err)
	}
	tmpl, err := template.New("eachkey").Parse(string(source))
	if err != nil {
		log.Fatal(err)
	}
	variants := []callbackVariant{
		{Name: "EachKey", ResultType: "int", MissingResult: "-1", MatchedResult: "i", ArrayCall: "arrOff, _ := ArrayEach"},
		{Name: "EachKeyErr", ResultType: "error", MissingResult: "nil", MatchedResult: "nil", ErrorCallback: true, ArrayCall: "arrOff, _, _ := arrayEachErr"},
	}
	var output bytes.Buffer
	if err := tmpl.Execute(&output, variants); err != nil {
		log.Fatal(err)
	}
	formatted, err := format.Source(output.Bytes())
	if err != nil {
		log.Fatal(err)
	}
	if err := ioutil.WriteFile("eachkey.go", formatted, 0644); err != nil {
		log.Fatal(err)
	}
}
