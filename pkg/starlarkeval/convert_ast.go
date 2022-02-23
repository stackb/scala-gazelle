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
	"fmt"
	"strconv"
	"strings"

	"github.com/bazelbuild/buildtools/build"
	"go.starlark.net/syntax"
)

func ConvFile(f *syntax.File) *build.File {
	stmts := []build.Expr{}
	for _, stmt := range f.Stmts {
		stmts = append(stmts, ConvStmt(stmt))
	}

	return &build.File{
		Type:     build.TypeDefault,
		Stmt:     stmts,
		Comments: ConvComments(f.Comments()),
	}
}

func ConvStmt(stmt syntax.Stmt) build.Expr {
	switch stmt := stmt.(type) {
	case *syntax.ExprStmt:
		s := ConvExpr(stmt.X)
		*s.Comment() = ConvComments(stmt.Comments())
		return s
	case *syntax.BranchStmt:
		return &build.BranchStmt{
			Token:    stmt.Token.String(),
			Comments: ConvComments(stmt.Comments()),
		}
	case *syntax.LoadStmt:
		load := &build.LoadStmt{
			Module:       ConvExpr(stmt.Module).(*build.StringExpr),
			ForceCompact: singleLine(stmt),
		}
		for _, ident := range stmt.From {
			load.From = append(load.From, ConvExpr(ident).(*build.Ident))
		}
		for _, ident := range stmt.To {
			load.To = append(load.To, ConvExpr(ident).(*build.Ident))
		}
		return load
	case *syntax.AssignStmt:
		return &build.AssignExpr{
			Op:       stmt.Op.String(),
			LHS:      ConvExpr(stmt.LHS),
			RHS:      ConvExpr(stmt.RHS),
			Comments: ConvComments(stmt.Comments()),
		}
	case *syntax.IfStmt:
		return &build.IfStmt{
			Cond:     ConvExpr(stmt.Cond),
			True:     ConvStmts(stmt.True),
			False:    ConvStmts(stmt.False),
			Comments: ConvComments(stmt.Comments()),
		}
	case *syntax.DefStmt:
		return &build.DefStmt{
			Name:     stmt.Name.Name,
			Comments: ConvComments(stmt.Comments()),
			Function: build.Function{
				Params: ConvExprs(stmt.Params),
				Body:   ConvStmts(stmt.Body),
			},
		}
	case *syntax.ForStmt:
		return &build.ForStmt{
			Vars:     ConvExpr(stmt.Vars),
			X:        ConvExpr(stmt.X),
			Comments: ConvComments(stmt.Comments()),
			Body:     ConvStmts(stmt.Body),
		}
	case *syntax.ReturnStmt:
		return &build.ReturnStmt{
			Comments: ConvComments(stmt.Comments()),
			Result:   ConvExpr(stmt.Result),
		}
	}
	panic("unreachable")
}

func ConvStmts(list []syntax.Stmt) []build.Expr {
	res := []build.Expr{}
	for _, i := range list {
		res = append(res, ConvStmt(i))
	}
	return res
}

func ConvExprs(list []syntax.Expr) []build.Expr {
	res := []build.Expr{}
	for _, i := range list {
		res = append(res, ConvExpr(i))
	}
	return res
}

func ConvCommentList(list []syntax.Comment, txt string) []build.Comment {
	res := []build.Comment{}
	for _, c := range list {
		res = append(res, build.Comment{Token: c.Text})
	}
	return res
}

func ConvComments(c *syntax.Comments) build.Comments {
	if c == nil {
		return build.Comments{}
	}
	return build.Comments{
		Before: ConvCommentList(c.Before, "before"),
		Suffix: ConvCommentList(c.Suffix, "suffix"),
		After:  ConvCommentList(c.After, "after"),
	}
}

// singleLine returns true if the node fits on a single line.
func singleLine(n syntax.Node) bool {
	start, end := n.Span()
	return start.Line == end.Line
}

func convClauses(list []syntax.Node) []build.Expr {
	res := []build.Expr{}
	for _, c := range list {
		switch stmt := c.(type) {
		case *syntax.ForClause:
			res = append(res, &build.ForClause{
				Vars: ConvExpr(stmt.Vars),
				X:    ConvExpr(stmt.X),
			})
		case *syntax.IfClause:
			res = append(res, &build.IfClause{
				Cond: ConvExpr(stmt.Cond),
			})
		}
	}
	return res
}

func ConvExpr(e syntax.Expr) build.Expr {
	if e == nil {
		return nil
	}
	switch e := e.(type) {
	case *syntax.Literal:
		switch e.Token {
		case syntax.INT:
			return &build.LiteralExpr{
				Token:    strconv.FormatInt(e.Value.(int64), 10),
				Comments: ConvComments(e.Comments()),
			}
		case syntax.FLOAT:
			return &build.LiteralExpr{
				Token:    e.Raw,
				Comments: ConvComments(e.Comments()),
			}
		case syntax.STRING:
			return &build.StringExpr{
				Value:       e.Value.(string),
				TripleQuote: strings.HasPrefix(e.Raw, "\"\"\""),
				Comments:    ConvComments(e.Comments()),
			}
		}
	case *syntax.Ident:
		return &build.Ident{Name: e.Name, Comments: ConvComments(e.Comments())}
	case *syntax.BinaryExpr:
		_, lhsEnd := e.X.Span()
		rhsBegin, _ := e.Y.Span()
		if e.Op.String() == "=" {
			return &build.AssignExpr{
				LHS:       ConvExpr(e.X),
				RHS:       ConvExpr(e.Y),
				Op:        e.Op.String(),
				LineBreak: lhsEnd.Line != rhsBegin.Line,
				Comments:  ConvComments(e.Comments()),
			}
		}
		return &build.BinaryExpr{
			X:         ConvExpr(e.X),
			Y:         ConvExpr(e.Y),
			Op:        e.Op.String(),
			LineBreak: lhsEnd.Line != rhsBegin.Line,
			Comments:  ConvComments(e.Comments()),
		}
	case *syntax.UnaryExpr:
		return &build.UnaryExpr{Op: e.Op.String(), X: ConvExpr(e.X)}
	case *syntax.SliceExpr:
		return &build.SliceExpr{X: ConvExpr(e.X), From: ConvExpr(e.Lo), To: ConvExpr(e.Hi), Step: ConvExpr(e.Step)}
	case *syntax.DotExpr:
		return &build.DotExpr{X: ConvExpr(e.X), Name: e.Name.Name}
	case *syntax.CallExpr:
		args := []build.Expr{}
		for _, a := range e.Args {
			args = append(args, ConvExpr(a))
		}
		return &build.CallExpr{
			X:            ConvExpr(e.Fn),
			List:         args,
			ForceCompact: singleLine(e),
		}
	case *syntax.ListExpr:
		list := []build.Expr{}
		for _, i := range e.List {
			list = append(list, ConvExpr(i))
		}
		return &build.ListExpr{List: list, Comments: ConvComments(e.Comments())}
	case *syntax.DictExpr:
		list := []*build.KeyValueExpr{}
		for i := range e.List {
			entry := e.List[i].(*syntax.DictEntry)
			list = append(list, &build.KeyValueExpr{
				Key:      ConvExpr(entry.Key),
				Value:    ConvExpr(entry.Value),
				Comments: ConvComments(entry.Comments()),
			})
		}
		return &build.DictExpr{List: list, Comments: ConvComments(e.Comments())}
	case *syntax.CondExpr:
		return &build.ConditionalExpr{
			Then:     ConvExpr(e.True),
			Test:     ConvExpr(e.Cond),
			Else:     ConvExpr(e.False),
			Comments: ConvComments(e.Comments()),
		}
	case *syntax.Comprehension:
		return &build.Comprehension{
			Body:     ConvExpr(e.Body),
			Clauses:  convClauses(e.Clauses),
			Comments: ConvComments(e.Comments()),
			Curly:    e.Curly,
		}
	case *syntax.ParenExpr:
		return &build.ParenExpr{
			X:        ConvExpr(e.X),
			Comments: ConvComments(e.Comments()),
		}
	case *syntax.TupleExpr:
		return &build.TupleExpr{
			List:         ConvExprs(e.List),
			NoBrackets:   !e.Lparen.IsValid(),
			Comments:     ConvComments(e.Comments()),
			ForceCompact: singleLine(e),
		}
	case *syntax.IndexExpr:
		return &build.IndexExpr{
			X:        ConvExpr(e.X),
			Y:        ConvExpr(e.Y),
			Comments: ConvComments(e.Comments()),
		}
	case *syntax.LambdaExpr:
		return &build.LambdaExpr{
			Comments: ConvComments(e.Comments()),
			Function: build.Function{
				Params: ConvExprs(e.Params),
				Body:   []build.Expr{ConvExpr(e.Body)},
			},
		}
	default:
		panic(fmt.Sprintf("other expr: %T %+v", e, e))
	}
	panic("unreachable")
}
