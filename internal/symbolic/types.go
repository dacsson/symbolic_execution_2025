// Package symbolic определяет базовые типы символьных выражений
package symbolic

// ExpressionType представляет тип символьного выражения
type ExpressionType int

const (
	IntType ExpressionType = iota
	FloatType
	BoolType
	ArrayType
	AddrType
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
	case AddrType:
		return "address"
	default:
		return "unknown"
	}
}
