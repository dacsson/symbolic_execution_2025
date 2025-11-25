package memory

import (
	"symbolic-execution-course/internal/symbolic"
)

type Memory interface {
	Allocate(tpe symbolic.ExpressionType) *symbolic.SymbolicPointer

	//--- BUILTIN ---
	AssignPrimitive(ptr *symbolic.SymbolicPointer, value symbolic.SymbolicExpression)
	GetPrimitive(ptr *symbolic.SymbolicPointer) symbolic.SymbolicExpression

	//--- STRUCTS ---
	AssignField(ptr *symbolic.SymbolicPointer, fieldIdx int, value symbolic.SymbolicExpression) symbolic.SymbolicExpression
	GetFieldValue(ptr *symbolic.SymbolicPointer, fieldIdx int, ty symbolic.ExpressionType) symbolic.SymbolicExpression

	//--- ARRAYS ---
	AssignToArray(ptr *symbolic.SymbolicPointer, fieldIdx int, value symbolic.SymbolicExpression)
	GetFromArray(ptr *symbolic.SymbolicPointer, fieldIdx int, ty symbolic.ExpressionType) symbolic.SymbolicExpression
}

type Id uint

// TODO: uhm...
const NonId = 0xff

type SymbolicMemory struct {
	Primitives map[symbolic.SymbolicPointer]symbolic.SymbolicExpression

	Objects  map[Id]map[Id]symbolic.SymbolicExpression
	ObjectId Id

	Arrays  map[Id]map[Id]symbolic.SymbolicExpression
	ArrayId Id
}

func NewSymbolicMemory() *SymbolicMemory {
	return &SymbolicMemory{
		Primitives: make(map[symbolic.SymbolicPointer]symbolic.SymbolicExpression),
		Objects:    make(map[Id]map[Id]symbolic.SymbolicExpression),
		Arrays:     make(map[Id]map[Id]symbolic.SymbolicExpression),
	}
}

func (mem *SymbolicMemory) Allocate(tpe symbolic.ExpressionType, structName string, init symbolic.SymbolicExpression) *symbolic.SymbolicPointer {
	switch tpe {
	case symbolic.ObjType:
		mem.ObjectId += 1
		return &symbolic.SymbolicPointer{Address: uint(mem.ObjectId), PointerType: tpe, Name: structName, Expr: init}
	case symbolic.ArrayType:
		mem.ArrayId += 1
		return &symbolic.SymbolicPointer{Address: uint(mem.ArrayId), PointerType: tpe, Name: structName, Expr: init}
	default:
		return &symbolic.SymbolicPointer{Address: NonId, PointerType: tpe, Expr: init}
	}
}

func (mem *SymbolicMemory) AssignPrimitive(ptr *symbolic.SymbolicPointer, value symbolic.SymbolicExpression) {
	mem.Primitives[*ptr] = value
}

func (mem *SymbolicMemory) GetPrimitive(ptr *symbolic.SymbolicPointer) symbolic.SymbolicExpression {
	return mem.Primitives[*ptr]
}

func (mem *SymbolicMemory) AssignField(ptr *symbolic.SymbolicPointer, fieldIdx int, value symbolic.SymbolicExpression) symbolic.SymbolicExpression {
	if mem.Objects[Id(ptr.Address)] == nil {
		mem.Objects[Id(ptr.Address)] = make(map[Id]symbolic.SymbolicExpression)
	}
	res := symbolic.NewFieldAssign(ptr.Expr, fieldIdx, value, ptr.Name)
	mem.Objects[Id(ptr.Address)][Id(fieldIdx)] = ptr.Expr
	ptr.Expr = res

	return res
}

func (mem *SymbolicMemory) GetFieldValue(ptr *symbolic.SymbolicPointer, fieldIdx int, ty symbolic.ExpressionType) symbolic.SymbolicExpression {
	key := mem.Objects[Id(ptr.Address)][Id(fieldIdx)]
	return symbolic.NewFieldAccess(ptr.Expr, fieldIdx, key, ptr.Name, ty)
}

func (mem *SymbolicMemory) AssignToArray(ptr *symbolic.SymbolicPointer, index int, value symbolic.SymbolicExpression) symbolic.SymbolicExpression {
	if mem.Arrays[Id(ptr.Address)] == nil {
		mem.Arrays[Id(ptr.Address)] = make(map[Id]symbolic.SymbolicExpression)
	}
	res := symbolic.NewFieldAssign(ptr.Expr, index, value, ptr.Name)
	mem.Arrays[Id(ptr.Address)][Id(index)] = ptr.Expr
	ptr.Expr = res

	return res
}

func (mem *SymbolicMemory) GetFromArray(ptr *symbolic.SymbolicPointer, fieldIdx int, ty symbolic.ExpressionType) symbolic.SymbolicExpression {
	key := mem.Arrays[Id(ptr.Address)][Id(fieldIdx)]
	return symbolic.NewFieldAccess(ptr.Expr, fieldIdx, key, ptr.Name, ty)
}
