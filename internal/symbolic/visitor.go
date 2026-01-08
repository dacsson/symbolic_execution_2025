package symbolic

// Visitor интерфейс для обхода символьных выражений (Visitor Pattern)
type Visitor interface {
	VisitVariable(expr *SymbolicVariable) interface{}
	VisitIntConstant(expr *IntConstant) interface{}
	VisitFloatConstant(expr *FloatConstant) interface{}
	VisitBoolConstant(expr *BoolConstant) interface{}
	VisitBinaryOperation(expr *BinaryOperation) interface{}
	VisitLogicalOperation(expr *LogicalOperation) interface{}
	VisitArray(expr *SymbolicArray) interface{}
	VisitUnaryOperation(expr *UnaryOperation) interface{}
	VisitArrayAccess(expr *ArrayAccess) interface{}
	VisitConditional(expr *ConditionalOperation) interface{}
<<<<<<< HEAD
	VisitPointer(expr *SymbolicPointer) interface{}
	VisitFieldAccess(expr *FieldAccess) interface{}
	VisitFieldAssign(expr *FieldAssign) interface{}

	// funcs
	VisitFunction(fu *Function) interface{}
	VisitFunctionCall(fc *FunctionCall) interface{}
=======
>>>>>>> origin/main
	// TODO: Добавьте методы для других типов выражений по мере необходимости
}
