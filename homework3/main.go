package main

import (
	"fmt"
	"log"
	"symbolic-execution-course/internal/symbolic"
	"symbolic-execution-course/internal/translator"

	"github.com/ebukreev/go-z3/z3"
)

func main() {

	// Создаём Z3 транслятор
	translator := translator.NewZ3Translator()
	defer translator.Close()

	//var mem = memory.NewSymbolicMemory()
	// TODO: there is gotta be a better way to keep array.expr filled
	//       or maybe i have a flawed logic idk
	var init_dummy = symbolic.NewSymbolicVariable("array", symbolic.ObjType)
	var array = translator.Mem.Allocate(symbolic.ArrayType, "", init_dummy)
	translator.Mem.AssignToArray(array, 5, symbolic.NewIntConstant(10))

	var fromArray = translator.Mem.GetFromArray(array, 5, symbolic.IntType)
	fmt.Printf("Z3 выражение создано: %T\n", fromArray)

	var anotherFromArray = translator.Mem.GetFromArray(array, 10, symbolic.IntType)
	fmt.Printf("Z3 выражение создано: %T\n", anotherFromArray)

	z3Test01Expr, err := translator.TranslateExpression(fromArray)
	if err != nil {
		log.Fatalf("Ошибка трансляции fromArray[5]: %v", err)
	}
	fmt.Printf("Result: %T\n", z3Test01Expr)

	z3Test02Expr, err := translator.TranslateExpression(anotherFromArray)
	if err != nil || z3Test02Expr == nil {
		log.Printf("[EXPECTED] Ошибка трансляции fromArray[10] out of bounds: %v", err)
	}
	fmt.Printf("Result: %T\n", z3Test02Expr)

	// Test1
	test1 := `
func testStructBasic() Person {
	var p Person
	p.Age = 25
	p.ID = 1001
	return p
}
`

	fmt.Printf("#=== TEST1 ===\n")
	fmt.Println(test1)

	//p_Age := translator.Mem.Allocate(symbolic.ArrayType, "", init_dummy)
	// Fields as arrays (but why)
	p_Age := symbolic.NewSymbolicArray("p_Age", symbolic.IntType, 1)

	p_Age_Ass := symbolic.NewBinaryOperation(p_Age, symbolic.NewIntConstant(25), symbolic.FIELD_ASSIGN)
	//_ = translator.Mem.AssignToArray(p_Age, 0, symbolic.NewIntConstant(25))
	//translator.Mem.AssignToArray(p_Age, 0, symbolic.NewIntConstant(25))

	p_ID := symbolic.NewSymbolicArray("p_ID", symbolic.IntType, 1)
	// p.ID = 1001
	p_ID_Ass := symbolic.NewBinaryOperation(p_ID, symbolic.NewIntConstant(1001), symbolic.FIELD_ASSIGN)
	//_ = translator.Mem.AssignToArray(p_ID, 0, symbolic.NewIntConstant(1001))
	//translator.Mem.AssignToArray(p_ID, 0, symbolic.NewIntConstant(1001))

	z3Test1Expr, err := translator.TranslateExpression(p_ID_Ass)
	if err != nil || z3Test1Expr == nil {
		log.Fatalf("Ошибка трансляции : %v", err)
	}
	fmt.Printf("Result: %T\n", z3Test1Expr)

	z3Test2Expr, err := translator.TranslateExpression(p_Age_Ass)
	if err != nil || z3Test2Expr == nil {
		log.Fatalf("Ошибка трансляции : %v", err)
	}
	fmt.Printf("Result: %T\n", z3Test2Expr)

	// Test2
	test2 := `
func testStructPointer() *Person {
	p := &Person{Age: 30, ID: 2002}
	p.Age = p.Age + 5
	return p
}
`

	fmt.Printf("#=== TEST2 ===\n")
	fmt.Println(test2)

	person := translator.Mem.Allocate(symbolic.ObjType, "Person", symbolic.NewSymbolicVariable("p", symbolic.ObjType))

	translator.Mem.AssignField(person, 0, symbolic.NewIntConstant(30))
	translator.Mem.AssignField(person, 1, symbolic.NewIntConstant(2002))

	field1 := translator.Mem.GetFieldValue(person, 0, symbolic.IntType)
	add1 := symbolic.NewBinaryOperation(field1, symbolic.NewIntConstant(5), symbolic.ADD)
	translator.Mem.AssignField(person, 0, add1)

	field1 = translator.Mem.GetFieldValue(person, 0, symbolic.IntType)

	z3Test3Expr, err := translator.TranslateExpression(field1)
	if err != nil || z3Test3Expr == nil {
		log.Fatalf("Ошибка трансляции : %v", err)
	}
	fmt.Printf("Result: %T\n", z3Test3Expr)

	// assert(p.Age == 35)
	solver := z3.NewSolver(translator.Ctx)
	assert35 := z3Test3Expr.(z3.Int).Eq(translator.Ctx.FromInt(35, translator.Ctx.IntSort()).(z3.Int))
	solver.Assert(assert35)
	sat, _ := solver.Check()
	if sat {
		fmt.Printf("[ASSERT] [SAT] p.Age == 35: Result: %s\n", solver.Model().String())
	} else {
		log.Fatalf("[ASSERT] [UNSAT] p.Age == 35")
	}

	// Test3
	test3 := `
func testArrayModification(arr [5]int) [5]int {
	for i := range arr {
		arr[i] = arr[i] + 1
	}
	return arr
}
`

	fmt.Printf("#=== TEST3 ===\n")
	fmt.Println(test3)

	arr2 := translator.Mem.Allocate(symbolic.ArrayType, "", init_dummy)

	// mutate
	var assignments [5]symbolic.SymbolicExpression
	for i := int64(0); i < 5; i++ {
		ell1 := translator.Mem.GetFieldValue(arr2, int(i), symbolic.IntType)
		mul := symbolic.NewBinaryOperation(ell1, symbolic.NewIntConstant(i), symbolic.ADD)
		i_ass := translator.Mem.AssignField(arr2, int(i), mul)
		assignments[i] = i_ass
	}
	arr3 := translator.Mem.GetFieldValue(arr2, 4, symbolic.IntType)

	z3Test4Expr, err := translator.TranslateExpression(arr3)
	if err != nil || z3Test4Expr == nil {
		log.Fatalf("Ошибка трансляции : %v", err)
	}
	fmt.Printf("Result: %T\n", z3Test4Expr)

	// assert(arr[2] != 0)
	solver2 := z3.NewSolver(translator.Ctx)
	assertNZero := z3Test4Expr.(z3.Int).NE(translator.Ctx.FromInt(0, translator.Ctx.IntSort()).(z3.Int))
	solver2.Assert(assertNZero)
	sat2, _ := solver2.Check()
	if sat2 {
		fmt.Printf("[ASSERT] [SAT] arr[2] != 0: Result: %s\n", solver2.Model().String())
	} else {
		log.Fatalf("[ASSERT] [UNSAT] arr[2] != 0")
	}

	// Test4 — Explicit aliasing: foo2 aliases foo1
	test4 := `
func AliasingExplicit(foo1 *Foo) int {
    foo2 := foo1    // explicit alias
    foo2.a = 5
    foo1.a = 2
    if foo2.a == 2 {
        return 4
    }
    return 5
}
`

	fmt.Printf("#=== TEST4: Explicit Aliasing ===\n")
	fmt.Println(test4)

	// Allocate foo1
	foo1 := translator.Mem.Allocate(
		symbolic.ObjType,
		"Foo",
		symbolic.NewSymbolicVariable("foo1", symbolic.ObjType),
	)

	foo2 := foo1 // is this how we should test it lol?

	translator.Mem.AssignField(foo2, 0, symbolic.NewIntConstant(5))
	translator.Mem.AssignField(foo1, 0, symbolic.NewIntConstant(2))

	// --- (should be 2 if aliasing works)
	foo2_a := translator.Mem.GetFieldValue(foo2, 0, symbolic.IntType)

	// Translate
	z3Foo2a, err := translator.TranslateExpression(foo2_a)
	if err != nil {
		log.Fatalf("Ошибка трансляции foo2.a: %v", err)
	}

	fmt.Printf("Z3 expression type: %T\n", z3Foo2a)

	// Assert foo2.a == 2 (aliasing effect)
	solver3 := z3.NewSolver(translator.Ctx)
	solver3.Assert(
		z3Foo2a.(z3.Int).Eq(
			translator.Ctx.FromInt(2, translator.Ctx.IntSort()).(z3.Int),
		),
	)

	sat3, _ := solver3.Check()
	if sat3 {
		fmt.Printf("[ALIAS TEST] [SAT] foo2.a == 2 (aliasing works!!!)\n")
		fmt.Printf("Model: %s\n", solver3.Model().String())
	} else {
		log.Fatalf("[ALIAS TEST] [UNSAT] expected foo2.a == 2 when aliasing!")
	}

}
