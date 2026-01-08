package main

import (
    "flag"
    "fmt"
    "go/ast"
    "go/parser"
    "go/token"
    "os"
    "path/filepath"
    "strings"

    "symbolic-execution-course/internal"
)

func runTest(name, source, funcName string) {
    fmt.Printf("\n======== Test %s =========\n", name)

    // print file content
    fmt.Println("File content:")
    fmt.Println(source)

    results := internal.Analyse(source, funcName)

    for i, interpreter := range results {
        fmt.Printf("* Path %d:\n", i)
        fmt.Printf("  - Path condition: %s\n", interpreter.PathCondition.String())
        if frame := interpreter.GetCurrentFrame(); frame != nil && frame.ReturnValue != nil {
            fmt.Printf("  - Return value: %s\n\n", frame.ReturnValue.String())
        }
    }
    fmt.Printf("\n======== End of Test %s =========\n", name)
}

func loadSource(root string) (string, error) {
    absRoot, err := filepath.Abs(root)
    if err != nil {
        return "", err
    }

    info, err := os.Stat(absRoot)
    if err != nil {
        return "", err
    }

    var sb strings.Builder

    readFile := func(p string) error {
        if strings.HasSuffix(p, "_test.go") {
            return nil
        }
        b, err := os.ReadFile(p)
        if err != nil {
            return err
        }
        // runner is ignored
        if strings.Contains(string(b), "func main() {") {
            return nil
        }
        sb.Write(b)
        sb.WriteString("\n\n")
        return nil
    }

    if info.IsDir() {
        err = filepath.WalkDir(absRoot, func(p string, d os.DirEntry, walkErr error) error {
            if walkErr != nil || d.IsDir() {
                return walkErr
            }
            if strings.HasSuffix(p, ".go") {
                return readFile(p)
            }
            return nil
        })
        if err != nil {
            return "", err
        }
    } else {
        if strings.HasSuffix(absRoot, ".go") {
            if err := readFile(absRoot); err != nil {
                return "", err
            }
        } else {
            return "", fmt.Errorf("%s is not a .go file", absRoot)
        }
    }
    return sb.String(), nil
}

func collectFuncNames(source string) ([]string, error) {
    fset := token.NewFileSet()
    file, err := parser.ParseFile(fset, "", source, parser.ParseComments)
    if err != nil {
        return nil, err
    }
    var names []string
    ast.Inspect(file, func(n ast.Node) bool {
        if fn, ok := n.(*ast.FuncDecl); ok && fn.Name != nil {
            if fn.Recv == nil {
                names = append(names, fn.Name.Name)
            }
        }
        return true
    })
    return names, nil
}

func main() {
    pathFlag := flag.String("path", ".", "relative path to a .go file or a directory containing .go files")
    funcFlag := flag.String("func", "", "comma‑separated list of function names to test (optional). If omitted, all functions are tested.")
    flag.Parse()

    source, err := loadSource(*pathFlag)
    if err != nil {
        fmt.Fprintf(os.Stderr, "failed to load source: %v\n", err)
        os.Exit(1)
    }

    var fnNames []string
    if strings.TrimSpace(*funcFlag) != "" {
        for _, part := range strings.Split(*funcFlag, ",") {
            name := strings.TrimSpace(part)
            if name != "" {
                fnNames = append(fnNames, name)
            }
        }
    } else {
        fnNames, err = collectFuncNames(source)
        if err != nil {
            fmt.Fprintf(os.Stderr, "failed to parse source for function names: %v\n", err)
            os.Exit(1)
        }
        if len(fnNames) == 0 {
            fmt.Println("no functions found in the supplied source")
            return
        }
    }

    for _, fn := range fnNames {
        runTest(fn, source, fn)
    }
}
