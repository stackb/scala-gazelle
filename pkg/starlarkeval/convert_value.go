/*
Copyright 2020 Google LLC

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    https://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// This file contains functions to convert from one AST to the other.
// Input: AST from go.starlark.net/syntax
// Output: AST from github.com/bazelbuild/buildtools/build

package starlarkeval

import (
	"strconv"
	"strings"

	"github.com/bazelbuild/buildtools/build"
	"go.starlark.net/starlark"
	// "go.starlark.net/syntax"
)

func ConvValue(value starlark.Value) build.Expr {
	switch t := value.(type) {
	case *starlark.Int:
		if val, ok := t.Int64(); ok {
			return &build.LiteralExpr{
				Token: strconv.FormatInt(val, 10),
			}
		}
	case *starlark.String:
		return &build.StringExpr{
			Value:       t.GoString(),
			TripleQuote: strings.HasPrefix(t.String(), "\"\"\""),
		}
	}
	return nil
}
