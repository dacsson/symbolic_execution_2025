package main

import (
	"fmt"
	"symbolic-execution-course/internal"
)

func main() {
	// NOTE: WIP, sorry

	source := `
	package main

func test1(x int) int {
	if x > 10 {
		return x + 1
	} else {
		return x - 1
	}
}
	`
	result := internal.Analyse(source, "test1")
	for _, interpreter := range result {
		fmt.Println(interpreter.ToString())
	}

	// NOTE: SSA will optimize b away? or rather split it into two ifs
	/*
		func test2(a bool, b bool) bool:
		0:                                                                entry P:0 S:2
			if a goto 3 else 2
		1:                                                              if.then P:1 S:0
			return true:bool
		2:                                                              if.done P:2 S:0
			return false:bool
		3:                                                            cond.true P:1 S:2
			if b goto 1 else 2
	*/
	// TODO: doesn't work -> need to add SymbolicFunction and memorize types properly
	source2 := `
	package main
// test2(a, b bool)
func test2() bool {
	a := true
	b := false
	if a && b {
		return true
	}
	return false
}
	`
	result2 := internal.Analyse(source2, "test2")
	for _, interpreter := range result2 {
		fmt.Println(interpreter.ToString())
	}

	// TODO: add support for variable on rhs
	//		 it will fail on unimplemented phi, but should translate for loop correctly
	source3 := `
	package main
// testForLoop(n int)
func testForLoop(n int) int {
	result := 1
	for i := 1; i <= 10; i++ {
		if i%2 == 0 {
			result *= i
		} else {
			result += i
		}
	}
	return result
}
	`

	result3 := internal.Analyse(source3, "testForLoop")
	for _, interpreter := range result3 {
		fmt.Println(interpreter.ToString())
	}

	source4 := `
package main

func testWhileLoop(n int) int {
	i := 0
	sum := 0

	for i < n {
		sum += i
		i++
	}
	return sum
}
`

	result4 := internal.Analyse(source4, "testWhileLoop")
	for _, interpreter := range result4 {
		fmt.Println(interpreter.ToString())
	}
}
