package memory

import (
	"symbolic-execution-course/internal/symbolic"
)

type Memory interface {
	Allocate(tpe symbolic.ExpressionType, structName string, init symbolic.SymbolicExpression) *symbolic.SymbolicPointer

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

	Aliases      map[Id]Id
	AliasesId    Id

	// Map of existing arrays lengths
	ArrLength map[Id]uint
}

func NewSymbolicMemory() *SymbolicMemory {
	return &SymbolicMemory{
		Primitives: make(map[symbolic.SymbolicPointer]symbolic.SymbolicExpression),
		Objects:    make(map[Id]map[Id]symbolic.SymbolicExpression),
		Arrays:     make(map[Id]map[Id]symbolic.SymbolicExpression),
		Aliases:    make(map[Id]Id),
		ArrLength:  make(map[Id]uint),
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
	case symbolic.AddrType:
		mem.AliasesId += 1
		return &symbolic.SymbolicPointer{Address: uint(mem.AliasesId), PointerType: tpe, Expr: symbolic.NewIntConstant(0)};
	default:
		return &symbolic.SymbolicPointer{Address: NonId, PointerType: tpe, Expr: init}
	}
}

func (sm *SymbolicMemory) AllocateArray(name string, elType symbolic.ExpressionType, length int) *symbolic.SymbolicPointer {
	arr := symbolic.NewSymbolicArray(name, elType, uint(length))


	ptr := sm.Allocate(symbolic.ArrayType, name, arr)

	for i := 0; i < length; i++ {
		sm.AssignToArray(ptr, i, symbolic.NewIntConstant(0))
	}

	// sm.ArrayId += 1
	// ptr := &symbolic.SymbolicPointer{Address: uint(sm.ArrayId), PointerType: symbolic.ArrayType, Name: name, Expr: arr}

	// sm.Arrays[ptr.Address] = arr

	sm.SetArrayLength(uint(length), ptr)
	return ptr
}


func (sm *SymbolicMemory) AllocateFullStruct(name string, fields []symbolic.SymbolicExpression) *symbolic.SymbolicPointer {
	structExpr := symbolic.NewSymbolicVariable(name, symbolic.ObjType)

	ptr := sm.Allocate(symbolic.ObjType, name, structExpr)

	for i, field := range fields {
		sm.AssignField(ptr, i, field)
	}

	// sm.Obj[ptr.Address] = structExpr
	return ptr
}

func (sm *SymbolicMemory) AllocateEmptyStruct(name string, fieldsNum int) *symbolic.SymbolicPointer {
	structExpr := symbolic.NewSymbolicVariable(name, symbolic.ObjType)

	ptr := sm.Allocate(symbolic.ObjType, name, structExpr)

	for i := 0; i < fieldsNum; i++ {
		// TODO: this should be an array
		sm.AssignField(ptr, i, symbolic.NewIntConstant(0))
	}

	// sm.Obj[ptr.Address] = structExpr
	return ptr
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

func (mem *SymbolicMemory) SetArrayLength(length uint, ptr *symbolic.SymbolicPointer)  {
	if mem.Arrays[Id(ptr.Address)] == nil {
		mem.Arrays[Id(ptr.Address)] = make(map[Id]symbolic.SymbolicExpression)
	}
	mem.ArrLength[Id(ptr.Address)] = length
}

func (mem *SymbolicMemory) GetFromArray(ptr *symbolic.SymbolicPointer, fieldIdx int, ty symbolic.ExpressionType) symbolic.SymbolicExpression {
	key := mem.Arrays[Id(ptr.Address)][Id(fieldIdx)]
	return symbolic.NewFieldAccess(ptr.Expr, fieldIdx, key, ptr.Name, ty)
}

func (sm *SymbolicMemory) getOriginalID(ptr *symbolic.SymbolicPointer) Id {
	if ptr.Address == 0 {
		return 0
	}
	if addr, exists := sm.Aliases[Id(ptr.Address)]; exists {
		return Id(addr)
	}
	return Id(ptr.Address)
}

func (sm *SymbolicMemory) GetArrayLength(ref *symbolic.SymbolicPointer) uint {
	originalID := sm.getOriginalID(ref)
	if length, exists := sm.ArrLength[originalID]; exists {
		return length
	}
	return 0
}

func (sm *SymbolicMemory) CreateAlias(ptr *symbolic.SymbolicPointer, addr uint) *symbolic.SymbolicPointer {
	ptrAddr := sm.getOriginalID(ptr)
	sm.Aliases[Id(addr)] = ptrAddr
	return &symbolic.SymbolicPointer{Address: addr, PointerType: ptr.PointerType, Expr: ptr.Expr, Name: ptr.Name}
}

func (sm *SymbolicMemory) Copy() *SymbolicMemory {
	newMem := &SymbolicMemory{
		Primitives: make(map[symbolic.SymbolicPointer]symbolic.SymbolicExpression),
		Objects:    make(map[Id]map[Id]symbolic.SymbolicExpression),
		Arrays:     make(map[Id]map[Id]symbolic.SymbolicExpression),
	}
	for id, value := range sm.Primitives {
		newMem.Primitives[id] = value
	}
	for id, fields := range sm.Objects {
		newMem.Objects[id] = make(map[Id]symbolic.SymbolicExpression)
		for fieldId, field := range fields {
			newMem.Objects[id][fieldId] = field
		}
	}
	for id, array := range sm.Arrays {
		newMem.Arrays[id] = make(map[Id]symbolic.SymbolicExpression)
		for index, element := range array {
			newMem.Arrays[id][index] = element
		}
	}
	return newMem
}
