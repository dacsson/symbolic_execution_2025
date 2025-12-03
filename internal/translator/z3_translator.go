// Package translator содержит реализацию транслятора в Z3
package translator

import (
	"math/big"
	"strconv"
	"symbolic-execution-course/internal/memory"
	"symbolic-execution-course/internal/symbolic"

	"github.com/ebukreev/go-z3/z3"
)

// Z3Translator транслирует символьные выражения в Z3 формулы
type Z3Translator struct {
	Ctx    *z3.Context
	config *z3.Config
	vars   map[string]z3.Value    // Кэш переменных
	objs   map[string]z3.Array    // Кэш обхектов
	Mem    *memory.SymbolicMemory // Мем ори
}

// NewZ3Translator создаёт новый экземпляр Z3 транслятора
func NewZ3Translator() *Z3Translator {
	config := &z3.Config{}
	ctx := z3.NewContext(config)

	return &Z3Translator{
		Ctx:    ctx,
		config: config,
		vars:   make(map[string]z3.Value),
		objs:   make(map[string]z3.Array),
		Mem:    memory.NewSymbolicMemory(),
	}
}

// GetContext возвращает Z3 контекст
func (zt *Z3Translator) GetContext() interface{} {
	return zt.Ctx
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
	// - Используйте zt.Ctx.IntConst(name) для int переменных
	// - Используйте zt.Ctx.BoolConst(name) для bool переменных
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
	// Создать Z3 константу с помощью zt.Ctx.FromBigInt или аналогичного метода
	return zt.Ctx.FromInt(expr.Value, zt.Ctx.IntSort())
}

func (zt *Z3Translator) VisitFloatConstant(expr *symbolic.FloatConstant) interface{} {
	return zt.Ctx.FromFloat32(expr.Value, zt.Ctx.FloatSort(8, 24))
}

// VisitBoolConstant транслирует булеву константу в Z3
func (zt *Z3Translator) VisitBoolConstant(expr *symbolic.BoolConstant) interface{} {
	// Использовать zt.Ctx.FromBool для создания Z3 булевой константы
	return zt.Ctx.FromBool(expr.Value)
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
	case symbolic.INDEX:
		switch expr.Left.Type() {
		case symbolic.ArrayType:
			return left.(z3.Array).Select(right.(z3.Value))
		default:
			panic("NOT AN ARRAY TYPE what are you doing")
		}
	case symbolic.FIELD_ACCESS:
		switch expr.Left.Type() {
		case symbolic.ObjType:
			str := getFieldNameStr(expr.Left.String(), expr.Right.String())
			_, err := zt.objs[str]
			if !err {
				zt.objs[str] = z3.Array{}
			}

			return zt.objs[str].Select(right.(z3.Value))
		default:
			panic("you are doing something wrong")
		}
	case symbolic.FIELD_ASSIGN:
		switch expr.Left.Type() {
		case symbolic.ArrayType:
			bigint := big.NewInt(0)
			zr := zt.Ctx.FromBigInt(bigint, zt.Ctx.IntSort())
			return left.(z3.Array).Store(zr, right.(z3.Value))
		default:
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
	// - AND: zt.Ctx.And(operands...)
	// - OR: zt.Ctx.Or(operands...)
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
		one := zt.Ctx.FromInt(1, zt.Ctx.IntSort()).(z3.Int)
		return operand.(z3.Int).Add(one)
	case symbolic.DECREMENT:
		one := zt.Ctx.FromInt(1, zt.Ctx.IntSort()).(z3.Int)
		return operand.(z3.Int).Sub(one)
	case symbolic.MINUS:
		// Is this bad lol?
		value, _, _ := operand.(z3.Int).AsInt64()
		return zt.Ctx.FromInt(value*(-1), zt.Ctx.IntSort()).(z3.Int)
	}

	panic("unreachable")
}

func (zt *Z3Translator) VisitArrayAccess(expr *symbolic.ArrayAccess) interface{} {
	arr := expr.Array.Accept(zt).(z3.Array)
	i := expr.Index.Accept(zt).(z3.Int) // TODO: can it be a non int value?
	return arr.Select(i)
}

func (zt *Z3Translator) VisitConditional(expr *symbolic.ConditionalOperation) interface{} {
	cond := expr.Condition.Accept(zt).(z3.Bool)
	btrue := expr.TrueBlock[0].Accept(zt).(z3.Value)
	for _, block := range expr.TrueBlock[1:] {
		btrue = block.Accept(zt).(z3.Value)
	}

	bfalse := expr.FalseBlock[0].Accept(zt).(z3.Value)
	for _, block := range expr.FalseBlock[1:] {
		bfalse = block.Accept(zt).(z3.Value)
	}

	return cond.IfThenElse(btrue, bfalse)
}

func (zt *Z3Translator) VisitPointer(expr *symbolic.SymbolicPointer) interface{} {
	switch expr.PointerType {
	case symbolic.ArrayType:
	case symbolic.ObjType:
		return expr.Address
	default:
		return zt.Mem.GetPrimitive(expr).(z3.Value)
	}
	// why do i need THIS??
	return 0
}

// Вспомогательные методы

func (zt *Z3Translator) createZ3Array(name string, expr symbolic.SymbolicArray) z3.Value {
	switch expr.ElType() {
	case symbolic.BoolType:
		zt.vars[name] = zt.Ctx.FreshConst(
			name,
			zt.Ctx.ArraySort(
				zt.Ctx.BoolSort(),
				zt.Ctx.BoolSort(),
			),
		)
	case symbolic.IntType:
		zt.vars[name] = zt.Ctx.FreshConst(
			name,
			zt.Ctx.ArraySort(
				zt.Ctx.IntSort(),
				zt.Ctx.IntSort(),
			),
		)
	case symbolic.FloatType:
		zt.vars[name] = zt.Ctx.FreshConst(
			name,
			zt.Ctx.ArraySort(
				zt.Ctx.FloatSort(8, 24),
				zt.Ctx.FloatSort(8, 24),
			),
		)
	default:
		panic("unimplemented yet")
	}

	return zt.vars[name]
}

func (zt *Z3Translator) VisitFieldAccess(expr *symbolic.FieldAccess) interface{} {
	// Guard for - for example array out of bounds index expr
	//if expr.Key == nil {
	//	return nil
	//}

	name := getFieldName(expr.Key.String(), expr.FieldIdx)
	index := zt.Ctx.Const(name, zt.Ctx.IntSort())

	fieldName := getFieldName(expr.StructName, expr.FieldIdx)
	_, err := zt.objs[fieldName]
	if !err {
		// TODO: sort can be array too
		as := zt.Ctx.ArraySort(zt.Ctx.IntSort(), expr.Type().AsSort(zt.Ctx))
		z := zt.Ctx.Const(expr.Obj.String(), as)
		zt.objs[fieldName] = z.(z3.Array)
	}

	return zt.objs[fieldName].Select(index)
}

func (zt *Z3Translator) VisitFieldAssign(expr *symbolic.FieldAssign) interface{} {
	str := getFieldName(expr.Obj.String(), expr.FieldIdx)
	index := zt.Ctx.Const(str, zt.Ctx.IntSort())

	fieldName := getFieldName(expr.StructName, expr.FieldIdx)
	_, err := zt.objs[fieldName]
	if !err {
		as := zt.Ctx.ArraySort(zt.Ctx.IntSort(), expr.Type().AsSort(zt.Ctx))
		z := zt.Ctx.Const(expr.Obj.String(), as)
		zt.objs[fieldName] = z.(z3.Array)
	}

	visit := expr.Value.Accept(zt)
	zt.objs[fieldName] = zt.objs[fieldName].Store(index, visit.(z3.Value))
	return zt.objs[fieldName]
}

func (zt *Z3Translator) VisitFunction(expr *symbolic.Function) interface{} {
	var argsSorts []z3.Sort
	for i := range expr.Args {
		argTy := expr.Args[i]
		argsSorts = append(argsSorts, argTy.AsSort(zt.Ctx))
	}
	return zt.Ctx.FuncDecl(expr.Name, argsSorts, expr.ReturnType.AsSort(zt.Ctx))
}

func (zt *Z3Translator) VisitFunctionCall(expr *symbolic.FunctionCall) interface{} {
	decl := zt.VisitFunction(&expr.FunctionDecl)
	var args []z3.Value
	for i := range expr.Args {
		arg := expr.Args[i]
		translatedArg := arg.Accept(zt)
		args = append(args, translatedArg.(z3.Value))
	}
	return decl.(z3.FuncDecl).Apply(args...)
}

// Mangling
func getFieldName(name string, index int) string {
	return "index_" + name + "." + strconv.Itoa(index)
}

func getFieldNameStr(name string, field string) string {
	return name + "." + field
}

// CreateABV Create Array of bitvectors with given bitvec size
// Array Int → BitVec(8)
func (zt *Z3Translator) CreateABV(bits int) z3.Sort {
	return zt.Ctx.ArraySort(zt.Ctx.IntSort(), zt.Ctx.BVSort(bits))
}

// createZ3Variable создаёт Z3 переменную соответствующего типа
func (zt *Z3Translator) createZ3Variable(name string, exprType symbolic.ExpressionType) z3.Value {
	// Создать Z3 переменную на основе типа
	switch exprType {
	case symbolic.FloatType:
		zt.vars[name] = zt.Ctx.FreshConst(name, zt.Ctx.FloatSort(8, 24))
	case symbolic.IntType:
		zt.vars[name] = zt.Ctx.FreshConst(name, zt.Ctx.IntSort())
	case symbolic.BoolType:
		zt.vars[name] = zt.Ctx.FreshConst(name, zt.Ctx.BoolSort())
		//case symbolic.ArrayType:
		//	zt.vars[name] = zt.Ctx.FreshConst(name, zt.Ctx.ArraySort(zt.Ctx.IntSort(), zt.Ctx.IntSort()))
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
