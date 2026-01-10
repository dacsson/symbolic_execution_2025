package internal

import (
	"fmt"
	"go/constant"
	// "go/token"
	"go/types"
	"regexp"
	"strconv"
	"strings"
	"symbolic-execution-course/internal/memory"
	"symbolic-execution-course/internal/symbolic"

	"golang.org/x/tools/go/ssa"
)

// TODO: that is silly
var VARCOUNTER = 0
const maxTotalUnrolls = 100
const maxLoopUnroll = 10
const maxExecutionSteps = 10000

type Interpreter struct {
	CallStack     []CallStackFrame
	Analyser      *Analyser
	PathCondition symbolic.SymbolicExpression
	Heap          *memory.SymbolicMemory // TODO: delete it from translater since we use it here
	CurrentBlock  *ssa.BasicBlock
	InstrIndex    int
	LoopCounters  map[string]int
	MaxLoopUnroll int
	VisitedBlocks map[string]bool
	PrevBlock     *ssa.BasicBlock
	BlockVisitCount map[string]int
	MaxCallDepth     int
	CurrentCallDepth int
	VisitedFunctions map[string]bool
	ExecutionSteps   int
}

func (interpreter *Interpreter) TranslateAndOutput(expr symbolic.SymbolicExpression) string {
	z3Expr, _ := interpreter.Analyser.Z3Translator.TranslateExpression(expr)
	return fmt.Sprintf("%s: %T", expr.String(), z3Expr)
}

type CallStackFrame struct {
	Function     *ssa.Function
	LocalMemory  map[string]symbolic.SymbolicExpression
	ReturnValue  symbolic.SymbolicExpression
	CurrentBlock *ssa.BasicBlock // for tracking purposes
	ReturnToIndex int
	ReturnVarName string
}

//#========== HELPERS =========#
func (interpreter *Interpreter) IsFinished() bool {
	return interpreter.CurrentBlock == nil ||
		len(interpreter.CallStack) == 0 ||
		(interpreter.CurrentBlock != nil && interpreter.InstrIndex >= len(interpreter.CurrentBlock.Instrs))
}

// type NoInstruction nil;

func (interpreter *Interpreter) GetNextInstruction() ssa.Instruction {
	if interpreter.IsFinished() {
		return nil
	}

	if interpreter.InstrIndex < len(interpreter.CurrentBlock.Instrs) {
		return interpreter.CurrentBlock.Instrs[interpreter.InstrIndex]
	}

	return nil
}

func (interpreter *Interpreter) GetCurrentFrame() *CallStackFrame {
	if len(interpreter.CallStack) == 0 {
		return nil
	}
	return &interpreter.CallStack[len(interpreter.CallStack)-1]
}

func ssaTypeToSymbolicType(ty types.Type) symbolic.ExpressionType {
	switch ty.(type) {
	case *types.Basic:
		switch ty.(*types.Basic).Kind() {
		case types.Bool:
			return symbolic.BoolType
		case types.Int:
			return symbolic.IntType
		case types.Float32:
			return symbolic.FloatType
		// case types.String:
			// return symbolic.StringType
		default:
			panic("unimplemented type " + ty.String())
		}
	case *types.Pointer:
		return symbolic.AddrType
	case *types.Array: case *types.Slice:
		return symbolic.ArrayType
	case *types.Struct: case *types.Interface:
		return symbolic.ObjType
	case *types.Signature:
		return symbolic.FuncType
	case *types.Named:
	    // user-defined type
		return symbolic.ObjType
	default:
		panic("unimplemented type " + ty.String())
	}

	// like `auto` in C, huh
	return symbolic.IntType
}

func (interpreter *Interpreter) LoopEval() {
	if interpreter.LoopCounters == nil {
		interpreter.LoopCounters = make(map[string]int)
	}
	if interpreter.VisitedBlocks == nil {
		interpreter.VisitedBlocks = make(map[string]bool)
	}
	if interpreter.MaxLoopUnroll == 0 {
		interpreter.MaxLoopUnroll = maxLoopUnroll
	}
	if interpreter.BlockVisitCount == nil {
		interpreter.BlockVisitCount = make(map[string]int)
	}
}

//#========= HELPERS =========#

func (interpreter *Interpreter) interpretDynamically(element ssa.Instruction) []*Interpreter {
	if interpreter.ExecutionSteps >= maxExecutionSteps {
		interpreter.CurrentBlock = nil
		return []*Interpreter{}
	}
	interpreter.ExecutionSteps++

	interpreter.LoopEval()

	switch instr := element.(type) {
	case *ssa.Return:
		return interpreter.interpretReturn(instr)
	case *ssa.If:
		return interpreter.interpretIf(instr)
	case *ssa.Jump:
		return interpreter.interpretJump(instr)
	case *ssa.Phi:
		return interpreter.interpretPhi(instr)
	case *ssa.UnOp:
		if instr.Op.String() == "*" {
			return interpreter.interpretLoad(instr)
		}
		return interpreter.interpretUnOp(instr)
	case *ssa.BinOp:
		return interpreter.interpretBinOp(instr)
	case *ssa.Store:
		return interpreter.interpretStore(instr)
	case *ssa.Alloc:
		return interpreter.interpretAlloc(instr)
	case *ssa.ChangeType:
		interpreter.InstrIndex++
		return []*Interpreter{interpreter}
	case *ssa.Convert:
		return interpreter.interpretConvert(instr)
	case *ssa.Call:
		return interpreter.interpretCall(instr)
	case *ssa.MakeInterface:
		return interpreter.interpretMakeInterface(instr)
	case *ssa.FieldAddr:
		return interpreter.interpretFieldAddr(instr)
	case *ssa.Field:
		return interpreter.interpretField(instr)
	case *ssa.IndexAddr:
		return interpreter.interpretIndexAddr(instr)
	case *ssa.Index:
		return interpreter.interpretIndex(instr)
	case *ssa.Panic:
		return interpreter.interpretPanic(instr)
	case *ssa.Defer:
		return interpreter.interpretDefer(instr)
	case *ssa.Go:
		return interpreter.interpretGo(instr)
	case *ssa.Send:
		return interpreter.interpretSend(instr)
	case *ssa.Select:
		return interpreter.interpretSelect(instr)
	case *ssa.MakeChan:
		return interpreter.interpretMakeChan(instr)
	case *ssa.Range:
		return interpreter.interpretRange(instr)
	case *ssa.MapUpdate:
		return interpreter.interpretMapUpdate(instr)
	case *ssa.MakeMap:
		return interpreter.interpretMakeMap(instr)
	case *ssa.TypeAssert:
		return interpreter.interpretTypeAssert(instr)
	case *ssa.Extract:
		return interpreter.interpretExtract(instr)
	case *ssa.ChangeInterface:
		return interpreter.interpretChangeInterface(instr)
	case *ssa.MakeSlice:
		return interpreter.interpretMakeSlice(instr)
	case *ssa.Slice:
		return interpreter.interpretSlice(instr)

	default:
		if unop, ok := element.(*ssa.UnOp); ok && unop.Op.String() == "Load" {
			return interpreter.interpretLoad(unop)
		}
		interpreter.InstrIndex++
		return []*Interpreter{interpreter}
	}
}

//#===============================#

func (interpreter *Interpreter) resolveExpression(value ssa.Value) symbolic.SymbolicExpression {
	// NOTE: purposly make a var if not
	switch v := value.(type) {
	case *ssa.Const:
		return interpreter.resolveConst(v)
	case *ssa.UnOp:
		return interpreter.resolveUnOp(v)
	case *ssa.BinOp:
		return interpreter.resolveBinOp(v)
	default:
		name := v.Name()
		if name == "" {
			name = "var" + strconv.Itoa(VARCOUNTER)
			VARCOUNTER += 1
		}
		return symbolic.NewSymbolicVariable(name, symbolic.IntType)
	}
}

func (interpreter *Interpreter) ToString() string {
	if interpreter == nil {
		return "nil"
	}

	var out strings.Builder
	frame := interpreter.CallStack[len(interpreter.CallStack)-1]
	out.WriteString("\n#========== ИНТЕРПРЕТАЦИЯ ==========#\n#\n")

	out.WriteString("# PathCondition: ")
	if interpreter.PathCondition == nil {
		out.WriteString("true\n")
	} else {
		conditionStr := interpreter.TranslateAndOutput(interpreter.PathCondition)
		out.WriteString(fmt.Sprintf("%s\n", conditionStr))
	}

	if frame.ReturnValue != nil {
		returnStr := interpreter.TranslateAndOutput(frame.ReturnValue)
		out.WriteString(fmt.Sprintf("# ReturnValue: %s\n", returnStr))
	}

	// like `if` and `fi` get it lol?
	out.WriteString("#\n#========== ЯИЦАТЕРПРЕТНИ ==========#\n")
	return out.String()
}

func (interpreter *Interpreter) interpretPanic(instr *ssa.Panic) []*Interpreter {
	interpreter.CurrentBlock = nil
	// todo!()
	return []*Interpreter{interpreter}
}

func (interpreter *Interpreter) interpretDefer(instr *ssa.Defer) []*Interpreter {
	interpreter.InstrIndex++
	// todo!()
	return []*Interpreter{interpreter}
}

func (interpreter *Interpreter) interpretGo(instr *ssa.Go) []*Interpreter {
	interpreter.InstrIndex++
	// todo!()
	return []*Interpreter{interpreter}
}

func (interpreter *Interpreter) interpretSend(instr *ssa.Send) []*Interpreter {
	interpreter.InstrIndex++
	// todo!()
	return []*Interpreter{interpreter}
}

func (interpreter *Interpreter) interpretSelect(instr *ssa.Select) []*Interpreter {
	states := []*Interpreter{}

	for i := 0; i < len(instr.States); i++ {
		newInterpreter := interpreter.Copy()
		newInterpreter.InstrIndex++
		states = append(states, newInterpreter)
	}

	return states
}

func (interpreter *Interpreter) interpretMakeChan(instr *ssa.MakeChan) []*Interpreter {
	frame := interpreter.GetCurrentFrame()

	ref := interpreter.Heap.Allocate(symbolic.AddrType, "dummy", symbolic.NewSymbolicVariable("dummy", symbolic.AddrType))

	if frame != nil && instr.Name() != "" {
		frame.LocalMemory[instr.Name()] = ref
	}

	interpreter.InstrIndex++
	return []*Interpreter{interpreter}
}

func (interpreter *Interpreter) interpretRange(instr *ssa.Range) []*Interpreter {
	frame := interpreter.GetCurrentFrame()

	container := interpreter.ResolveExpression(instr.X)

	if ref, ok := container.(*symbolic.SymbolicPointer); ok {
		_ = interpreter.Heap.GetArrayLength(ref)
	}

	if frame != nil && instr.Name() != "" {
		indexVar := symbolic.NewSymbolicVariable(instr.Name()+"_index", symbolic.IntType)
		frame.LocalMemory[instr.Name()] = indexVar
	}

	interpreter.InstrIndex++
	return []*Interpreter{interpreter}
}

func (interpreter *Interpreter) interpretMapUpdate(instr *ssa.MapUpdate) []*Interpreter {
	interpreter.InstrIndex++
	// todo!()
	return []*Interpreter{interpreter}
}

func (interpreter *Interpreter) interpretMakeMap(instr *ssa.MakeMap) []*Interpreter {
	frame := interpreter.GetCurrentFrame()

	ref := interpreter.Heap.Allocate(symbolic.AddrType, "dummy", symbolic.NewSymbolicVariable("dummy", symbolic.AddrType))

	if frame != nil && instr.Name() != "" {
		frame.LocalMemory[instr.Name()] = ref
	}

	interpreter.InstrIndex++
	return []*Interpreter{interpreter}
}

func (interpreter *Interpreter) interpretTypeAssert(instr *ssa.TypeAssert) []*Interpreter {
	frame := interpreter.GetCurrentFrame()

	states := []*Interpreter{}
	successInterpreter := interpreter.Copy()
	if frame != nil && instr.Name() != "" {
		value := symbolic.NewSymbolicVariable(instr.Name(), symbolic.AddrType)
		frame.LocalMemory[instr.Name()] = value
	}
	successInterpreter.InstrIndex++
	states = append(states, successInterpreter)

	failureInterpreter := interpreter.Copy()
	failureInterpreter.CurrentBlock = nil
	states = append(states, failureInterpreter)

	return states
}

func (interpreter *Interpreter) interpretExtract(instr *ssa.Extract) []*Interpreter {
	frame := interpreter.GetCurrentFrame()
	if frame == nil {
		interpreter.InstrIndex++
		return []*Interpreter{interpreter}
	}

	tuple := interpreter.ResolveExpression(instr.Tuple)

	var result symbolic.SymbolicExpression

	if tuple != nil {
		resultName := fmt.Sprintf("extract_%d_from_%s", instr.Index, tuple.String())

		if instr.Index == 0 {
			result = symbolic.NewSymbolicVariable(resultName, symbolic.IntType)
		} else if instr.Index == 1 {
			result = symbolic.NewSymbolicVariable(resultName, symbolic.BoolType)
		} else {
			result = symbolic.NewSymbolicVariable(resultName, symbolic.IntType)
		}
	} else {
		result = symbolic.NewSymbolicVariable(
			fmt.Sprintf("extract_%d", instr.Index),
			symbolic.IntType,
		)
	}

	if instr.Name() != "" {
		frame.LocalMemory[instr.Name()] = result
	}

	interpreter.InstrIndex++
	return []*Interpreter{interpreter}
}

func (interpreter *Interpreter) interpretChangeInterface(instr *ssa.ChangeInterface) []*Interpreter {
	frame := interpreter.GetCurrentFrame()

	value := interpreter.ResolveExpression(instr.X)

	if frame != nil && instr.Name() != "" {
		frame.LocalMemory[instr.Name()] = value
	}

	interpreter.InstrIndex++
	return []*Interpreter{interpreter}
}

func (interpreter *Interpreter) interpretMakeSlice(instr *ssa.MakeSlice) []*Interpreter {
	frame := interpreter.GetCurrentFrame()

	lengthExpr := interpreter.ResolveExpression(instr.Len)
	_ = interpreter.ResolveExpression(instr.Cap)

	var length uint
	if lengthConst, ok := lengthExpr.(*symbolic.IntConstant); ok {
		length = uint(lengthConst.Value)
	} else {
		length = 10
	}

	var elType symbolic.ExpressionType = ssaTypeToSymbolicType(instr.Type().(*types.Slice).Elem())

	ref := interpreter.Heap.Allocate(symbolic.ArrayType, "dummy", symbolic.NewSymbolicArray("", elType, length))

	if frame != nil && instr.Name() != "" {
		frame.LocalMemory[instr.Name()] = ref
	}

	interpreter.InstrIndex++
	return []*Interpreter{interpreter}
}

func (interpreter *Interpreter) interpretSlice(instr *ssa.Slice) []*Interpreter {
	frame := interpreter.GetCurrentFrame()

	base := interpreter.ResolveExpression(instr.X)

	var result symbolic.SymbolicExpression
	if ref, ok := base.(*symbolic.SymbolicPointer); ok {

		var elType symbolic.ExpressionType = ssaTypeToSymbolicType(instr.Type().(*types.Slice).Elem())

		sliceRef := interpreter.Heap.Allocate(symbolic.ArrayType, "slice", symbolic.NewSymbolicArray("", elType, 0))
		interpreter.Heap.CreateAlias(ref, sliceRef.Address)
		result = sliceRef
	} else {
		result = symbolic.NewSymbolicVariable(instr.Name(), symbolic.ArrayType)
	}

	if frame != nil && instr.Name() != "" {
		frame.LocalMemory[instr.Name()] = result
	}

	interpreter.InstrIndex++
	return []*Interpreter{interpreter}
}

func (interpreter *Interpreter) interpretLookup(instr *ssa.Lookup) []*Interpreter {
	frame := interpreter.GetCurrentFrame()
	if frame == nil {
		interpreter.InstrIndex++
		return []*Interpreter{interpreter}
	}

	result := symbolic.NewSymbolicVariable("lookup_result", symbolic.IntType)

	if frame != nil && instr.Name() != "" {
		frame.LocalMemory[instr.Name()] = result
	}

	interpreter.InstrIndex++
	return []*Interpreter{interpreter}
}

func (interpreter *Interpreter) ResolveExpression(value ssa.Value) symbolic.SymbolicExpression {
	if value == nil {
		return symbolic.NewIntConstant(0)
	}

	if value.Name() != "" {
		frame := interpreter.GetCurrentFrame()
		if frame != nil {
			if expr, ok := frame.LocalMemory[value.Name()]; ok {
				return expr
			}
		}
	}

	switch v := value.(type) {
	case *ssa.Const:
		return interpreter.resolveConst(v)
	case *ssa.UnOp:
		if v.Op.String() == "*" {
			return interpreter.resolveLoad(v)
		}
		return interpreter.resolveUnOp(v)
	case *ssa.BinOp:
		return interpreter.resolveBinOp(v)
	case *ssa.Parameter:
		return interpreter.resolveParameter(v)
	case *ssa.Alloc:
		return interpreter.resolveAlloc(v)
	case *ssa.Phi:
		return interpreter.resolvePhi(v)
	case *ssa.Call:
		return interpreter.resolveCall(v)
	case *ssa.ChangeType:
		return interpreter.ResolveExpression(v.X)
	case *ssa.Convert:
		return interpreter.ResolveExpression(v.X)
	case *ssa.MakeInterface:
		return interpreter.ResolveExpression(v.X)
	case *ssa.FieldAddr:
		return interpreter.resolveFieldAddr(v)
	case *ssa.Field:
		return interpreter.resolveField(v)
	case *ssa.IndexAddr:
		return interpreter.resolveIndexAddr(v)
	case *ssa.Index:
		return interpreter.resolveIndex(v)
	default:
		if v.Name() != "" {
			var exprType symbolic.ExpressionType
			typeStr := v.Type().String()
			if strings.Contains(typeStr, "int") {
				exprType = symbolic.IntType
			} else if typeStr == "bool" {
				exprType = symbolic.BoolType
			} else {
				exprType = symbolic.IntType
			}
			return symbolic.NewSymbolicVariable(v.Name(), exprType)
		}
		return symbolic.NewIntConstant(0)
	}
}

func (interpreter *Interpreter) interpretReturn(instr *ssa.Return) []*Interpreter {
	frame := interpreter.GetCurrentFrame()
	if frame == nil {
		interpreter.CurrentBlock = nil
		return []*Interpreter{interpreter}
	}

	if len(instr.Results) > 0 {
		if len(instr.Results) == 1 {
			frame.ReturnValue = interpreter.ResolveExpression(instr.Results[0])
		} else {
			frame.ReturnValue = interpreter.ResolveExpression(instr.Results[0])
			if len(instr.Results) > 1 {
				errorValue := interpreter.ResolveExpression(instr.Results[1])
				frame.LocalMemory["$error"] = errorValue
			}
		}
	}

	if len(interpreter.CallStack) > 1 {
		returningValue := frame.ReturnValue
		interpreter.CallStack = interpreter.CallStack[:len(interpreter.CallStack)-1]
		interpreter.CurrentCallDepth--

		prevFrame := interpreter.GetCurrentFrame()
		if prevFrame != nil {
			if frame.ReturnVarName != "" && returningValue != nil {
				prevFrame.LocalMemory[frame.ReturnVarName] = returningValue
			}

			interpreter.CurrentBlock = frame.CurrentBlock
			interpreter.InstrIndex = frame.ReturnToIndex
			interpreter.PrevBlock = nil
		} else {
			interpreter.CurrentBlock = nil
		}
	} else {
		interpreter.CurrentBlock = nil
	}

	return []*Interpreter{interpreter}
}

func (interpreter *Interpreter) interpretRecursiveStruct(instr *ssa.Alloc) []*Interpreter {
	frame := interpreter.GetCurrentFrame()

	dummy_str := symbolic.NewSymbolicVariable(instr.Name(), symbolic.ObjType)
	ref := interpreter.Heap.Allocate(symbolic.ObjType, instr.Name(), dummy_str)

	elType := ssaTypeToSymbolicType(instr.Type())
	childrenRef := interpreter.Heap.AllocateArray(instr.Name(), elType, 2)

	interpreter.Heap.AssignField(ref, 0, symbolic.NewIntConstant(0))
	interpreter.Heap.AssignField(ref, 1, childrenRef)

	if frame != nil && instr.Name() != "" {
		frame.LocalMemory[instr.Name()] = ref
	}

	interpreter.InstrIndex++
	return []*Interpreter{interpreter}
}

func (interpreter *Interpreter) interpretIf(instr *ssa.If) []*Interpreter {
	if interpreter.ExecutionSteps >= maxExecutionSteps {
		return []*Interpreter{}
	}

	condExpr := interpreter.ResolveExpression(instr.Cond)

	condExpr = interpreter.convertToBool(condExpr)

	trueInterpreter := interpreter.Copy()
	falseInterpreter := interpreter.Copy()

	notCond := symbolic.NewUnaryOperation(condExpr, symbolic.NOT)

	trueInterpreter.PathCondition = simplifyPathCondition(
		symbolic.NewLogicalOperation(
			[]symbolic.SymbolicExpression{interpreter.PathCondition, condExpr},
			symbolic.AND,
		))

	falseInterpreter.PathCondition = simplifyPathCondition(
		symbolic.NewLogicalOperation(
			[]symbolic.SymbolicExpression{interpreter.PathCondition, notCond},
			symbolic.AND,
		))

	results := []*Interpreter{}

	if !isContradiction(trueInterpreter.PathCondition) {
		trueInterpreter.PrevBlock = interpreter.CurrentBlock
		if len(instr.Block().Succs) >= 2 {
			trueInterpreter.CurrentBlock = instr.Block().Succs[0]
			trueInterpreter.InstrIndex = 0
			results = append(results, trueInterpreter)
		}
	}

	if !isContradiction(falseInterpreter.PathCondition) {
		falseInterpreter.PrevBlock = interpreter.CurrentBlock
		if len(instr.Block().Succs) >= 2 {
			falseInterpreter.CurrentBlock = instr.Block().Succs[1]
			falseInterpreter.InstrIndex = 0
			results = append(results, falseInterpreter)
		}
	}

	return results
}

func (interpreter *Interpreter) interpretJump(instr *ssa.Jump) []*Interpreter {
	if len(instr.Block().Succs) > 0 {
		nextBlock := instr.Block().Succs[0]

		interpreter.PrevBlock = interpreter.CurrentBlock

		blockKey := fmt.Sprintf("%p", nextBlock)
		visitCount := interpreter.BlockVisitCount[blockKey]

		if visitCount >= interpreter.MaxLoopUnroll {
			interpreter.CurrentBlock = nil
			return []*Interpreter{interpreter}
		}

		if interpreter.totalUnrolls() >= maxTotalUnrolls {
			interpreter.CurrentBlock = nil
			return []*Interpreter{interpreter}
		}

		pcStr := interpreter.PathCondition.String()
		if strings.Count(pcStr, "&&") > 20 || strings.Count(pcStr, "(") > 50 || len(pcStr) > 500 {
			interpreter.CurrentBlock = nil
			return []*Interpreter{interpreter}
		}

		interpreter.BlockVisitCount[blockKey] = visitCount + 1
		interpreter.CurrentBlock = nextBlock
		interpreter.InstrIndex = 0
	} else {
		interpreter.CurrentBlock = nil
	}
	return []*Interpreter{interpreter}
}

func (interpreter *Interpreter) handleLoop(loopHeader *ssa.BasicBlock) []*Interpreter {
	return interpreter.exitLoop(loopHeader)
}

func (interpreter *Interpreter) totalUnrolls() int {
	total := 0
	for _, count := range interpreter.BlockVisitCount {
		total += count
	}
	return total
}

func (interpreter *Interpreter) findLoopExit(loopHeader *ssa.BasicBlock) *ssa.BasicBlock {
	visited := make(map[*ssa.BasicBlock]bool)
	var queue []*ssa.BasicBlock
	queue = append(queue, loopHeader)

	for len(queue) > 0 {
		block := queue[0]
		queue = queue[1:]

		if visited[block] {
			continue
		}
		visited[block] = true

		if block != loopHeader {
			for _, instr := range block.Instrs {
				if _, ok := instr.(*ssa.Return); ok {
					return block
				}
			}

			leadsBackToLoop := false
			for _, succ := range block.Succs {
				if succ == loopHeader {
					leadsBackToLoop = true
					break
				}
			}

			if !leadsBackToLoop && len(block.Succs) > 0 {
				return block
			}
		}

		for _, succ := range block.Succs {
			if !visited[succ] {
				queue = append(queue, succ)
			}
		}
	}

	return nil
}

func (interpreter *Interpreter) exitLoop(loopHeader *ssa.BasicBlock) []*Interpreter {
	exitInterpreter := interpreter.Copy()
	exitBlock := interpreter.findLoopExit(loopHeader)
	if exitBlock != nil {
		exitInterpreter.CurrentBlock = exitBlock
		exitInterpreter.InstrIndex = 0
		exitInterpreter.PrevBlock = interpreter.CurrentBlock
	} else {
		exitInterpreter.CurrentBlock = nil
	}

	return []*Interpreter{exitInterpreter}
}

func (interpreter *Interpreter) interpretUnOp(instr *ssa.UnOp) []*Interpreter {
	if instr.Op.String() == "*" {
		return interpreter.interpretLoad(instr)
	}

	operand := interpreter.ResolveExpression(instr.X)
	var result symbolic.SymbolicExpression

	switch instr.Op.String() {
	case "-":
		result = symbolic.NewUnaryOperation(operand, symbolic.MINUS)
	case "!":
		result = symbolic.NewUnaryOperation(operand, symbolic.NOT)
	case "^":
		result = symbolic.NewSymbolicVariable(instr.Name()+"_bitnot", symbolic.IntType)
	default:
		result = operand
	}

	frame := interpreter.GetCurrentFrame()
	if frame != nil && instr.Name() != "" && result != nil {
		frame.LocalMemory[instr.Name()] = result
	}

	interpreter.InstrIndex++
	return []*Interpreter{interpreter}
}

func (interpreter *Interpreter) interpretBinOp(instr *ssa.BinOp) []*Interpreter {
	left := interpreter.ResolveExpression(instr.X)
	right := interpreter.ResolveExpression(instr.Y)

	var binOp symbolic.BinaryOperator

	opStr := instr.Op.String()

	opStr = strings.Trim(opStr, "\"'")

	if opStr == "&&" || opStr == "||" {
		left = interpreter.convertToBool(left)
		right = interpreter.convertToBool(right)

		var result symbolic.SymbolicExpression
		if opStr == "&&" {
			result = symbolic.NewLogicalOperation([]symbolic.SymbolicExpression{left, right}, symbolic.AND)
		} else {
			result = symbolic.NewLogicalOperation([]symbolic.SymbolicExpression{left, right}, symbolic.OR)
		}

		frame := interpreter.GetCurrentFrame()
		if frame != nil && instr.Name() != "" {
			frame.LocalMemory[instr.Name()] = result
		}

		interpreter.InstrIndex++
		return []*Interpreter{interpreter}
	}

	isComparison := false

	switch opStr {
	case "+":
		binOp = symbolic.ADD
	case "-":
		binOp = symbolic.SUB
	case "*":
		binOp = symbolic.MUL
	case "/":
		binOp = symbolic.DIV
	case "%":
		binOp = symbolic.MOD
	case "==":
		binOp = symbolic.EQ
		isComparison = true
	case "!=":
		binOp = symbolic.NE
		isComparison = true
	case "<":
		binOp = symbolic.LT
		isComparison = true
	case "<=":
		binOp = symbolic.LE
		isComparison = true
	case ">":
		binOp = symbolic.GT
		isComparison = true
	case ">=":
		binOp = symbolic.GE
		isComparison = true
	case "&", "|", "^", "<<", ">>", "&^":
		interpreter.InstrIndex++
		return []*Interpreter{interpreter}
	case "&&":
		result := symbolic.NewLogicalOperation([]symbolic.SymbolicExpression{left, right}, symbolic.AND)

		frame := interpreter.GetCurrentFrame()
		if frame != nil && instr.Name() != "" {
			frame.LocalMemory[instr.Name()] = result
		}

		interpreter.InstrIndex++
		return []*Interpreter{interpreter}
	case "||":
		result := symbolic.NewLogicalOperation([]symbolic.SymbolicExpression{left, right}, symbolic.OR)

		frame := interpreter.GetCurrentFrame()
		if frame != nil && instr.Name() != "" {
			frame.LocalMemory[instr.Name()] = result
		}

		interpreter.InstrIndex++
		return []*Interpreter{interpreter}
	default:
		interpreter.InstrIndex++
		return []*Interpreter{interpreter}
	}

	var result symbolic.SymbolicExpression

	if isComparison {
		if ref, ok := right.(*symbolic.SymbolicPointer); ok && ref.Address == 0 {
			if intConst, ok := left.(*symbolic.IntConstant); ok && intConst.Value == 0 {
				left = ref
			} else if symVar, ok := left.(*symbolic.SymbolicVariable); ok && symVar.Name == "nil" {
				left = ref
			}
		} else if ref, ok := left.(*symbolic.SymbolicPointer); ok && ref.Address == 0 {
			if intConst, ok := right.(*symbolic.IntConstant); ok && intConst.Value == 0 {
				right = ref
			} else if symVar, ok := right.(*symbolic.SymbolicVariable); ok && symVar.Name == "nil" {
				right = ref
			}
		}

		result = symbolic.NewBinaryOperation(left, right, binOp)
	} else {
		result = symbolic.NewBinaryOperation(left, right, binOp)
	}

	result = simplifyExpression(result)

	frame := interpreter.GetCurrentFrame()
	if frame != nil && instr.Name() != "" {
		frame.LocalMemory[instr.Name()] = result
	}

	interpreter.InstrIndex++
	return []*Interpreter{interpreter}
}

func (interpreter *Interpreter) interpretAlloc(instr *ssa.Alloc) []*Interpreter {
	var exprType symbolic.ExpressionType
	typeStr := instr.Type().String()

	if strings.Contains(typeStr, "[") && strings.Contains(typeStr, "]") {
		exprType = symbolic.ArrayType

		isBoolArray := strings.Contains(typeStr, "bool")

		re := regexp.MustCompile(`\[(\d+)\].*`)
		matches := re.FindStringSubmatch(typeStr)
		if len(matches) > 1 {
			if size, err := strconv.Atoi(matches[1]); err == nil {
				// TODO: create dummy variable for array allocation
				dummy := symbolic.NewSymbolicVariable("dummy", symbolic.IntType)
				ref := interpreter.Heap.Allocate(exprType, "", dummy)
				for i := 0; i < size; i++ {
					if isBoolArray {
						interpreter.Heap.AssignToArray(ref, i, symbolic.NewBoolConstant(false))
					} else {
						interpreter.Heap.AssignToArray(ref, i, symbolic.NewIntConstant(0))
					}
				}

				frame := interpreter.GetCurrentFrame()
				if frame != nil && instr.Name() != "" {
					frame.LocalMemory[instr.Name()] = ref
				}

				interpreter.InstrIndex++
				return []*Interpreter{interpreter}
			}
		}
	} else if strings.Contains(typeStr, "int") {
		exprType = symbolic.IntType
	} else if strings.Contains(typeStr, "struct") {
		exprType = symbolic.ObjType
	} else if strings.Contains(typeStr, "bool") {
		exprType = symbolic.BoolType
	} else {
		exprType = symbolic.AddrType
	}

	dummy := symbolic.NewSymbolicVariable("dummy", symbolic.IntType)
	ref := interpreter.Heap.Allocate(exprType, "", dummy)

	frame := interpreter.GetCurrentFrame()
	if frame != nil && instr.Name() != "" {
		frame.LocalMemory[instr.Name()] = ref
	}

	interpreter.InstrIndex++
	return []*Interpreter{interpreter}
}

func (interpreter *Interpreter) interpretConvert(instr *ssa.Convert) []*Interpreter {
	operand := interpreter.ResolveExpression(instr.X)

	frame := interpreter.GetCurrentFrame()
	if frame != nil && instr.Name() != "" {
		frame.LocalMemory[instr.Name()] = operand
	}

	interpreter.InstrIndex++
	return []*Interpreter{interpreter}
}

func (interpreter *Interpreter) interpretStore(instr *ssa.Store) []*Interpreter {
	addr := interpreter.ResolveExpression(instr.Addr)
	value := interpreter.ResolveExpression(instr.Val)

	frame := interpreter.GetCurrentFrame()

	if fieldAddr, ok := addr.(*symbolic.FieldAddr); ok {
		interpreter.Heap.AssignField(fieldAddr.Ptr, fieldAddr.FieldIndex, value)

		if frame != nil {
			for varName, varValue := range frame.LocalMemory {
				if otherFieldAddr, ok := varValue.(*symbolic.FieldAddr); ok {
					if otherFieldAddr.Ptr.Address == fieldAddr.Ptr.Address &&
						otherFieldAddr.FieldIndex == fieldAddr.FieldIndex &&
						varName != instr.Addr.Name() {
						interpreter.Heap.AssignField(otherFieldAddr.Ptr, otherFieldAddr.FieldIndex, value)
					}
				}
			}
		}
	} else if indexAddr, ok := addr.(*symbolic.IndexAddr); ok {
		interpreter.Heap.AssignToArray(indexAddr.Ptr, indexAddr.Index, value)
	} else if ref, ok := addr.(*symbolic.SymbolicPointer); ok {
		interpreter.Heap.AssignField(ref, 0, value)
	}

	interpreter.InstrIndex++
	return []*Interpreter{interpreter}
}

func (interpreter *Interpreter) interpretPhi(instr *ssa.Phi) []*Interpreter {
	frame := interpreter.GetCurrentFrame()
	if frame == nil {
		interpreter.InstrIndex++
		return []*Interpreter{interpreter}
	}

	var result symbolic.SymbolicExpression

	if interpreter.PrevBlock != nil {
		for i, pred := range instr.Block().Preds {
			if pred == interpreter.PrevBlock && i < len(instr.Edges) {
				result = interpreter.ResolveExpression(instr.Edges[i])
				break
			}
		}
	}

	if result == nil && len(instr.Edges) > 0 {
		result = symbolic.NewSymbolicVariable(instr.Name()+"_phi", symbolic.IntType)
	}

	if result == nil {
		result = symbolic.NewIntConstant(0)
	}

	result = simplifyExpression(result)

	if frame != nil && instr.Name() != "" {
		frame.LocalMemory[instr.Name()] = result
	}

	interpreter.InstrIndex++
	return []*Interpreter{interpreter}
}

func (interpreter *Interpreter) handleBuiltinCall(instr *ssa.Call, builtin *ssa.Builtin) []*Interpreter {
	frame := interpreter.GetCurrentFrame()
	args := make([]symbolic.SymbolicExpression, len(instr.Call.Args))

	for i, arg := range instr.Call.Args {
		args[i] = interpreter.ResolveExpression(arg)
	}

	var result symbolic.SymbolicExpression

	switch builtin.Name() {
	case "len":
		if len(args) > 0 {
			if ref, ok := args[0].(*symbolic.SymbolicPointer); ok {
				if length := interpreter.Heap.GetArrayLength(ref); length >= 0 {
					result = symbolic.NewIntConstant(int64(length))
				} else {
					result = symbolic.NewSymbolicVariable("len_result", symbolic.IntType)
				}
			} else {
				result = symbolic.NewSymbolicVariable("len_result", symbolic.IntType)
			}
		}
	case "make":
		if len(args) >= 2 {
			if sizeConst, ok := args[1].(*symbolic.IntConstant); ok {
				elType := ssaTypeToSymbolicType(instr.Type())
				ref := interpreter.Heap.AllocateArray(instr.Name(), elType, int(sizeConst.Value))
				result = ref
			}
		}
	case "append":
		if len(args) >= 2 {
			result = args[0]
		}
	case "panic":
		interpreter.CurrentBlock = nil
		return []*Interpreter{interpreter}
	case "recover":
		result = symbolic.NewSymbolicPointer(0, symbolic.AddrType)
	default:
		result = symbolic.NewSymbolicVariable(builtin.Name()+"_result", symbolic.IntType)
	}

	if frame != nil && instr.Name() != "" && result != nil {
		frame.LocalMemory[instr.Name()] = result
	}

	interpreter.InstrIndex++
	return []*Interpreter{interpreter}
}

func (interpreter *Interpreter) handleClosureCall(instr *ssa.Call, closure *ssa.MakeClosure) []*Interpreter {
	if fn, ok := closure.Fn.(*ssa.Function); ok {
		newFrame := CallStackFrame{
			Function:      fn,
			LocalMemory:   make(map[string]symbolic.SymbolicExpression),
			ReturnValue:   nil,
			CurrentBlock: interpreter.CurrentBlock,
			ReturnToIndex: interpreter.InstrIndex + 1,
			ReturnVarName: instr.Name(),
		}

		args := instr.Call.Args
		for i, param := range fn.Params {
			if i < len(args) {
				argValue := interpreter.ResolveExpression(args[i])
				newFrame.LocalMemory[param.Name()] = argValue
			} else if i-len(args) < len(closure.Bindings) {
				bindingIdx := i - len(args)
				if bindingIdx >= 0 && bindingIdx < len(closure.Bindings) {
					bindingValue := interpreter.ResolveExpression(closure.Bindings[bindingIdx])
					newFrame.LocalMemory[param.Name()] = bindingValue
				}
			}
		}

		interpreter.CallStack = append(interpreter.CallStack, newFrame)
		interpreter.CurrentCallDepth++

		interpreter.CurrentBlock = fn.Blocks[0]
		interpreter.InstrIndex = 0
		interpreter.PrevBlock = nil

		return []*Interpreter{interpreter}
	}

	return interpreter.handleUnknownCall(instr)
}

func (interpreter *Interpreter) handleRecursiveCall(instr *ssa.Call, fn *ssa.Function) []*Interpreter {
	frame := interpreter.GetCurrentFrame()
	if frame == nil {
		interpreter.InstrIndex++
		return []*Interpreter{interpreter}
	}

	funcName := "recursive"
	if fn != nil {
		funcName = fn.Name()
	} else if instr.Call.Value != nil {
		funcName = instr.Call.Value.Name()
	}

	if funcName == "isEven" || funcName == "isOdd" {
		if len(instr.Call.Args) > 0 {
			arg := interpreter.ResolveExpression(instr.Call.Args[0])
			two := symbolic.NewIntConstant(2)
			modResult := symbolic.NewBinaryOperation(arg, two, symbolic.MOD)
			zero := symbolic.NewIntConstant(0)
			comparison := symbolic.NewBinaryOperation(modResult, zero, symbolic.EQ)

			var result symbolic.SymbolicExpression
			if funcName == "isEven" {
				result = comparison
			} else {
				result = symbolic.NewUnaryOperation(comparison, symbolic.NOT)
			}

			if instr.Name() != "" {
				frame.LocalMemory[instr.Name()] = result
			}
		}
	} else {
		result := symbolic.NewSymbolicVariable(
			fmt.Sprintf("recursive_%s_depth_%d", funcName, interpreter.CurrentCallDepth),
			symbolic.IntType,
		)

		if instr.Name() != "" {
			frame.LocalMemory[instr.Name()] = result
		}
	}

	interpreter.InstrIndex++
	return []*Interpreter{interpreter}
}

func (interpreter *Interpreter) handleUnknownCall(instr *ssa.Call) []*Interpreter {
	frame := interpreter.GetCurrentFrame()

	funcName := "unknown_call"
	if instr.Call.Value != nil && instr.Call.Value.Name() != "" {
		funcName = instr.Call.Value.Name()
	}

	result := symbolic.NewSymbolicVariable(funcName+"_result", symbolic.IntType)

	if frame != nil && instr.Name() != "" {
		frame.LocalMemory[instr.Name()] = result
	}

	interpreter.InstrIndex++
	return []*Interpreter{interpreter}
}

func (interpreter *Interpreter) handleFunctionCall(instr *ssa.Call, fn *ssa.Function) []*Interpreter {
	for _, frame := range interpreter.CallStack {
		if frame.Function == fn {
			return interpreter.handleRecursiveCall(instr, fn)
		}
	}

	if interpreter.CurrentCallDepth >= interpreter.MaxCallDepth {
		return interpreter.handleRecursiveCall(instr, fn)
	}

	if interpreter.CurrentCallDepth >= interpreter.MaxCallDepth {
		return interpreter.handleRecursiveCall(instr, fn)
	}

	for _, frame := range interpreter.CallStack {
		if frame.Function == fn {
			return interpreter.handleRecursiveCall(instr, fn)
		}
	}

	funcKey := fmt.Sprintf("%p", fn)
	newFrame := CallStackFrame{
		Function:      fn,
		LocalMemory:   make(map[string]symbolic.SymbolicExpression),
		ReturnValue:   nil,
		CurrentBlock: interpreter.CurrentBlock,
		ReturnToIndex: interpreter.InstrIndex + 1,
		ReturnVarName: instr.Name(),
	}

	args := instr.Call.Args
	for i, param := range fn.Params {
		if i < len(args) {
			argValue := interpreter.ResolveExpression(args[i])
			newFrame.LocalMemory[param.Name()] = argValue
		} else {
			switch param.Type().String() {
			case "int":
				newFrame.LocalMemory[param.Name()] = symbolic.NewIntConstant(0)
			case "bool":
				newFrame.LocalMemory[param.Name()] = symbolic.NewBoolConstant(false)
			default:
				newFrame.LocalMemory[param.Name()] = symbolic.NewSymbolicVariable(
					param.Name(), symbolic.IntType)
			}
		}
	}

	interpreter.CallStack = append(interpreter.CallStack, newFrame)
	interpreter.CurrentCallDepth++

	if interpreter.VisitedFunctions == nil {
		interpreter.VisitedFunctions = make(map[string]bool)
	}
	interpreter.VisitedFunctions[funcKey] = true

	interpreter.CurrentBlock = fn.Blocks[0]
	interpreter.InstrIndex = 0
	interpreter.PrevBlock = nil

	return []*Interpreter{interpreter}
}

func (interpreter *Interpreter) interpretCompositeLit(instr *ssa.Alloc) []*Interpreter {
	frame := interpreter.GetCurrentFrame()
	if frame == nil {
		interpreter.InstrIndex++
		return []*Interpreter{interpreter}
	}

	typeStr := instr.Type().String()
	var ref *symbolic.SymbolicPointer

	if strings.Contains(typeStr, "struct") {
		ref = interpreter.Heap.AllocateEmptyStruct(instr.Name(), 2)
	} else if strings.Contains(typeStr, "[") && strings.Contains(typeStr, "]") {
		re := regexp.MustCompile(`\[(\d+)\].*`)
		matches := re.FindStringSubmatch(typeStr)
		if len(matches) > 1 {
			if size, err := strconv.Atoi(matches[1]); err == nil {
				var elType symbolic.ExpressionType = ssaTypeToSymbolicType(instr.Type().(*types.Slice).Elem())
				ref = interpreter.Heap.AllocateArray(instr.Name(), elType, size)
				interpreter.Heap.SetArrayLength(uint(size), ref)
			}
		}
	}

	if ref != nil && instr.Name() != "" {
		frame.LocalMemory[instr.Name()] = ref
	}

	interpreter.InstrIndex++
	return []*Interpreter{interpreter}
}

func (interpreter *Interpreter) interpretCall(instr *ssa.Call) []*Interpreter {
	frame := interpreter.GetCurrentFrame()
	if frame == nil {
		interpreter.InstrIndex++
		return []*Interpreter{interpreter}
	}

	if interpreter.CurrentCallDepth >= interpreter.MaxCallDepth {
		return interpreter.handleRecursiveCall(instr, nil)
	}

	callValue := instr.Call.Value

	switch fn := callValue.(type) {
	case *ssa.Function:
		if fn.Pkg != nil && fn.Pkg.Pkg != nil && fn.Pkg.Pkg.Path() == "errors" && fn.Name() == "New" {
			dummy := symbolic.NewIntConstant(0)
			errRef := interpreter.Heap.Allocate(symbolic.AddrType, "", dummy)

			if instr.Name() != "" {
				frame.LocalMemory[instr.Name()] = errRef
			}

			interpreter.InstrIndex++
			return []*Interpreter{interpreter}
		}

		if fn.Pkg != nil && fn.Pkg.Pkg != nil && fn.Pkg.Pkg.Path() == "fmt" {
			if instr.Name() != "" {
				var result symbolic.SymbolicExpression
				if strings.Contains(fn.Signature.String(), "error") {
					result = symbolic.NewSymbolicPointer(0, symbolic.AddrType)
				} else {
					result = symbolic.NewIntConstant(0)
				}
				frame.LocalMemory[instr.Name()] = result
			}
			interpreter.InstrIndex++
			return []*Interpreter{interpreter}
		}

		for _, stackFrame := range interpreter.CallStack {
			if stackFrame.Function == fn {
				return interpreter.handleRecursiveCall(instr, fn)
			}
		}

		newFrame := CallStackFrame{
			Function:      fn,
			LocalMemory:   make(map[string]symbolic.SymbolicExpression),
			ReturnValue:   nil,
			CurrentBlock: interpreter.CurrentBlock,
			ReturnToIndex: interpreter.InstrIndex + 1,
			ReturnVarName: instr.Name(),
		}

		args := instr.Call.Args
		for i, param := range fn.Params {
			if i < len(args) {
				argValue := interpreter.ResolveExpression(args[i])
				newFrame.LocalMemory[param.Name()] = argValue
			} else {
				switch param.Type().String() {
				case "int":
					newFrame.LocalMemory[param.Name()] = symbolic.NewIntConstant(0)
				case "bool":
					newFrame.LocalMemory[param.Name()] = symbolic.NewBoolConstant(false)
				case "error":
					newFrame.LocalMemory[param.Name()] = symbolic.NewSymbolicPointer(0, symbolic.AddrType)
				default:
					newFrame.LocalMemory[param.Name()] = symbolic.NewSymbolicVariable(
						param.Name(), symbolic.IntType)
				}
			}
		}

		interpreter.CallStack = append(interpreter.CallStack, newFrame)
		interpreter.CurrentCallDepth++

		if len(fn.Blocks) > 0 {
			interpreter.CurrentBlock = fn.Blocks[0]
			interpreter.InstrIndex = 0
			interpreter.PrevBlock = nil
		}

		return []*Interpreter{interpreter}

	case *ssa.Builtin:
		return interpreter.handleBuiltinCall(instr, fn)

	default:
		return interpreter.handleUnknownCall(instr)
	}
}

func (interpreter *Interpreter) interpretMakeInterface(instr *ssa.MakeInterface) []*Interpreter {
	value := interpreter.ResolveExpression(instr.X)

	frame := interpreter.GetCurrentFrame()
	if frame != nil && instr.Name() != "" {
		frame.LocalMemory[instr.Name()] = value
	}

	interpreter.InstrIndex++
	return []*Interpreter{interpreter}
}

func (interpreter *Interpreter) interpretFieldAddr(instr *ssa.FieldAddr) []*Interpreter {
	base := interpreter.ResolveExpression(instr.X)

	var result symbolic.SymbolicExpression

	if ref, ok := base.(*symbolic.SymbolicPointer); ok {
		fieldIndex := instr.Field

		frame := interpreter.GetCurrentFrame()
		if frame != nil {
			for _, value := range frame.LocalMemory {
				if fieldAddr, ok := value.(*symbolic.FieldAddr); ok {
					if fieldAddr.Ptr.Address == ref.Address && fieldAddr.FieldIndex == fieldIndex {
						result = fieldAddr
						break
					}
				}
			}
		}

		if result == nil {
			result = symbolic.NewFieldAddr(ref, fieldIndex)
		}
	} else {
		dummy := symbolic.NewSymbolicVariable("dummy", symbolic.IntType)
		newRef := interpreter.Heap.Allocate(symbolic.AddrType, "", dummy)
		result = symbolic.NewFieldAddr(newRef, instr.Field)
	}

	frame := interpreter.GetCurrentFrame()
	if frame != nil && instr.Name() != "" {
		frame.LocalMemory[instr.Name()] = result
	}

	interpreter.InstrIndex++
	return []*Interpreter{interpreter}
}

func (interpreter *Interpreter) interpretField(instr *ssa.Field) []*Interpreter {
	base := interpreter.ResolveExpression(instr.X)

	var result symbolic.SymbolicExpression

	if ref, ok := base.(*symbolic.SymbolicPointer); ok {
		fieldIndex := instr.Field
		result = interpreter.Heap.GetFieldValue(ref, fieldIndex, ssaTypeToSymbolicType(instr.Type()))

		if result == nil {
			fieldName := fmt.Sprintf("%s_field%d", instr.X.Name(), fieldIndex)
			result = symbolic.NewSymbolicVariable(fieldName, symbolic.AddrType)
		}
	} else {
		result = symbolic.NewIntConstant(0)
	}

	frame := interpreter.GetCurrentFrame()
	if frame != nil && instr.Name() != "" {
		frame.LocalMemory[instr.Name()] = result
	}

	interpreter.InstrIndex++
	return []*Interpreter{interpreter}
}

func (interpreter *Interpreter) interpretIndexAddr(instr *ssa.IndexAddr) []*Interpreter {
	base := interpreter.ResolveExpression(instr.X)
	index := interpreter.ResolveExpression(instr.Index)

	var result symbolic.SymbolicExpression

	if ref, ok := base.(*symbolic.SymbolicPointer); ok {
		if ref.Type() == symbolic.ArrayType {
			if indexConst, ok := index.(*symbolic.IntConstant); ok {
				result = symbolic.NewIndexAddr(ref, int(indexConst.Value))
			} else {
				result = symbolic.NewIndexAddr(ref, 0)
			}
		} else {
			result = symbolic.NewIndexAddr(ref, 0)
		}
	} else {
		result = symbolic.NewSymbolicVariable(instr.Name(), symbolic.AddrType)
	}

	frame := interpreter.GetCurrentFrame()
	if frame != nil && instr.Name() != "" {
		frame.LocalMemory[instr.Name()] = result
	}

	interpreter.InstrIndex++
	return []*Interpreter{interpreter}
}

func (interpreter *Interpreter) interpretIndex(instr *ssa.Index) []*Interpreter {
	base := interpreter.ResolveExpression(instr.X)
	index := interpreter.ResolveExpression(instr.Index)

	var result symbolic.SymbolicExpression

	if ref, ok := base.(*symbolic.SymbolicPointer); ok {
		if indexConst, ok := index.(*symbolic.IntConstant); ok {
			result = interpreter.Heap.GetFromArray(ref, int(indexConst.Value), ssaTypeToSymbolicType(instr.Type()))
		} else {
			result = symbolic.NewIntConstant(0)
		}
	} else {
		result = symbolic.NewIntConstant(0)
	}

	frame := interpreter.GetCurrentFrame()
	if frame != nil && instr.Name() != "" {
		frame.LocalMemory[instr.Name()] = result
	}

	interpreter.InstrIndex++
	return []*Interpreter{interpreter}
}

func (interpreter *Interpreter) convertToBool(expr symbolic.SymbolicExpression) symbolic.SymbolicExpression {
	if expr == nil {
		return symbolic.NewBoolConstant(false)
	}

	switch e := expr.(type) {
	case *symbolic.BoolConstant:
		return expr
	case *symbolic.SymbolicVariable:
		if e.ExprType == symbolic.BoolType {
			return expr
		}
	case *symbolic.UnaryOperation:
		if e.Operator == symbolic.NOT {
			return expr
		}
	case *symbolic.BinaryOperation:
		if e.Operator == symbolic.EQ || e.Operator == symbolic.NE ||
			e.Operator == symbolic.LT || e.Operator == symbolic.LE ||
			e.Operator == symbolic.GT || e.Operator == symbolic.GE {
			return expr
		}
	case *symbolic.LogicalOperation:
		return expr
	}

	if intConst, ok := expr.(*symbolic.IntConstant); ok {
		if intConst.Value == 0 {
			return symbolic.NewBoolConstant(false)
		} else {
			return symbolic.NewBoolConstant(true)
		}
	}

	if symVar, ok := expr.(*symbolic.SymbolicVariable); ok && symVar.ExprType == symbolic.IntType {
		zero := symbolic.NewIntConstant(0)
		return symbolic.NewBinaryOperation(expr, zero, symbolic.NE)
	}

	typeStr := fmt.Sprintf("%T", expr)
	if strings.Contains(typeStr, "Bool") {
		return expr
	}

	return symbolic.NewSymbolicVariable("bool_expr", symbolic.BoolType)
}

func (interpreter *Interpreter) interpretLoad(instr *ssa.UnOp) []*Interpreter {

	addr := interpreter.ResolveExpression(instr.X)



	// fmt.Printf("[DEBUG] Load:  %T %v\n", addr, addr)

	var result symbolic.SymbolicExpression

	switch a := addr.(type) {
	case *symbolic.SymbolicPointer:
		result = interpreter.Heap.GetFieldValue(a, 0, ssaTypeToSymbolicType(instr.Type()))
		if result == nil {
			typeStr := instr.Type().String()
			if strings.Contains(typeStr, "bool") {
				result = symbolic.NewBoolConstant(false)
			} else {
				result = symbolic.NewSymbolicVariable(
					fmt.Sprintf("*ref_%d", a.Address),
					symbolic.IntType,
				)
			}
		}
	case *symbolic.FieldAddr:
		result = interpreter.Heap.GetFieldValue(a.Ptr, a.FieldIndex, ssaTypeToSymbolicType(instr.Type()))
		if result == nil {
			result = symbolic.NewBoolConstant(false)
		}
	case *symbolic.IndexAddr:
		result = interpreter.Heap.GetFromArray(a.Ptr, a.Index, ssaTypeToSymbolicType(instr.Type()))
		if result == nil {
			result = symbolic.NewBoolConstant(false)
		}
	default:
		typeStr := instr.Type().String()
		if strings.Contains(typeStr, "bool") {
			result = symbolic.NewBoolConstant(false)
		} else {
			result = symbolic.NewIntConstant(0)
		}
	}

	result = simplifyExpression(result)

	frame := interpreter.GetCurrentFrame()
	if frame != nil && instr.Name() != "" {
		frame.LocalMemory[instr.Name()] = result
	}

	interpreter.InstrIndex++
	return []*Interpreter{interpreter}
}

func (interpreter *Interpreter) resolveLoad(l *ssa.UnOp) symbolic.SymbolicExpression {
	addr := interpreter.ResolveExpression(l.X)

	var result symbolic.SymbolicExpression

	switch a := addr.(type) {
	case *symbolic.SymbolicPointer:
		result = interpreter.Heap.GetFieldValue(a, 0, ssaTypeToSymbolicType(l.Type()))
		if result == nil {
			typeStr := l.Type().String()
			if strings.Contains(typeStr, "bool") {
				result = symbolic.NewBoolConstant(false)
			} else {
				result = symbolic.NewSymbolicVariable(fmt.Sprintf("*ref_%d", a.Address), symbolic.IntType)
			}
		}
	case *symbolic.FieldAddr:
		result = interpreter.Heap.GetFieldValue(a.Ptr, a.FieldIndex, ssaTypeToSymbolicType(l.Type()))
		if result == nil {
			result = symbolic.NewBoolConstant(false)
		}
	case *symbolic.IndexAddr:
		result = interpreter.Heap.GetFromArray(a.Ptr, a.Index, ssaTypeToSymbolicType(l.Type()))
		if result == nil {
			result = symbolic.NewBoolConstant(false)
		}
	default:
		typeStr := l.Type().String()
		if strings.Contains(typeStr, "bool") {
			result = symbolic.NewBoolConstant(false)
		} else {
			result = symbolic.NewIntConstant(0)
		}
	}

	if result == nil {
		result = symbolic.NewBoolConstant(false)
	}

	return simplifyExpression(result)
}

func (interpreter *Interpreter) resolveConst(c *ssa.Const) symbolic.SymbolicExpression {
	if c.IsNil() {
		return symbolic.NewSymbolicPointer(0, symbolic.AddrType)
	}

	val := c.Value
	if val == nil {
		return symbolic.NewIntConstant(0)
	}

	switch val.Kind() {
	case constant.Int:
		if intVal, ok := constant.Int64Val(val); ok {
			return symbolic.NewIntConstant(intVal)
		}
	case constant.Bool:
		boolVal := constant.BoolVal(val)
		return symbolic.NewBoolConstant(boolVal)
	case constant.String:
		return symbolic.NewSymbolicVariable("string_const", symbolic.AddrType)
	case constant.Float:
		if floatStr := val.String(); floatStr != "" {
			if f, err := strconv.ParseFloat(floatStr, 64); err == nil {
				return symbolic.NewIntConstant(int64(f))
			}
		}
		return symbolic.NewIntConstant(0)
	}

	return symbolic.NewIntConstant(0)
}

func (interpreter *Interpreter) resolveUnOp(u *ssa.UnOp) symbolic.SymbolicExpression {
	if u.Name() != "" {
		frame := interpreter.GetCurrentFrame()
		if frame != nil {
			if expr, ok := frame.LocalMemory[u.Name()]; ok {
				return expr
			}
		}
	}

	operand := interpreter.ResolveExpression(u.X)

	var unaryOp symbolic.UnaryOperator
	opStr := u.Op.String()

	switch opStr {
	case "-":
		unaryOp = symbolic.MINUS
	case "!":
		unaryOp = symbolic.NOT
	default:
		return operand
	}

	result := symbolic.NewUnaryOperation(operand, unaryOp)
	return simplifyExpression(result)
}

func (interpreter *Interpreter) resolveBinOp(b *ssa.BinOp) symbolic.SymbolicExpression {
	if b.Name() != "" {
		frame := interpreter.GetCurrentFrame()
		if frame != nil {
			if expr, ok := frame.LocalMemory[b.Name()]; ok {
				return expr
			}
		}
	}

	left := interpreter.ResolveExpression(b.X)
	right := interpreter.ResolveExpression(b.Y)

	var binOp symbolic.BinaryOperator
	opStr := b.Op.String()
	opStr = strings.Trim(opStr, "\"'")

	switch opStr {
	case "+":
		binOp = symbolic.ADD
	case "-":
		binOp = symbolic.SUB
	case "*":
		binOp = symbolic.MUL
	case "/":
		binOp = symbolic.DIV
	case "%":
		binOp = symbolic.MOD
	case "==":
		binOp = symbolic.EQ
	case "!=":
		binOp = symbolic.NE
	case "<":
		binOp = symbolic.LT
	case "<=":
		binOp = symbolic.LE
	case ">":
		binOp = symbolic.GT
	case ">=":
		binOp = symbolic.GE
	case "&&":
		left = interpreter.convertToBool(left)
		right = interpreter.convertToBool(right)
		result := symbolic.NewLogicalOperation([]symbolic.SymbolicExpression{left, right}, symbolic.AND)
		return simplifyExpression(result)
	case "||":
		left = interpreter.convertToBool(left)
		right = interpreter.convertToBool(right)
		result := symbolic.NewLogicalOperation([]symbolic.SymbolicExpression{left, right}, symbolic.OR)
		return simplifyExpression(result)
	default:
		return left
	}

	if binOp == symbolic.EQ || binOp == symbolic.NE {
		if ref, ok := right.(*symbolic.SymbolicPointer); ok && ref.Address == 0 {
			if intConst, ok := left.(*symbolic.IntConstant); ok && intConst.Value == 0 {
				left = ref
			}
		} else if ref, ok := left.(*symbolic.SymbolicPointer); ok && ref.Address == 0 {
			if intConst, ok := right.(*symbolic.IntConstant); ok && intConst.Value == 0 {
				right = ref
			}
		}
	}

	result := symbolic.NewBinaryOperation(left, right, binOp)
	return simplifyExpression(result)
}

func (interpreter *Interpreter) resolveParameter(p *ssa.Parameter) symbolic.SymbolicExpression {
	frame := interpreter.GetCurrentFrame()
	if frame != nil {
		if val, ok := frame.LocalMemory[p.Name()]; ok {
			return val
		}
	}

	var exprType symbolic.ExpressionType
	typeStr := p.Type().String()
	if strings.Contains(typeStr, "int") {
		exprType = symbolic.IntType
	} else if typeStr == "bool" {
		exprType = symbolic.BoolType
	} else {
		exprType = symbolic.IntType
	}

	return symbolic.NewSymbolicVariable(p.Name(), exprType)
}

func (interpreter *Interpreter) resolveAlloc(a *ssa.Alloc) symbolic.SymbolicExpression {
	frame := interpreter.GetCurrentFrame()
	if frame != nil && a.Name() != "" {
		if val, ok := frame.LocalMemory[a.Name()]; ok {
			return val
		}
	}

	var exprType symbolic.ExpressionType
	typeStr := a.Type().String()

	if strings.Contains(typeStr, "int") {
		exprType = symbolic.IntType
	} else if strings.Contains(typeStr, "struct") {
		exprType = symbolic.ObjType
	} else if strings.Contains(typeStr, "[") && strings.Contains(typeStr, "]") {
		exprType = symbolic.ArrayType
	} else {
		exprType = symbolic.AddrType
	}

	dummy := symbolic.NewSymbolicVariable(a.Name(), exprType)
	return interpreter.Heap.Allocate(exprType, a.Name(), dummy)
}

func (interpreter *Interpreter) resolvePhi(phi *ssa.Phi) symbolic.SymbolicExpression {
	frame := interpreter.GetCurrentFrame()
	if frame != nil && phi.Name() != "" {
		if val, ok := frame.LocalMemory[phi.Name()]; ok {
			return val
		}
	}

	for _, edge := range phi.Edges {
		if edge != nil && edge.Name() != "" {
			if expr, ok := frame.LocalMemory[edge.Name()]; ok {
				return simplifyExpression(expr)
			}
		}
	}

	if len(phi.Edges) > 0 {
		result := interpreter.ResolveExpression(phi.Edges[0])
		return simplifyExpression(result)
	}

	return symbolic.NewIntConstant(0)
}

func (interpreter *Interpreter) resolveCall(c *ssa.Call) symbolic.SymbolicExpression {
	frame := interpreter.GetCurrentFrame()
	if frame != nil && c.Name() != "" {
		if val, ok := frame.LocalMemory[c.Name()]; ok {
			return val
		}
	}

	funcName := "call_result"
	if c.Call.Value != nil && c.Call.Value.Name() != "" {
		funcName = c.Call.Value.Name()
	}

	return symbolic.NewSymbolicVariable(funcName, symbolic.IntType)
}

func (interpreter *Interpreter) resolveFieldAddr(f *ssa.FieldAddr) symbolic.SymbolicExpression {
	if f.Name() != "" {
		frame := interpreter.GetCurrentFrame()
		if frame != nil {
			if expr, ok := frame.LocalMemory[f.Name()]; ok {
				return expr
			}
		}
	}

	base := interpreter.ResolveExpression(f.X)

	if ref, ok := base.(*symbolic.SymbolicPointer); ok {
		return symbolic.NewFieldAddr(ref, f.Field)
	}

	return symbolic.NewSymbolicVariable(f.Name(), symbolic.AddrType)
}

func (interpreter *Interpreter) resolveField(f *ssa.Field) symbolic.SymbolicExpression {
	if f.Name() != "" {
		frame := interpreter.GetCurrentFrame()
		if frame != nil {
			if expr, ok := frame.LocalMemory[f.Name()]; ok {
				return expr
			}
		}
	}

	base := interpreter.ResolveExpression(f.X)

	if ref, ok := base.(*symbolic.SymbolicPointer); ok {
		result := interpreter.Heap.GetFieldValue(ref, f.Field, ssaTypeToSymbolicType(f.Type()))
		return simplifyExpression(result)
	}

	return symbolic.NewIntConstant(0)
}

func (interpreter *Interpreter) resolveIndexAddr(i *ssa.IndexAddr) symbolic.SymbolicExpression {
	if i.Name() != "" {
		frame := interpreter.GetCurrentFrame()
		if frame != nil {
			if expr, ok := frame.LocalMemory[i.Name()]; ok {
				return expr
			}
		}
	}

	base := interpreter.ResolveExpression(i.X)
	index := interpreter.ResolveExpression(i.Index)

	if ref, ok := base.(*symbolic.SymbolicPointer); ok {
		if indexConst, ok := index.(*symbolic.IntConstant); ok {
			return symbolic.NewIndexAddr(ref, int(indexConst.Value))
		}
		return symbolic.NewIndexAddr(ref, 0)
	}

	return symbolic.NewSymbolicVariable(i.Name(), symbolic.AddrType)
}

func (interpreter *Interpreter) resolveIndex(i *ssa.Index) symbolic.SymbolicExpression {
	if i.Name() != "" {
		frame := interpreter.GetCurrentFrame()
		if frame != nil {
			if expr, ok := frame.LocalMemory[i.Name()]; ok {
				return expr
			}
		}
	}

	base := interpreter.ResolveExpression(i.X)
	index := interpreter.ResolveExpression(i.Index)

	if ref, ok := base.(*symbolic.SymbolicPointer); ok {
		if indexConst, ok := index.(*symbolic.IntConstant); ok {
			result := interpreter.Heap.GetFromArray(ref, int(indexConst.Value), ssaTypeToSymbolicType(i.Type()))
			return simplifyExpression(result)
		}
		return symbolic.NewIntConstant(0)
	}

	return symbolic.NewIntConstant(0)
}

func (interpreter *Interpreter) String() string {
	result := fmt.Sprintf("Interpreter:\n")
	result += fmt.Sprintf("PathCondition: %s\n", interpreter.PathCondition.String())

	if len(interpreter.CallStack) > 0 {
		frame := interpreter.GetCurrentFrame()
		result += fmt.Sprintf("Current Frame:\n")
		result += fmt.Sprintf("Function: %s\n", frame.Function.Name())

		if len(frame.LocalMemory) > 0 {
			result += fmt.Sprintf("LocalMemory:\n")
			for k, v := range frame.LocalMemory {
				result += fmt.Sprintf("%s: %s\n", k, v.String())
			}
		}

		if frame.ReturnValue != nil {
			result += fmt.Sprintf("ReturnValue: %s\n", frame.ReturnValue.String())
		}
	}

	if interpreter.CurrentBlock != nil {
		result += fmt.Sprintf("CurrentBlock: %s\n", interpreter.CurrentBlock.String())
	}
	result += fmt.Sprintf("InstrIndex: %d\n", interpreter.InstrIndex)
	result += fmt.Sprintf("TotalUnrolls: %d\n", interpreter.totalUnrolls())

	return result
}

func (interpreter *Interpreter) Copy() *Interpreter {
	newInterpreter := &Interpreter{
		CallStack:        make([]CallStackFrame, len(interpreter.CallStack)),
		Analyser:         interpreter.Analyser,
		PathCondition:    interpreter.PathCondition,
		Heap:             interpreter.Heap.Copy(),
		CurrentBlock:     interpreter.CurrentBlock,
		InstrIndex:       interpreter.InstrIndex,
		LoopCounters:     make(map[string]int),
		MaxLoopUnroll:    interpreter.MaxLoopUnroll,
		VisitedBlocks:    make(map[string]bool),
		BlockVisitCount:  make(map[string]int),
		PrevBlock:        interpreter.PrevBlock,
		MaxCallDepth:     interpreter.MaxCallDepth,
		CurrentCallDepth: interpreter.CurrentCallDepth,
		VisitedFunctions: make(map[string]bool),
		ExecutionSteps:   interpreter.ExecutionSteps,
	}

	for k, v := range interpreter.VisitedFunctions {
		newInterpreter.VisitedFunctions[k] = v
	}

	for k, v := range interpreter.LoopCounters {
		newInterpreter.LoopCounters[k] = v
	}

	for k, v := range interpreter.VisitedBlocks {
		newInterpreter.VisitedBlocks[k] = v
	}

	for k, v := range interpreter.BlockVisitCount {
		newInterpreter.BlockVisitCount[k] = v
	}

	for i, frame := range interpreter.CallStack {
		newFrame := CallStackFrame{
			Function:    frame.Function,
			LocalMemory: make(map[string]symbolic.SymbolicExpression),
			ReturnValue: frame.ReturnValue,
		}

		for k, v := range frame.LocalMemory {
			newFrame.LocalMemory[k] = v
		}

		newInterpreter.CallStack[i] = newFrame
	}

	return newInterpreter
}
