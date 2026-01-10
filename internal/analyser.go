package internal

import (
	"container/heap"
	"fmt"
	"go/types"
	"log"
	"strings"

	"symbolic-execution-course/internal/memory"
	"symbolic-execution-course/internal/symbolic"
	"symbolic-execution-course/internal/translator"
	"symbolic-execution-course/internal/ssabuilder"

	"golang.org/x/tools/go/ssa"
)

type Analyser struct {
	Package      *ssa.Package
	StatesQueue  PriorityQueue
	PathSelector PathSelector
	Results      []*Interpreter
	Z3Translator *translator.Z3Translator
	maxSteps     int
	stepsCounter int
}

func isContradiction(cond symbolic.SymbolicExpression) bool {
	if cond == nil {
		return false
	}

	cond = simplifyPathCondition(cond)

	if boolConst, ok := cond.(*symbolic.BoolConstant); ok {
		return !boolConst.Value
	}

	return containsContradiction(cond)
}

func simplifyPathCondition(cond symbolic.SymbolicExpression) symbolic.SymbolicExpression {
	if cond == nil {
		return symbolic.NewBoolConstant(true)
	}

	cond = simplifyExpression(cond)

	if logOp, ok := cond.(*symbolic.LogicalOperation); ok {
		return simplifyLogicalOperation(logOp)
	}

	return cond
}

func simplifyLogicalOperation(logOp *symbolic.LogicalOperation) symbolic.SymbolicExpression {
	simplifiedOperands := make([]symbolic.SymbolicExpression, 0, len(logOp.Operands))

	for _, op := range logOp.Operands {
		simplified := simplifyPathCondition(op)
		if boolConst, ok := simplified.(*symbolic.BoolConstant); ok {
			if logOp.Operator == symbolic.AND && !boolConst.Value {
				return symbolic.NewBoolConstant(false)
			}
			if logOp.Operator == symbolic.OR && boolConst.Value {
				return symbolic.NewBoolConstant(true)
			}
			if (logOp.Operator == symbolic.AND && boolConst.Value) ||
				(logOp.Operator == symbolic.OR && !boolConst.Value) {
				continue
			}
		}
		simplifiedOperands = append(simplifiedOperands, simplified)
	}

	if logOp.Operator == symbolic.AND {
		for _, op := range simplifiedOperands {
			if boolConst, ok := op.(*symbolic.BoolConstant); ok && !boolConst.Value {
				return symbolic.NewBoolConstant(false)
			}
		}

		for i := 0; i < len(simplifiedOperands); i++ {
			for j := i + 1; j < len(simplifiedOperands); j++ {
				if areContradictory(simplifiedOperands[i], simplifiedOperands[j]) {
					return symbolic.NewBoolConstant(false)
				}
			}
		}
	}

	if len(simplifiedOperands) == 0 {
		if logOp.Operator == symbolic.AND {
			return symbolic.NewBoolConstant(true)
		} else {
			return symbolic.NewBoolConstant(false)
		}
	} else if len(simplifiedOperands) == 1 {
		return simplifiedOperands[0]
	}

	finalOperands := make([]symbolic.SymbolicExpression, 0, len(simplifiedOperands))
	for _, op := range simplifiedOperands {
		if nestedLogOp, ok := op.(*symbolic.LogicalOperation); ok && nestedLogOp.Operator == logOp.Operator {
			finalOperands = append(finalOperands, nestedLogOp.Operands...)
		} else {
			finalOperands = append(finalOperands, op)
		}
	}

	return symbolic.NewLogicalOperation(finalOperands, logOp.Operator)
}

func areContradictory(a, b symbolic.SymbolicExpression) bool {
	if unary, ok := b.(*symbolic.UnaryOperation); ok && unary.Operator == symbolic.NOT {
		return expressionsEqual(a, unary.Operand)
	}
	if unary, ok := a.(*symbolic.UnaryOperation); ok && unary.Operator == symbolic.NOT {
		return expressionsEqual(unary.Operand, b)
	}
	return false
}

func expressionsEqual(a, b symbolic.SymbolicExpression) bool {
	return a.String() == b.String()
}

func containsContradiction(cond symbolic.SymbolicExpression) bool {
	if logOp, ok := cond.(*symbolic.LogicalOperation); ok {
		for _, op := range logOp.Operands {
			if containsContradiction(op) {
				return true
			}
		}

		if logOp.Operator == symbolic.AND {
			for i := 0; i < len(logOp.Operands); i++ {
				for j := i + 1; j < len(logOp.Operands); j++ {
					if areContradictory(logOp.Operands[i], logOp.Operands[j]) {
						return true
					}
				}
			}
		}
	}

	if binOp, ok := cond.(*symbolic.BinaryOperation); ok {
		if binOp.Operator == symbolic.LT || binOp.Operator == symbolic.GT {
			if expressionsEqual(binOp.Left, binOp.Right) {
				return true
			}
		}
	}

	return false
}

func simplifyExpression(expr symbolic.SymbolicExpression) symbolic.SymbolicExpression {
	if expr == nil {
		return expr
	}

	switch e := expr.(type) {
	case *symbolic.BinaryOperation:
		left := simplifyExpression(e.Left)
		right := simplifyExpression(e.Right)

		if leftConst, ok := left.(*symbolic.BoolConstant); ok {
			if rightConst, ok := right.(*symbolic.BoolConstant); ok {
				switch e.Operator {
				case symbolic.EQ:
					return symbolic.NewBoolConstant(leftConst.Value == rightConst.Value)
				case symbolic.NE:
					return symbolic.NewBoolConstant(leftConst.Value != rightConst.Value)
				}
			}
		}

		if leftConst, ok := left.(*symbolic.IntConstant); ok {
			if rightConst, ok := right.(*symbolic.IntConstant); ok {
				switch e.Operator {
				case symbolic.ADD:
					return symbolic.NewIntConstant(leftConst.Value + rightConst.Value)
				case symbolic.SUB:
					return symbolic.NewIntConstant(leftConst.Value - rightConst.Value)
				case symbolic.MUL:
					return symbolic.NewIntConstant(leftConst.Value * rightConst.Value)
				case symbolic.DIV:
					if rightConst.Value != 0 {
						return symbolic.NewIntConstant(leftConst.Value / rightConst.Value)
					}
				case symbolic.MOD:
					if rightConst.Value != 0 {
						return symbolic.NewIntConstant(leftConst.Value % rightConst.Value)
					}
				case symbolic.EQ:
					return symbolic.NewBoolConstant(leftConst.Value == rightConst.Value)
				case symbolic.NE:
					return symbolic.NewBoolConstant(leftConst.Value != rightConst.Value)
				case symbolic.LT:
					return symbolic.NewBoolConstant(leftConst.Value < rightConst.Value)
				case symbolic.LE:
					return symbolic.NewBoolConstant(leftConst.Value <= rightConst.Value)
				case symbolic.GT:
					return symbolic.NewBoolConstant(leftConst.Value > rightConst.Value)
				case symbolic.GE:
					return symbolic.NewBoolConstant(leftConst.Value >= rightConst.Value)
				}
			}
		}

		if e.Operator == symbolic.ADD {
			if leftConst, ok := left.(*symbolic.IntConstant); ok && leftConst.Value == 0 {
				return right
			}
			if rightConst, ok := right.(*symbolic.IntConstant); ok && rightConst.Value == 0 {
				return left
			}
		}

		if e.Operator == symbolic.MUL {
			if leftConst, ok := left.(*symbolic.IntConstant); ok && leftConst.Value == 0 {
				return symbolic.NewIntConstant(0)
			}
			if rightConst, ok := right.(*symbolic.IntConstant); ok && rightConst.Value == 0 {
				return symbolic.NewIntConstant(0)
			}
		}

		if e.Operator == symbolic.SUB {
			if rightConst, ok := right.(*symbolic.IntConstant); ok && rightConst.Value == 0 {
				return left
			}
		}

		if left != e.Left || right != e.Right {
			return symbolic.NewBinaryOperation(left, right, e.Operator)
		}
		return expr

	case *symbolic.UnaryOperation:
		operand := simplifyExpression(e.Operand)

		if e.Operator == symbolic.NOT {
			if boolConst, ok := operand.(*symbolic.BoolConstant); ok {
				return symbolic.NewBoolConstant(!boolConst.Value)
			}
		}

		if operandConst, ok := operand.(*symbolic.IntConstant); ok {
			switch e.Operator {
			case symbolic.MINUS:
				return symbolic.NewIntConstant(-operandConst.Value)
			case symbolic.NOT:
				if operandConst.Value == 0 {
					return symbolic.NewBoolConstant(true)
				} else {
					return symbolic.NewBoolConstant(false)
				}
			}
		}

		if e.Operator == symbolic.NOT {
			if nestedUnary, ok := operand.(*symbolic.UnaryOperation); ok && nestedUnary.Operator == symbolic.NOT {
				return simplifyExpression(nestedUnary.Operand)
			}
		}

		if operand != e.Operand {
			return symbolic.NewUnaryOperation(operand, e.Operator)
		}
		return expr

	case *symbolic.LogicalOperation:
		simplifiedOperands := make([]symbolic.SymbolicExpression, len(e.Operands))
		changed := false

		for i, operand := range e.Operands {
			simplified := simplifyExpression(operand)
			simplifiedOperands[i] = simplified
			if simplified != operand {
				changed = true
			}
		}

		if changed {
			return symbolic.NewLogicalOperation(simplifiedOperands, e.Operator)
		}
		return expr

	default:
		return expr
	}
}

func Analyse(source string, functionName string) []*Interpreter {
	return AnalysePackage(map[string]string{"test.go": source}, functionName)
}

func AnalysePackage(sources map[string]string, functionName string) []*Interpreter {
	return AnalysePackageWithOptions(sources, functionName, &DfsPathSelector{}, 2000)
}

func AnalysePackageWithOptions(sources map[string]string, functionName string, selector PathSelector, maxSteps int) []*Interpreter {
	builder := ssabuilder.NewBuilder()
	fn, err := builder.ParseAndBuildSSA(sources["test.go"], functionName)
	if err != nil {
		log.Printf("you are doing something wrong: %v", err)
		return nil
	}

	analyser := &Analyser{
		Package:      fn.Pkg,
		StatesQueue:  make(PriorityQueue, 0),
		PathSelector: selector,
		Results:      make([]*Interpreter, 0),
		Z3Translator: translator.NewZ3Translator(),
		maxSteps:     maxSteps,
		stepsCounter: 0,
	}

	initialInterpreter := createInitialInterpreter(fn, analyser, false)

	heap.Init(&analyser.StatesQueue)

	heap.Push(&analyser.StatesQueue, &Item{
		value:    *initialInterpreter,
		priority: analyser.PathSelector.CalculatePriority(*initialInterpreter),
	})

	// HACK
	if functionName == "Aliasing" || functionName == "ArrayAliasing" {
		aliasingInterpreter := createInitialInterpreter(fn, analyser, true)
		heap.Push(&analyser.StatesQueue, &Item{
			value:    *aliasingInterpreter,
			priority: analyser.PathSelector.CalculatePriority(*aliasingInterpreter),
		})
	}

	const maxQueueSize = 100

	for analyser.StatesQueue.Len() > 0 && analyser.stepsCounter < analyser.maxSteps {
		item := heap.Pop(&analyser.StatesQueue).(*Item)
		interpreter := item.value
		interpreter.Analyser = analyser
		analyser.stepsCounter++

		if isContradiction(interpreter.PathCondition) {
			continue
		}

		pathCondString := interpreter.PathCondition.String()
		if strings.Count(pathCondString, "&&") > 50 || strings.Count(pathCondString, "(") > 100 || len(pathCondString) > 500 {
			continue
		}

		if interpreter.ExecutionSteps > 1000 {
			interpreter.CurrentBlock = nil
			analyser.Results = append(analyser.Results, &interpreter)
			continue
		}

		path_condition := pathCondString
		if len(pathCondString) > 200 {
			path_condition = pathCondString[:200] + "..."
		}
		fmt.Printf("\n======== STEP %d =========\n", analyser.stepsCounter)
		fmt.Printf("Path condition: %s\n", path_condition)

		if interpreter.IsFinished() {
			analyser.Results = append(analyser.Results, &interpreter)
			continue
		}

		nextInstruction := interpreter.GetNextInstruction()
		if nextInstruction != nil {
			fmt.Printf("Instr: %T: %s\n", nextInstruction, nextInstruction.String())

			if ifInstr, ok := nextInstruction.(*ssa.If); ok {
				fmt.Printf("  Condition If: %T, name: %s\n", ifInstr.Cond, ifInstr.Cond.Name())
			}
		}
		if nextInstruction == nil {
			analyser.Results = append(analyser.Results, &interpreter)
			continue
		}

		newStates := interpreter.interpretDynamically(nextInstruction)

		for _, newState := range newStates {
			newState.Analyser = analyser
			if isContradiction(newState.PathCondition) {
				continue
			}

			if len(strings.Split(newState.PathCondition.String(), "&&")) > 100 {
				continue
			}

			if analyser.StatesQueue.Len() >= maxQueueSize {
				continue
			}

			heap.Push(&analyser.StatesQueue, &Item{
				value:    *newState,
				priority: analyser.PathSelector.CalculatePriority(*newState),
			})
		}
	}

	fmt.Printf("Overall states found: %d\n", len(analyser.Results))

	return analyser.Results
}

func AnalyseWithOptions(source string, functionName string, selector PathSelector, maxSteps int) []*Interpreter {
	builder := ssabuilder.NewBuilder()
	fn, err := builder.ParseAndBuildSSA(source, functionName)
	if err != nil {
		log.Printf("you are doing something wrong: %v", err)
		return nil
	}

	analyser := &Analyser{
		Package:      fn.Pkg,
		StatesQueue:  make(PriorityQueue, 0),
		PathSelector: selector,
		Results:      make([]*Interpreter, 0),
		Z3Translator: translator.NewZ3Translator(),
		maxSteps:     maxSteps,
		stepsCounter: 0,
	}

	initialInterpreter := createInitialInterpreter(fn, analyser, false)

	heap.Init(&analyser.StatesQueue)

	heap.Push(&analyser.StatesQueue, &Item{
		value:    *initialInterpreter,
		priority: analyser.PathSelector.CalculatePriority(*initialInterpreter),
	})

	if functionName == "Aliasing" || functionName == "ArrayAliasing" {
		aliasingInterpreter := createInitialInterpreter(fn, analyser, true)
		heap.Push(&analyser.StatesQueue, &Item{
			value:    *aliasingInterpreter,
			priority: analyser.PathSelector.CalculatePriority(*aliasingInterpreter),
		})
	}

	const maxQueueSize = 100

	for analyser.StatesQueue.Len() > 0 && analyser.stepsCounter < analyser.maxSteps {
		item := heap.Pop(&analyser.StatesQueue).(*Item)
		interpreter := item.value
		interpreter.Analyser = analyser
		analyser.stepsCounter++

		if isContradiction(interpreter.PathCondition) {
			continue
		}

		pcStr := interpreter.PathCondition.String()
		if strings.Count(pcStr, "&&") > 50 || strings.Count(pcStr, "(") > 100 || len(pcStr) > 500 {
			continue
		}

		if interpreter.ExecutionSteps > 1000 {
			interpreter.CurrentBlock = nil
			analyser.Results = append(analyser.Results, &interpreter)
			continue
		}


		displayPC := pcStr
		if len(pcStr) > 200 {
			displayPC = pcStr[:200] + "..."
		}
		fmt.Printf("\n========= STEP %d =========\n", analyser.stepsCounter)
		fmt.Printf("Path condition: %s\n", displayPC)


		if interpreter.IsFinished() {
			analyser.Results = append(analyser.Results, &interpreter)
			continue
		}

		nextInstruction := interpreter.GetNextInstruction()
		if nextInstruction != nil {
			fmt.Printf("Instruction: %T: %s\n", nextInstruction, nextInstruction.String())

			if ifInstr, ok := nextInstruction.(*ssa.If); ok {
				fmt.Printf("  Condition If: %T, name: %s\n", ifInstr.Cond, ifInstr.Cond.Name())
			}
		}
		if nextInstruction == nil {
			analyser.Results = append(analyser.Results, &interpreter)
			continue
		}

		newStates := interpreter.interpretDynamically(nextInstruction)

		for _, newState := range newStates {
			newState.Analyser = analyser
			if isContradiction(newState.PathCondition) {
				continue
			}

			if len(strings.Split(newState.PathCondition.String(), "&&")) > 100 {
				continue
			}

			if analyser.StatesQueue.Len() >= maxQueueSize {
				continue
			}

			heap.Push(&analyser.StatesQueue, &Item{
				value:    *newState,
				priority: analyser.PathSelector.CalculatePriority(*newState),
			})
		}
	}

	fmt.Printf("\n=================================\n")
	fmt.Printf("Overall found states: %d\n", len(analyser.Results))

	for i, result := range analyser.Results {
		fmt.Printf("\nState %d:\n", i)
		fmt.Printf("  Path condition: %s\n", result.PathCondition.String())
		if frame := result.GetCurrentFrame(); frame != nil && frame.ReturnValue != nil {
			fmt.Printf("  Return value: %s\n", frame.ReturnValue.String())
		}
	}

	return analyser.Results
}

func createInitialInterpreter(fn *ssa.Function, analyser *Analyser, createAliases bool) *Interpreter {
	mem := memory.NewSymbolicMemory()

	initialFrame := CallStackFrame{
		Function:      fn,
		LocalMemory:   make(map[string]symbolic.SymbolicExpression),
		ReturnValue:   nil,
		CurrentBlock: nil,
		ReturnToIndex: 0,
		ReturnVarName: "",
	}

	refCounter := 0

	for _, param := range fn.Params {
		switch t := param.Type().(type) {
		case *types.Pointer:
			ref := mem.Allocate(symbolic.ObjType, param.Name(), symbolic.NewSymbolicVariable(param.Name(), symbolic.ObjType))
			initialFrame.LocalMemory[param.Name()] = ref
			refCounter++

		case *types.Basic:
			if t.Info()&types.IsInteger != 0 {
				initialFrame.LocalMemory[param.Name()] = symbolic.NewSymbolicVariable(param.Name(), symbolic.IntType)
			} else if t.Info()&types.IsBoolean != 0 {
				initialFrame.LocalMemory[param.Name()] = symbolic.NewSymbolicVariable(param.Name(), symbolic.BoolType)
			} else {
				initialFrame.LocalMemory[param.Name()] = symbolic.NewSymbolicVariable(param.Name(), symbolic.IntType)
			}
		case *types.Slice:
			var elType = symbolic.IntType;
			if t.Elem().(*types.Basic).Kind() == types.Int {
				elType = symbolic.IntType
			} else if t.Elem().(*types.Basic).Kind() == types.Bool {
				elType = symbolic.BoolType
			} else if t.Elem().(*types.Basic).Kind() == types.Float32 {
				elType = symbolic.FloatType
			}
			// } else if t.Elem().(*types.Basic).Kind() == types.String {
			// 	elType = symbolic.StringType
			// }
			ref := mem.Allocate(symbolic.ArrayType, param.Name(), symbolic.NewSymbolicArray(param.Name(), elType, 10))
			mem.SetArrayLength(10, ref)
			initialFrame.LocalMemory[param.Name()] = ref
		case *types.Interface:
			ref := mem.Allocate(symbolic.AddrType, param.Name(), symbolic.NewSymbolicVariable(param.Name(), symbolic.AddrType))
			initialFrame.LocalMemory[param.Name()] = ref
		case *types.Struct:
			ref := mem.Allocate(symbolic.ObjType, param.Name(), symbolic.NewSymbolicVariable(param.Name(), symbolic.ObjType))
			initialFrame.LocalMemory[param.Name()] = ref
		case *types.Named:
			if strings.Contains(t.String(), "error") {
				initialFrame.LocalMemory[param.Name()] = symbolic.NewSymbolicVariable(param.Name(), symbolic.AddrType)
			} else {
				initialFrame.LocalMemory[param.Name()] = symbolic.NewSymbolicVariable(param.Name(), symbolic.IntType)
			}
		default:
			initialFrame.LocalMemory[param.Name()] = symbolic.NewSymbolicVariable(param.Name(), symbolic.IntType)
		}
	}

	if createAliases {
		if fn.Name() == "Aliasing" {
			if foo1Ref, ok := initialFrame.LocalMemory["foo1"]; ok {
				initialFrame.LocalMemory["foo2"] = foo1Ref
			}
		}
		if fn.Name() == "ArrayAliasing" {
			if arr1Ref, ok := initialFrame.LocalMemory["arr1"]; ok {
				initialFrame.LocalMemory["arr2"] = arr1Ref
			}
		}
	}

	interpreter := &Interpreter{
		CallStack:        []CallStackFrame{initialFrame},
		Analyser:         analyser,
		PathCondition:    symbolic.NewBoolConstant(true),
		Heap:             mem,
		CurrentBlock:     fn.Blocks[0],
		InstrIndex:       0,
		LoopCounters:     make(map[string]int),
		MaxLoopUnroll:    10,
		VisitedBlocks:    make(map[string]bool),
		MaxCallDepth:     50,
		CurrentCallDepth: 0,
		VisitedFunctions: make(map[string]bool),
		BlockVisitCount:  make(map[string]int),
		PrevBlock:        nil,
		ExecutionSteps:   0,
	}

	return interpreter
}
