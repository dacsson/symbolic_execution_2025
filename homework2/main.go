// Демонстрационная программа для тестирования символьных выражений
package main

import (
	"fmt"
	"log"
	"symbolic-execution-course/internal/symbolic"
	"symbolic-execution-course/internal/translator"
)

func main() {
	fmt.Println("=== Symbolic Expressions Demo ===")

	// Создаём простые символьные выражения
	x := symbolic.NewSymbolicVariable("x", symbolic.IntType)
	y := symbolic.NewSymbolicVariable("y", symbolic.IntType)
	five := symbolic.NewIntConstant(5)

	// Создаём выражение: x + y > 5
	sum := symbolic.NewBinaryOperation(x, y, symbolic.ADD)
	condition := symbolic.NewBinaryOperation(sum, five, symbolic.GT)

	fmt.Printf("Выражение: %s\n", condition.String())
	fmt.Printf("Тип выражения: %s\n", condition.Type().String())

	// Создаём Z3 транслятор
	translator := translator.NewZ3Translator()
	defer translator.Close()

	// Транслируем в Z3
	z3Expr, err := translator.TranslateExpression(condition)
	if err != nil {
		log.Fatalf("Ошибка трансляции: %v", err)
	}

	fmt.Printf("Z3 выражение создано: %T\n", z3Expr)

	// Создаём более сложное выражение: (x > 0) && (y < 10)
	zero := symbolic.NewIntConstant(0)
	ten := symbolic.NewIntConstant(10)

	cond1 := symbolic.NewBinaryOperation(x, zero, symbolic.GT)
	cond2 := symbolic.NewBinaryOperation(y, ten, symbolic.LT)

	andExpr := symbolic.NewLogicalOperation([]symbolic.SymbolicExpression{cond1, cond2}, symbolic.AND)

	fmt.Printf("Сложное выражение: %s\n", andExpr.String())

	// Транслируем сложное выражение
	z3AndExpr, err := translator.TranslateExpression(andExpr)
	if err != nil {
		log.Fatalf("Ошибка трансляции сложного выражения: %v", err)
	}
	fmt.Printf("Сложное Z3 выражение создано: %T\n", z3AndExpr)

	// Test1
	test1 := `
func add(a, b int) int {
	return a + b
}
`

	fmt.Printf("#=== TEST1 ===\n")
	fmt.Println(test1)

	a := symbolic.NewSymbolicVariable("a", symbolic.IntType)
	b := symbolic.NewSymbolicVariable("b", symbolic.IntType)
	addition := symbolic.NewBinaryOperation(a, b, symbolic.ADD)
	fmt.Printf("SMT: %s\n", addition.String())

	z3Test1Expr, err := translator.TranslateExpression(addition)
	if err != nil {
		log.Fatalf("Ошибка трансляции сложного выражения: %v", err)
	}
	fmt.Printf("Result: %T\n", z3Test1Expr)

	// TEST2
	test2 := `
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
`

	fmt.Printf("#=== TEST2 ===\n")
	fmt.Println(test2)

	a2 := symbolic.NewSymbolicVariable("a", symbolic.IntType)
	b2 := symbolic.NewSymbolicVariable("b", symbolic.IntType)

	btrue := []symbolic.SymbolicExpression{a2}
	bfalse := []symbolic.SymbolicExpression{b2}
	condition1 := symbolic.NewBinaryOperation(a2, b2, symbolic.GT)
	branch := symbolic.NewConditionalOperation(condition1, btrue, bfalse)
	fmt.Printf("SMT: %s\n", branch.String())

	z3Test2, err := translator.TranslateExpression(branch)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}
	fmt.Printf("Result: %T\n", z3Test2)

	test3 := ` 
func calculate(x, y int) int {
	sum := x + y
	diff := x - y
	product := sum * diff
	return product
}
`

	fmt.Printf("#=== TEST3 ===\n")
	fmt.Println(test3)

	x1 := symbolic.NewSymbolicVariable("x", symbolic.IntType)
	y1 := symbolic.NewSymbolicVariable("y", symbolic.IntType)
	sum1 := symbolic.NewBinaryOperation(x1, y1, symbolic.ADD)
	diff1 := symbolic.NewBinaryOperation(x1, y1, symbolic.SUB)
	product := symbolic.NewBinaryOperation(sum1, diff1, symbolic.MUL)
	fmt.Printf("SMT: %s\n", product.String())

	z3Test3, err := translator.TranslateExpression(product)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}
	fmt.Printf("Result: %T\n", z3Test3)

	test4 := ` 
func signFunction(x int) int {
	if x > 0 {
		return 1
	}
	if x < 0 {
		return -1
	}
	return 0
}
`
	fmt.Printf("#=== TEST4 ===\n")
	fmt.Println(test4)

	x2 := symbolic.NewSymbolicVariable("x", symbolic.IntType)
	one := symbolic.NewIntConstant(1)
	mone := symbolic.NewIntConstant(-1)

	// x == 0 (last branch)
	//condition4 := symbolic.NewBinaryOperation(x2, zero, symbolic.EQ)
	//btrue4 := []symbolic.SymbolicExpression{zero}
	//var bfalse4 []symbolic.SymbolicExpression
	//branch4 := symbolic.NewConditionalOperation(condition4, btrue4, bfalse4)
	//ops := []symbolic.SymbolicExpression{condition4, zero}
	//branch4 := symbolic.NewLogicalOperation(ops, symbolic.IMPLIES)

	// x < 0
	condition3 := symbolic.NewBinaryOperation(x2, zero, symbolic.LT)
	btrue3 := []symbolic.SymbolicExpression{mone}
	bfalse3 := []symbolic.SymbolicExpression{zero}
	branch3 := symbolic.NewConditionalOperation(condition3, btrue3, bfalse3)

	// x > 0
	btrue2 := []symbolic.SymbolicExpression{one}
	bfalse2 := []symbolic.SymbolicExpression{branch3}
	condition2 := symbolic.NewBinaryOperation(x2, zero, symbolic.GT)
	branch2 := symbolic.NewConditionalOperation(condition2, btrue2, bfalse2)

	fmt.Printf("SMT: %s\n", branch2.String())

	z3Test4, err := translator.TranslateExpression(branch2)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}
	fmt.Printf("Result: %T\n", z3Test4)

	test5 := `
func unaryOps(x int, flag bool) int {
	result := -x
	if !flag {
		result = -result
	}
	return result
}
`

	fmt.Printf("#=== TEST5 ===\n")
	fmt.Println(test5)

	x3 := symbolic.NewSymbolicVariable("x", symbolic.IntType)
	flag := symbolic.NewSymbolicVariable("flag", symbolic.BoolType)
	res := symbolic.NewUnaryOperation(x3, symbolic.MINUS)

	condition4 := symbolic.NewUnaryOperation(flag, symbolic.NOT)

	btrue4 := []symbolic.SymbolicExpression{symbolic.NewUnaryOperation(res, symbolic.MINUS)}
	bfalse4 := []symbolic.SymbolicExpression{res}

	branch4 := symbolic.NewConditionalOperation(condition4, btrue4, bfalse4)

	fmt.Printf("SMT: %s\n", branch4.String())

	z3Test5, err := translator.TranslateExpression(branch4)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}
	fmt.Printf("Result: %T\n", z3Test5)

	test6 := `
func arrayOps(x []int) int {
	if x[0] < 0 {
		return 0
	} else {
		return x[0]
	}
}
`

	fmt.Printf("#=== TEST6 ===\n")
	fmt.Println(test6)

	x4 := symbolic.NewSymbolicArray("x", symbolic.IntType, 1)
	arr_acc := symbolic.NewArrayAccess(*x4, zero)

	condition5 := symbolic.NewBinaryOperation(arr_acc, zero, symbolic.LT)

	btrue5 := []symbolic.SymbolicExpression{zero}
	bfalse5 := []symbolic.SymbolicExpression{arr_acc}
	branch5 := symbolic.NewConditionalOperation(condition5, btrue5, bfalse5)

	fmt.Printf("SMT: %s\n", branch5.String())

	z3Test6, err := translator.TranslateExpression(branch5)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}
	fmt.Printf("Result: %T\n", z3Test6)
}
