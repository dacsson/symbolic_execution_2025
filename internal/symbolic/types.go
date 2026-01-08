// Package symbolic определяет базовые типы символьных выражений
package symbolic

import "github.com/ebukreev/go-z3/z3"

// ExpressionType представляет тип символьного выражения
type ExpressionType int

const (
	IntType ExpressionType = iota
	FloatType
	BoolType
	ArrayType
	AddrType
	ObjType
	FuncType
	// Добавьте другие типы по необходимости
)

// String возвращает строковое представление типа
func (et ExpressionType) String() string {
	switch et {
	case IntType:
		return "int"
	case BoolType:
		return "bool"
	case ArrayType:
		return "array"
	case FloatType:
		return "float"
<<<<<<< HEAD
	case AddrType:
		return "address"
	case ObjType:
		return "object"
=======
>>>>>>> origin/main
	default:
		return "unknown"
	}
}

// AsSort type to sort (array?)
func (et ExpressionType) AsSort(ctx *z3.Context) z3.Sort {
	switch et {
	case IntType:
		return ctx.IntSort()
	case BoolType:
		return ctx.BoolSort()
	case FloatType:
		return ctx.FloatSort(8, 24)
	//case ArrayType:
	//	if withTy == nil {
	//		panic("withTy is nil, probably unknown array element type found")
	//	}
	//	return ctx.ArraySort(ctx.IntSort(), withTy.asSort(ctx, nil))
	default:
		panic("unknown type")
	}
}
