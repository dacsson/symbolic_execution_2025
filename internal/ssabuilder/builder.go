// Package ssabuilder предоставляет функции для построения SSA представления
package ssabuilder

import (
	"fmt"
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"os"

	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/ssautil"
)

// Builder отвечает за построение SSA из исходного кода Go
type Builder struct {
	fset *token.FileSet
}

// NewBuilder создаёт новый экземпляр Builder
func NewBuilder() *Builder {
	return &Builder{
		fset: token.NewFileSet(),
	}
}

// TODO: Реализуйте следующие методы в рамках домашнего задания

// ParseAndBuildSSA парсит исходный код Go и создаёт SSA представление
// Возвращает SSA программу и функцию по имени
func (b *Builder) ParseAndBuildSSA(source string, funcName string) (*ssa.Function, error) {
	fset := token.NewFileSet()

	// 1. Парсинг исходного кода с помощью go/parser
	fmt.Println("#=== 1. Парсинг исходного кода с помощью go/parser ===#")
	file, err := parser.ParseFile(fset, "main.go", source, parser.ParseComments)
	if err != nil {
		fmt.Println("Ошибка при парсинге исходного кода:", err)
		panic("parser error")
	}
	files := []*ast.File{file}

	for _, node := range file.Decls {
		if node, ok := node.(*ast.FuncDecl); ok {
			fmt.Printf("  -- Найдена функция: %s\n", node.Name.Name)
		}
	}
	fmt.Println("#=== Парсинг завершен ===#")

	// 2. Создание SSA программы
	fmt.Println("#=== 2. Создание SSA программы ===#")

	// Create package of source
	pkg := types.NewPackage("homework1/main.go", "main")
	tconfig := &types.Config{Importer: importer.Default()}

	// Build SSA
	ssa_form, _, err := ssautil.BuildPackage(
		tconfig,
		fset,
		pkg,
		files,
		ssa.SanityCheckFunctions,
	)
	if err != nil {
		panic("type error in package")
	}

	// Create SSA
	ssa_form.Build()

	// Print results
	ssa_form.WriteTo(os.Stdout)

	// 3. Поиск нужной функции по имени
	if fn_decl := ssa_form.Func(funcName); fn_decl != nil {
		// Print out the package-level functions.
		ssa_form.Func("init").WriteTo(os.Stdout)
		ssa_form.Func(funcName).WriteTo(os.Stdout)
		fmt.Println("#=== Создание SSA завершено ===#")
		return fn_decl, nil
	}

	return nil, fmt.Errorf("Function %s not found", funcName)
}
