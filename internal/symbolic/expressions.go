// Package symbolic содержит конкретные реализации символьных выражений
package symbolic

import "fmt"

// SymbolicExpression - базовый интерфейс для всех символьных выражений
type SymbolicExpression interface {
	// Type возвращает тип выражения
	Type() ExpressionType

	// String возвращает строковое представление выражения
	String() string

	// Accept принимает visitor для обхода дерева выражений
	Accept(visitor Visitor) interface{}
}

// SymbolicVariable представляет символьную переменную
type SymbolicVariable struct {
	Name     string
	ExprType ExpressionType
}

// NewSymbolicVariable создаёт новую символьную переменную
func NewSymbolicVariable(name string, exprType ExpressionType) *SymbolicVariable {
	return &SymbolicVariable{
		Name:     name,
		ExprType: exprType,
	}
}

// Type возвращает тип переменной
func (sv *SymbolicVariable) Type() ExpressionType {
	return sv.ExprType
}

// String возвращает строковое представление переменной
func (sv *SymbolicVariable) String() string {
	return sv.Name
}

// Accept реализует Visitor pattern
func (sv *SymbolicVariable) Accept(visitor Visitor) interface{} {
	return visitor.VisitVariable(sv)
}

// SymbolicArray Symbolic array type
type SymbolicArray struct {
	Name     string
	ElemType ExpressionType
	Size     uint
	Elements []SymbolicVariable
}

func NewSymbolicArray(name string, elemType ExpressionType, size uint, els []SymbolicVariable) *SymbolicArray {
	return &SymbolicArray{name, elemType, size, els}
}

func (sa *SymbolicArray) Type() ExpressionType {
	return ArrayType
}

func (sa *SymbolicArray) ElType() ExpressionType {
	return sa.ElemType
}

func (sa *SymbolicArray) String() string {
	res := "[ "
	for i, el := range sa.Elements {
		if i == len(sa.Elements)-1 {
			res += el.String()
		} else {
			res += el.String() + ", "
		}
	}

	res += "]"
	return res
}

func (sa *SymbolicArray) Accept(visitor Visitor) interface{} {
	return visitor.VisitArray(sa)
}

// IntConstant представляет целочисленную константу
type IntConstant struct {
	Value int64
}

// NewIntConstant создаёт новую целочисленную константу
func NewIntConstant(value int64) *IntConstant {
	return &IntConstant{Value: value}
}

// Type возвращает тип константы
func (ic *IntConstant) Type() ExpressionType {
	return IntType
}

// String возвращает строковое представление константы
func (ic *IntConstant) String() string {
	return fmt.Sprintf("%d", ic.Value)
}

// Accept реализует Visitor pattern
func (ic *IntConstant) Accept(visitor Visitor) interface{} {
	return visitor.VisitIntConstant(ic)
}

type FloatConstant struct {
	Value float32
}

func NewFloatConstant(value float32) *FloatConstant {
	return &FloatConstant{Value: value}
}

func (fc *FloatConstant) Type() ExpressionType {
	return FloatType
}

func (fc *FloatConstant) String() string {
	return fmt.Sprintf("%f", fc.Value)
}

func (fc *FloatConstant) Accept(visitor Visitor) interface{} {
	return visitor.VisitFloatConstant(fc)
}

// BoolConstant представляет булеву константу
type BoolConstant struct {
	Value bool
}

// NewBoolConstant создаёт новую булеву константу
func NewBoolConstant(value bool) *BoolConstant {
	return &BoolConstant{Value: value}
}

// Type возвращает тип константы
func (bc *BoolConstant) Type() ExpressionType {
	return BoolType
}

// String возвращает строковое представление константы
func (bc *BoolConstant) String() string {
	return fmt.Sprintf("%t", bc.Value)
}

// Accept реализует Visitor pattern
func (bc *BoolConstant) Accept(visitor Visitor) interface{} {
	return visitor.VisitBoolConstant(bc)
}

// BinaryOperation представляет бинарную операцию
type BinaryOperation struct {
	Left     SymbolicExpression
	Right    SymbolicExpression
	Operator BinaryOperator
}

// TODO: Реализуйте следующие методы в рамках домашнего задания

// NewBinaryOperation создаёт новую бинарную операцию
func NewBinaryOperation(left, right SymbolicExpression, op BinaryOperator) *BinaryOperation {
	// Создать новую бинарную операцию и проверить совместимость типов
	if left.Type() != right.Type() {
		return nil
	}
	return &BinaryOperation{Left: left, Right: right, Operator: op}
}

// Type возвращает результирующий тип операции
func (bo *BinaryOperation) Type() ExpressionType {
	// Определить результирующий тип на основе операции и типов операндов
	// Например: int + int = int, int < int = bool
	switch bo.Operator {
	case EQ:
	case GE:
	case GT:
	case LE:
	case LT:
	case NE:
		return BoolType
	case ADD:
	case SUB:
	case MUL:
	case MOD:
		return IntType
	case DIV:
		return FloatType
	}

	// UNREACHABLE?
	return 0
}

// String возвращает строковое представление операции
func (bo *BinaryOperation) String() string {
	// Формат: "(left operator right)"
	return fmt.Sprintf("(%s %s %s)", bo.Left.String(), bo.Operator.String(), bo.Right.String())
}

// Accept реализует Visitor pattern
func (bo *BinaryOperation) Accept(visitor Visitor) interface{} {
	return visitor.VisitBinaryOperation(bo)
}

// LogicalOperation представляет логическую операцию
type LogicalOperation struct {
	Operands []SymbolicExpression
	Operator LogicalOperator
}

// TODO: Реализуйте следующие методы в рамках домашнего задания

// NewLogicalOperation создаёт новую логическую операцию
func NewLogicalOperation(operands []SymbolicExpression, op LogicalOperator) *LogicalOperation {
	// Создать логическую операцию и проверить типы операндов
	return &LogicalOperation{Operands: operands, Operator: op}
}

// Type возвращает тип логической операции (всегда bool)
func (lo *LogicalOperation) Type() ExpressionType {
	return BoolType
}

// String возвращает строковое представление логической операции
func (lo *LogicalOperation) String() string {
	// Для NOT: "!operand"
	// Для AND/OR: "(operand1 && operand2 && ...)"
	// Для IMPLIES: "(operand1 => operand2)"
	res := ""
	switch lo.Operator {
	case AND:
		// Concat all operands
		res = "("
		for i, operand := range lo.Operands {
			if i == len(lo.Operands)-1 {
				res += operand.String()
			} else {
				res += operand.String() + " " + "&& "
			}
		}
		res += ")"
		return res
	case OR:
		// Concat all operands
		res = "("
		for i, operand := range lo.Operands {
			if i == len(lo.Operands)-1 {
				res += operand.String()
			} else {
				res += operand.String() + " " + "|| "
			}
		}
		res += ")"
	case IMPLIES:
		res = fmt.Sprintf("%s => %s", lo.Operands[0].String(), lo.Operands[1].String())
	}

	return res
}

// Accept реализует Visitor pattern
func (lo *LogicalOperation) Accept(visitor Visitor) interface{} {
	return visitor.VisitLogicalOperation(lo)
}

// Операторы для бинарных выражений
type BinaryOperator int

const (
	// Арифметические операторы
	ADD BinaryOperator = iota
	SUB
	MUL
	DIV
	MOD

	// Операторы сравнения
	EQ // равно
	NE // не равно
	LT // меньше
	LE // меньше или равно
	GT // больше
	GE // больше или равно
)

// String возвращает строковое представление оператора
func (op BinaryOperator) String() string {
	switch op {
	case ADD:
		return "+"
	case SUB:
		return "-"
	case MUL:
		return "*"
	case DIV:
		return "/"
	case MOD:
		return "%"
	case EQ:
		return "=="
	case NE:
		return "!="
	case LT:
		return "<"
	case LE:
		return "<="
	case GT:
		return ">"
	case GE:
		return ">="
	default:
		return "unknown"
	}
}

// Логические операторы
type LogicalOperator int

const (
	AND LogicalOperator = iota
	OR
	//NOT
	IMPLIES
)

// String возвращает строковое представление логического оператора
func (op LogicalOperator) String() string {
	switch op {
	case AND:
		return "&&"
	case OR:
		return "||"
	case IMPLIES:
		return "=>"
	default:
		return "unknown"
	}
}

type UnaryOperator int

const (
	NOT   UnaryOperator = iota
	MINUS               // MINUS unary minus "-1"
	INCREMENT
	DECREMENT
)

func (op UnaryOperator) String() string {
	switch op {
	case MINUS:
		return "-"
	case INCREMENT:
		return "++"
	case DECREMENT:
		return "--"
	case NOT:
		return "!"
	default:
		return "unknown"
	}
}

type UnaryOperation struct {
	Operand  SymbolicExpression
	Operator UnaryOperator
}

func (uo *UnaryOperation) Type() ExpressionType {
	switch uo.Operator {
	case MINUS:
	case INCREMENT:
	case DECREMENT:
		return IntType
	case NOT:
		return BoolType
	default:
		return BoolType
	}
	panic("unreachable")
}

func (uo *UnaryOperation) String() string {
	res := ""
	switch uo.Operator {
	case NOT:
		res = fmt.Sprintf("!%s", uo.Operand.String())
	case MINUS:
		res = fmt.Sprintf("-%s", uo.Operand.String())
	case INCREMENT:
		res = fmt.Sprintf("%s++", uo.Operand.String())
	case DECREMENT:
		res = fmt.Sprintf("%s--", uo.Operand.String())
	}

	return res
}

// TODO: Добавьте дополнительные типы выражений по необходимости:
// -[x] SymbolicArray
// -[x] UnaryOperation (унарные операции: -x, !x)
// - ArrayAccess (доступ к элементам массива: arr[index])
// - FunctionCall (вызовы функций: f(x, y))
// - ConditionalExpression (тернарный оператор: condition ? true_expr : false_expr)
