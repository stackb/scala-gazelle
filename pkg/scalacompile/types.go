package scalacompile

import "encoding/xml"

// ScalaCompileSpec describes the symbols derived from attempting to compile a scala source file.
type ScalaCompileSpec struct {
	// NotFound is a list of types that were not found (e.g. "not found: value DateUtils")
	NotFound []*NotFoundSymbol `json:"notFound,omitempty"`
	// E.g. "object Session is not a member of package com.foo.core"
	NotMember []*NotMemberSymbol `json:"notMember,omitempty"`
}

type NotFoundSymbol struct {
	Kind string `json:"kind"`
	Name string `json:"name"`
}

type NotMemberSymbol struct {
	Kind    string `json:"kind"`
	Name    string `json:"name"`
	Package string `json:"package"`
}

type CompileRequest struct {
	XMLName xml.Name `xml:"compileRequest"`
	Files   []string `xml:"file"`
}

type CompileResponse struct {
	XMLName     xml.Name     `xml:"compileResponse"`
	Diagnostics []Diagnostic `xml:"diagnostic"`
}

type Diagnostic struct {
	XMLName  xml.Name `xml:"diagnostic"`
	Source   string   `xml:"source,attr"`
	Line     int      `xml:"line,attr"`
	Severity string   `xml:"sev,attr"`
	Message  string   `xml:",chardata"`
}
