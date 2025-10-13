// Package translator содержит реализацию транслятора в Z3
package translator

import (
	"symbolic-execution-course/internal/symbolic"

	"github.com/ebukreev/go-z3/z3"
)

// Z3Translator транслирует символьные выражения в Z3 формулы
type Z3Translator struct {
	ctx    *z3.Context
	config *z3.Config
	vars   map[string]z3.Value // Кэш переменных
}

// NewZ3Translator создаёт новый экземпляр Z3 транслятора
func NewZ3Translator() *Z3Translator {
	config := &z3.Config{}
	ctx := z3.NewContext(config)

	return &Z3Translator{
		ctx:    ctx,
		config: config,
		vars:   make(map[string]z3.Value),
	}
}

// GetContext возвращает Z3 контекст
func (zt *Z3Translator) GetContext() interface{} {
	return zt.ctx
}

// Reset сбрасывает состояние транслятора
func (zt *Z3Translator) Reset() {
	zt.vars = make(map[string]z3.Value)
}

// Close освобождает ресурсы
func (zt *Z3Translator) Close() {
	// Z3 контекст закрывается автоматически
}

// TranslateExpression транслирует символьное выражение в Z3
func (zt *Z3Translator) TranslateExpression(expr symbolic.SymbolicExpression) (interface{}, error) {
	return expr.Accept(zt), nil
}

// TODO: Реализуйте следующие методы в рамках домашнего задания

// VisitVariable транслирует символьную переменную в Z3
func (zt *Z3Translator) VisitVariable(expr *symbolic.SymbolicVariable) interface{} {
	// Проверить, есть ли переменная в кэше
	// Если нет - создать новую Z3 переменную соответствующего типа
	// Добавить в кэш и вернуть

	// Подсказки:
	// - Используйте zt.ctx.IntConst(name) для int переменных
	// - Используйте zt.ctx.BoolConst(name) для bool переменных
	// - Храните переменные в zt.vars для повторного использования

	if _, exists := zt.vars[expr.Name]; exists {
		return zt.vars[expr.Name]
	}

	if expr.Type() == symbolic.ArrayType {
		panic("you are doing something wrong")
	}
	// Else add to vars
	zt.vars[expr.Name] = zt.createZ3Variable(expr.Name, expr.ExprType)

	return zt.vars[expr.Name]
}

// VisitIntConstant транслирует целочисленную константу в Z3
func (zt *Z3Translator) VisitIntConstant(expr *symbolic.IntConstant) interface{} {
	// Создать Z3 константу с помощью zt.ctx.FromBigInt или аналогичного метода
	return zt.ctx.FromInt(expr.Value, zt.ctx.IntSort())
}

func (zt *Z3Translator) VisitFloatConstant(expr *symbolic.FloatConstant) interface{} {
	return zt.ctx.FromFloat32(expr.Value, zt.ctx.FloatSort(8, 24))
}

// VisitBoolConstant транслирует булеву константу в Z3
func (zt *Z3Translator) VisitBoolConstant(expr *symbolic.BoolConstant) interface{} {
	// Использовать zt.ctx.FromBool для создания Z3 булевой константы
	return zt.ctx.FromBool(expr.Value)
}

// VisitBinaryOperation транслирует бинарную операцию в Z3
func (zt *Z3Translator) VisitBinaryOperation(expr *symbolic.BinaryOperation) interface{} {
	// TODO: Реализовать
	// 1. Транслировать левый и правый операнды
	// 2. В зависимости от оператора создать соответствующую Z3 операцию

	left := expr.Left.Accept(zt)
	right := expr.Right.Accept(zt)

	switch expr.Operator {
	case symbolic.ADD:
		switch expr.Left.Type() {
		case symbolic.IntType:
			return left.(z3.Int).Add(right.(z3.Int))
		case symbolic.FloatType:
			return left.(z3.Float).Add(right.(z3.Float))
		case symbolic.BoolType:
			panic("you are doing something wrong")
		case symbolic.ArrayType:
			panic("you are doing something wrong")
		}
	case symbolic.SUB:
		switch expr.Left.Type() {
		case symbolic.IntType:
			return left.(z3.Int).Sub(right.(z3.Int))
		case symbolic.FloatType:
			return left.(z3.Float).Sub(right.(z3.Float))
		case symbolic.BoolType:
			panic("you are doing something wrong")
		case symbolic.ArrayType:
			panic("you are doing something wrong")
		}
	case symbolic.MUL:
		switch expr.Left.Type() {
		case symbolic.IntType:
			return left.(z3.Int).Mul(right.(z3.Int))
		case symbolic.FloatType:
			return left.(z3.Float).Mul(right.(z3.Float))
		default:
			panic("you are doing something wrong")
		}
	case symbolic.DIV:
		switch expr.Left.Type() {
		case symbolic.IntType:
			return left.(z3.Int).Div(right.(z3.Int))
		case symbolic.FloatType:
			return left.(z3.Float).Div(right.(z3.Float))
		default:
			panic("you are doing something wrong")
		}
	case symbolic.MOD:
		switch expr.Left.Type() {
		case symbolic.IntType:
			return left.(z3.Int).Mod(right.(z3.Int))
		default:
			panic("you are doing something wrong")
		}
	case symbolic.EQ:
		switch expr.Left.Type() {
		case symbolic.BoolType:
			left.(z3.Bool).Eq(right.(z3.Bool))
		case symbolic.IntType:
			return left.(z3.Int).Eq(right.(z3.Int))
		case symbolic.FloatType:
			return left.(z3.Float).Eq(right.(z3.Float))
		case symbolic.ArrayType:
			panic("you are doing something wrong")
		}
	case symbolic.NE:
		switch expr.Left.Type() {
		case symbolic.BoolType:
			left.(z3.Bool).NE(right.(z3.Bool))
		case symbolic.IntType:
			return left.(z3.Int).NE(right.(z3.Int))
		case symbolic.FloatType:
			return left.(z3.Float).NE(right.(z3.Float))
		case symbolic.ArrayType:
			panic("you are doing something wrong")
		}
	case symbolic.LT:
		switch expr.Left.Type() {
		case symbolic.BoolType:
			panic("you are doing something wrong")
		case symbolic.IntType:
			return left.(z3.Int).LT(right.(z3.Int))
		case symbolic.FloatType:
			return left.(z3.Float).LT(right.(z3.Float))
		case symbolic.ArrayType:
			panic("you are doing something wrong")
		}
	case symbolic.LE:
		switch expr.Left.Type() {
		case symbolic.BoolType:
			panic("you are doing something wrong")
		case symbolic.IntType:
			return left.(z3.Int).LE(right.(z3.Int))
		case symbolic.FloatType:
			return left.(z3.Float).LE(right.(z3.Float))
		case symbolic.ArrayType:
			panic("you are doing something wrong")
		}
	case symbolic.GT:
		switch expr.Left.Type() {
		case symbolic.BoolType:
			panic("you are doing something wrong")
		case symbolic.IntType:
			return left.(z3.Int).GT(right.(z3.Int))
		case symbolic.FloatType:
			return left.(z3.Float).GT(right.(z3.Float))
		case symbolic.ArrayType:
			panic("you are doing something wrong")
		}
	case symbolic.GE:
		switch expr.Left.Type() {
		case symbolic.BoolType:
			panic("you are doing something wrong")
		case symbolic.IntType:
			return left.(z3.Int).GE(right.(z3.Int))
		case symbolic.FloatType:
			return left.(z3.Float).GE(right.(z3.Float))
		case symbolic.ArrayType:
			panic("you are doing something wrong")
		}
	}

	// Подсказки по операциям в Z3:
	// - Арифметические: left.Add(right), left.Sub(right), left.Mul(right), left.Div(right)
	// - Сравнения: left.Eq(right), left.LT(right), left.LE(right), etc.
	// - Приводите типы: left.(z3.Int), right.(z3.Int) для int операций

	panic("unreachable")
}

// VisitLogicalOperation транслирует логическую операцию в Z3
func (zt *Z3Translator) VisitLogicalOperation(expr *symbolic.LogicalOperation) interface{} {
	// 1. Транслировать все операнды
	// 2. Применить соответствующую логическую операцию

	//left := expr.Left.Accept(zt)
	//right := expr.Right.Accept(zt)

	switch expr.Operator {
	case symbolic.AND:
		operand := expr.Operands[0].Accept(zt)
		next := expr.Operands[1].Accept(zt)

		res := operand.(z3.Bool).And(next.(z3.Bool))

		for i := 2; i < len(expr.Operands); i++ {
			operand := expr.Operands[i].Accept(zt)

			switch expr.Operands[i].Type() {
			case symbolic.BoolType:
				res = res.And(operand.(z3.Bool))
			default:
				panic("you are doing something wrong")
			}
		}

		return res
	case symbolic.OR:
		operand := expr.Operands[0].Accept(zt)
		next := expr.Operands[1].Accept(zt)

		res := operand.(z3.Bool).Or(next.(z3.Bool))

		for i := 2; i < len(expr.Operands); i++ {
			operand := expr.Operands[i].Accept(zt)

			switch expr.Operands[i].Type() {
			case symbolic.BoolType:
				res = res.Or(operand.(z3.Bool))
			default:
				panic("you are doing something wrong")
			}
		}

		return res
	case symbolic.IMPLIES:
		operand := expr.Operands[0].Accept(zt)
		next := expr.Operands[1].Accept(zt)

		return operand.(z3.Bool).Implies(next.(z3.Bool))
	}

	// Подсказки:
	// - AND: zt.ctx.And(operands...)
	// - OR: zt.ctx.Or(operands...)
	// - NOT: operand.Not() (для единственного операнда)
	// - IMPLIES: antecedent.Implies(consequent)

	panic("не реализовано")
}

func (zt *Z3Translator) VisitArray(expr *symbolic.SymbolicArray) interface{} {
	zt.vars[expr.Name] = zt.createZ3Array(expr.Name, *expr)
	return zt.vars[expr.Name]
}

func (zt *Z3Translator) VisitUnaryOperation(expr *symbolic.UnaryOperation) interface{} {
	operand := expr.Operand.Accept(zt)
	switch expr.Operator {
	case symbolic.NOT:
		return operand.(z3.Bool).Not()
	case symbolic.INCREMENT:
		one := zt.ctx.FromInt(1, zt.ctx.IntSort()).(z3.Int)
		return operand.(z3.Int).Add(one)
	case symbolic.DECREMENT:
		one := zt.ctx.FromInt(1, zt.ctx.IntSort()).(z3.Int)
		return operand.(z3.Int).Sub(one)
	case symbolic.MINUS:
		// Is this bad lol?
		value := operand.(symbolic.IntConstant).Value
		return zt.ctx.FromInt(value*(-1), zt.ctx.IntSort()).(z3.Int)
	}

	panic("unreachable")
}

// Вспомогательные методы

func (zt *Z3Translator) createZ3Array(name string, expr symbolic.SymbolicArray) z3.Value {
	switch expr.ElType() {
	case symbolic.BoolType:
		zt.vars[name] = zt.ctx.FreshConst(
			name,
			zt.ctx.ArraySort(
				zt.ctx.BoolSort(),
				zt.ctx.BoolSort(),
			),
		)
	case symbolic.IntType:
		zt.vars[name] = zt.ctx.FreshConst(
			name,
			zt.ctx.ArraySort(
				zt.ctx.IntSort(),
				zt.ctx.IntSort(),
			),
		)
	case symbolic.FloatType:
		zt.vars[name] = zt.ctx.FreshConst(
			name,
			zt.ctx.ArraySort(
				zt.ctx.FloatSort(8, 24),
				zt.ctx.FloatSort(8, 24),
			),
		)
	default:
		panic("unimplemented yet")
	}

	return zt.vars[name]
}

// createZ3Variable создаёт Z3 переменную соответствующего типа
func (zt *Z3Translator) createZ3Variable(name string, exprType symbolic.ExpressionType) z3.Value {
	// Создать Z3 переменную на основе типа
	switch exprType {
	case symbolic.FloatType:
		zt.vars[name] = zt.ctx.FreshConst(name, zt.ctx.FloatSort(8, 24))
	case symbolic.IntType:
		zt.vars[name] = zt.ctx.FreshConst(name, zt.ctx.IntSort())
	case symbolic.BoolType:
		zt.vars[name] = zt.ctx.FreshConst(name, zt.ctx.BoolSort())
		//case symbolic.ArrayType:
		//	zt.vars[name] = zt.ctx.FreshConst(name, zt.ctx.ArraySort(zt.ctx.IntSort(), zt.ctx.IntSort()))
	default:
		panic("unhandled default case")
	}

	return zt.vars[name]
}

// castToZ3Type приводит значение к нужному Z3 типу
func (zt *Z3Translator) castToZ3Type(value interface{}, targetType symbolic.ExpressionType) (z3.Value, error) {
	// Безопасно привести interface{} к конкретному Z3 типу

	switch targetType {
	case symbolic.IntType:
		return value.(z3.Int), nil
	case symbolic.FloatType:
		return value.(z3.Float), nil
	case symbolic.BoolType:
		return value.(z3.Bool), nil
	case symbolic.ArrayType:
		return value.(z3.Array), nil
	}

	panic("unknown expression type")
}
