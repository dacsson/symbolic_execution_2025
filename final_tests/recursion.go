package final_tests

import (
	"errors"
)

func Factorial(n int) (int, error) {
	if n < 0 {
		return 0, errors.New("IllegalArgumentException")
	}
	if n == 0 {
		return 1, nil
	}
	result, _ := Factorial(n - 1)
	return n * result, nil
}

func InfiniteRecursion(i int) {
	InfiniteRecursion(i + 1)
}
