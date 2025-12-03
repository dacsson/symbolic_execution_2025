package internal

import (
	"bytes"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"log"
	"strconv"

	"golang.org/x/tools/go/ast/astutil"
)

// TODO: not all implemented
//
//	analyzerWhile, etc.?
func analyzerFor(forStmt ast.ForStmt) (string, int, int, int) {
	var var_name string
	var startValue, endValue int

	if assignStmt, ok := forStmt.Init.(*ast.AssignStmt); ok {
		if len(assignStmt.Lhs) == 1 {
			if ident, ok := assignStmt.Lhs[0].(*ast.Ident); ok {
				var_name = ident.Name
			}
		} else {
			panic("")
		}
		if len(assignStmt.Rhs) == 1 {
			if basicLit, ok := assignStmt.Rhs[0].(*ast.BasicLit); ok {
				if basicLit.Kind == token.INT {
					startValue, _ = strconv.Atoi(basicLit.Value)
				}
			}
		} else {
			panic("")
		}
	} else {
		panic("")
	}

	var doesInclude bool
	if binaryExpr, ok := forStmt.Cond.(*ast.BinaryExpr); ok {
		switch binaryExpr.Op {
		case token.LSS: // "<"
			doesInclude = true
			//iters = endValue - startValue
		case token.LEQ:
			doesInclude = false
			// + 1
		default:
			panic("unsupported relation operator")
		}

		if basicLit, ok := binaryExpr.Y.(*ast.BasicLit); ok && basicLit.Kind == token.INT {
			endValue, _ = strconv.Atoi(basicLit.Value)
		} else {
			panic("")
		}
	} else {
		panic("")
	}

	iters := 0
	step := 0

	if doesInclude {
		iters = endValue - startValue
	} else {
		iters = endValue - startValue + 1
	}

	if incStmt, ok := forStmt.Post.(*ast.IncDecStmt); ok {
		switch incStmt.Tok {
		// ++ && --
		case token.INC:
			step = 1
		case token.DEC:
			step = -1
		default:
			panic("")
		}
	}

	return var_name, startValue, iters, step
}

func genNewBlock(cursor *astutil.Cursor) bool {
	if forStmt, ok := cursor.Node().(*ast.ForStmt); ok {
		var stmts []ast.Stmt
		varName, start, iters, steps := analyzerFor(*forStmt)

		for i := range iters {
			curr := start + i*steps
			for _, stmt := range forStmt.Body.List {
				// TODO: test, is this deref/desctroy ptr
				//		 i dont remember why i typed this comment ???
				stmt_ := astutil.Apply(
					stmt,
					func(cursor *astutil.Cursor) bool {
						if ident, ok := cursor.Node().(*ast.Ident); ok && ident.Name == varName {
							newLit := &ast.BasicLit{Kind: token.INT, Value: strconv.Itoa(curr)}
							cursor.Replace(newLit)
						}
						return true
					},
					nil).(ast.Stmt)
				stmts = append(stmts, stmt_)
			}
		}

		block := &ast.BlockStmt{
			Lbrace: forStmt.Body.Lbrace,
			Rbrace: forStmt.Body.Rbrace,
			List:   stmts,
		}
		cursor.Replace(block)
	}
	return true
}

func unrollLoops(source string) string {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, "", source, 0)
	if err != nil {
		panic("Error unrolling")
	}

	newNode := astutil.Apply(node, genNewBlock, nil)

	// TODO: not safe, dont forget
	//var buf *bytes.Buffer = nil
	var buf bytes.Buffer
	// log.Fatal(buf.String)
	if err := format.Node(&buf, fset, newNode); err != nil {
		log.Fatal(err)
	}
	return buf.String()
}
