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
	VisitPointer(expr *SymbolicPointer) interface{}
	// TODO: Добавьте методы для других типов выражений по мере необходимости
}
