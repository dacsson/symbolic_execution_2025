package internal

import (
	"fmt"
	"go/constant"
	"go/token"
	"strconv"
	"strings"
	"symbolic-execution-course/internal/memory"
	"symbolic-execution-course/internal/symbolic"

	"golang.org/x/tools/go/ssa"
)

// TODO: that is silly
var VARCOUNTER = 0

type Interpreter struct {
	CallStack     []CallStackFrame
	Analyser      *Analyser
	PathCondition symbolic.SymbolicExpression
	Heap          *memory.SymbolicMemory // TODO: delete it from translater since we use it here
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
}

func (interpreter *Interpreter) interpretDynamically(element ssa.Instruction) []Interpreter {
	switch instr := element.(type) {
	case *ssa.UnOp:
		return interpreter.interpretUnOp(instr)
	case *ssa.BinOp:
		return interpreter.interpretBinOp(instr)
	case *ssa.Return:
		return interpreter.interpretReturn(instr)
	case *ssa.If:
		return interpreter.interpretIf(instr)
	case *ssa.Jump:
		return interpreter.interpretJump(instr)
	case *ssa.Phi:
		return interpreter.interpretPhi(instr)
	default:
		panic("unimplemented instruction " + element.String())
	}
}

func (interpreter *Interpreter) interpretUnOp(instr *ssa.UnOp) []Interpreter {
	result := interpreter.resolveExpression(instr)
	interpreter.CallStack[len(interpreter.CallStack)-1].LocalMemory[instr.Name()] = result

	return []Interpreter{*interpreter}
}

func (interpreter *Interpreter) interpretBinOp(instr *ssa.BinOp) []Interpreter {
	result := interpreter.resolveExpression(instr)
	interpreter.CallStack[len(interpreter.CallStack)-1].LocalMemory[instr.Name()] = result

	return []Interpreter{*interpreter}
}

func (interpreter *Interpreter) interpretReturn(instr *ssa.Return) []Interpreter {
	if len(instr.Results) > 0 {
		interpreter.CallStack[len(interpreter.CallStack)-1].ReturnValue = interpreter.resolveExpression(instr.Results[0])
	} else {
		interpreter.CallStack[len(interpreter.CallStack)-1].ReturnValue = nil
	}

	return []Interpreter{*interpreter}
}

func (interpreter *Interpreter) interpretIf(instr *ssa.If) []Interpreter {
	condition := interpreter.resolveExpression(instr.Cond)

	trueState := *interpreter
	trueState.CallStack = make([]CallStackFrame, len(interpreter.CallStack))
	copy(trueState.CallStack, interpreter.CallStack)

	falseState := *interpreter
	falseState.CallStack = make([]CallStackFrame, len(interpreter.CallStack))
	copy(falseState.CallStack, interpreter.CallStack)

	if trueState.PathCondition == nil {
		trueState.PathCondition = condition
	} else {
		trueState.PathCondition = symbolic.NewLogicalOperation(
			[]symbolic.SymbolicExpression{trueState.PathCondition, condition},
			symbolic.AND,
		)
	}

	// TODO: remember to neg
	// if condition {
	negatedCond := symbolic.NewUnaryOperation(condition, symbolic.NOT)
	if falseState.PathCondition == nil {
		falseState.PathCondition = negatedCond
	} else {
		falseState.PathCondition = symbolic.NewLogicalOperation(
			[]symbolic.SymbolicExpression{falseState.PathCondition, negatedCond},
			symbolic.AND,
		)
	}

	trueState.CallStack[len(trueState.CallStack)-1].CurrentBlock = instr.Block().Succs[0]
	falseState.CallStack[len(falseState.CallStack)-1].CurrentBlock = instr.Block().Succs[1]

	return []Interpreter{trueState, falseState}
}

func (interpreter *Interpreter) interpretJump(instr *ssa.Jump) []Interpreter {
	newInterpreter := *interpreter
	newInterpreter.CallStack = make([]CallStackFrame, len(interpreter.CallStack))
	copy(newInterpreter.CallStack, interpreter.CallStack)
	newInterpreter.CallStack[len(newInterpreter.CallStack)-1].CurrentBlock = instr.Block().Succs[0]

	return []Interpreter{newInterpreter}
}

func (interpreter *Interpreter) interpretPhi(instr *ssa.Phi) []Interpreter {
	panic("not implemented")
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

func (interpreter *Interpreter) resolveConst(v *ssa.Const) symbolic.SymbolicExpression {
	val := v.Value
	switch val.Kind() {
	case constant.Bool:
		return symbolic.NewBoolConstant(constant.BoolVal(val))
	case constant.Int:
		return symbolic.NewIntConstant(v.Int64())
	default:
		panic("unimplemented " + v.String())
	}
}

func (interpreter *Interpreter) resolveUnOp(v *ssa.UnOp) symbolic.SymbolicExpression {
	expr := interpreter.resolveExpression(v.X)
	switch v.Op {
	case token.NOT:
		return symbolic.NewUnaryOperation(expr, symbolic.NOT)
	case token.SUB:
		return symbolic.NewUnaryOperation(expr, symbolic.MINUS)
	default:
		panic("unimplemented " + v.String())
	}
}

func (interpreter *Interpreter) resolveBinOp(v *ssa.BinOp) symbolic.SymbolicExpression {
	lhs := interpreter.resolveExpression(v.X)
	rhs := interpreter.resolveExpression(v.Y)
	var op symbolic.BinaryOperator
	switch v.Op {
	case token.ADD:
		op = symbolic.ADD
	case token.SUB:
		op = symbolic.SUB
	case token.MUL:
		op = symbolic.MUL
	case token.QUO:
		op = symbolic.DIV
	case token.REM:
		op = symbolic.MOD
	case token.EQL:
		op = symbolic.EQ
	case token.NEQ:
		op = symbolic.NE
	case token.LSS:
		op = symbolic.LT
	case token.LEQ:
		op = symbolic.LE
	case token.GTR:
		op = symbolic.GT
	case token.GEQ:
		op = symbolic.GE
	default:
		panic("unimplemented " + v.String())
	}

	return symbolic.NewBinaryOperation(lhs, rhs, op)
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
