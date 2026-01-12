package main

import (
	"fmt"
	"symbolic-execution-course/internal"
)

func runTest(name, source, funcName string) {
	fmt.Printf("\n======== Test %s =========\n", name)
	// instr := buildInstruction(source, funcName)
	// results := internal.interpretDynamically(instr)
	results := internal.Analyse(source, funcName)

	for i, interpreter := range results {
		fmt.Printf("* Path %d:\n", i)
		// fmt.Printf("[DEBUG] Steps %d:\n", interpreter.ExecutionSteps)
		fmt.Printf("  - Path condition: %s\n", interpreter.PathCondition.String())
		if frame := interpreter.GetCurrentFrame(); frame != nil && frame.ReturnValue != nil {
			fmt.Printf("  - Return value: %s\n\n", frame.ReturnValue.String())
		}
	}

	fmt.Printf("\n======== End of Test %s =========\n", name)
}

func main() {
	source := `
package main

func factorial(n int) int {
	if n <= 1 {
		return 1
	}
	return n * factorial(n-1)
}

func testRecursive(x int) int {
	if x < 5 {
		return factorial(x)
	}
	return -1
}
`
	runTest("recursive factorial", source, "testRecursive")

	test1 := `
package main

func isEven(n int) bool {
	if n == 0 {
		return true
	}
	return isOdd(n - 1)
}

func isOdd(n int) bool {
	if n == 0 {
		return false
	}
	return isEven(n - 1)
}

func testMutualRecursion(x int) bool {
	if x >= 0 {
		return isEven(x)
	}
	return false
}
`
	runTest("mutual recursion", test1, "testMutualRecursion")

	test2 := `
package main

func testArrayOperations(n int) int {
	arr := [5]int{1, 2, 3, 4, 5}
	sum := 0

	for i := 0; i < len(arr); i++ {
		if i%2 == 0 {
			sum += arr[i] * n
		} else {
			sum += arr[i]
		}
	}
	return sum
}
`
	runTest("array operations", test2, "testArrayOperations")

	test3 := `
package main

func testSliceDynamic(n int) int {
	slice := make([]int, n)
	for i := 0; i < n; i++ {
		slice[i] = i * i
	}

	result := 0
	for _, val := range slice {
		if val%2 == 0 {
			result += val
		} else {
			result -= val
		}
	}
	return result
}
`
	runTest("dynamic slices", test3, "testSliceDynamic")

	test4 := `
package main

type Point struct {
	X, Y int
}

type Rectangle struct {
	TopLeft, BottomRight Point
}

func (r Rectangle) Area() int {
	width := r.BottomRight.X - r.TopLeft.X
	height := r.TopLeft.Y - r.BottomRight.Y
	if width < 0 || height < 0 {
		return -1
	}
	return width * height
}

func testStructOperations(x1, y1, x2, y2 int) int {
	rect := Rectangle{
		TopLeft:     Point{X: x1, Y: y1},
		BottomRight: Point{X: x2, Y: y2},
	}
	return rect.Area()
}
`
	runTest("struct operations", test4, "testStructOperations")

	test5 := `
package main

type Shape interface {
	Area() int
	Perimeter() int
}

type Circle struct {
	Radius int
}

func (c Circle) Area() int {
	return 3 * c.Radius * c.Radius
}

func (c Circle) Perimeter() int {
	return 2 * 3 * c.Radius
}

type Square struct {
	Side int
}

func (s Square) Area() int {
	return s.Side * s.Side
}

func (s Square) Perimeter() int {
	return 4 * s.Side
}

func testInterface(shape Shape, multiplier int) int {
	area := shape.Area()
	perimeter := shape.Perimeter()

	if area > perimeter {
		return area * multiplier
	}
	return perimeter * multiplier
}
`
	runTest("interface", test5, "testInterface")

	test6 := `
package main

func safeDivide(a, b int) (int, error) {
	if b == 0 {
		return 0, nil // имитация ошибки
	}
	return a / b, nil
}

func testErrorHandling(x, y int) int {
	result, err := safeDivide(x, y)
	if err != nil {
		return -1
	}

	if result > 10 {
		return result * 2
	}
	return result
}
`
	runTest("error handling", test6, "testErrorHandling")

}
