package internal

import (
	"container/heap"
	"go/types"
	"symbolic-execution-course/internal/memory"
	"symbolic-execution-course/internal/ssa"
	"symbolic-execution-course/internal/symbolic"
	"symbolic-execution-course/internal/translator"
)

type Analyser struct {
	Package      *types.Package
	StatesQueue  PriorityQueue
	PathSelector PathSelector
	Results      []Interpreter
	Z3Translator *translator.Z3Translator
}

func Analyse(source string, functionName string) []Interpreter {
	source = unrollLoops(source)

	builder := ssa.NewBuilder()
	f, err := builder.ParseAndBuildSSA(source, functionName)
	if err != nil || f == nil {
		panic("Error parsing and building SSA: " + functionName + " " + source)
	}

	z3Translator := translator.NewZ3Translator()
	defer z3Translator.Close()
	a := Analyser{
		Package:      f.Pkg.Pkg,
		PathSelector: &BfsPathSelector{},
		StatesQueue:  make(PriorityQueue, 0),
		Results:      make([]Interpreter, 0),
		Z3Translator: z3Translator,
	}

	mem := memory.NewSymbolicMemory()
	initialState := Interpreter{
		CallStack: []CallStackFrame{{
			Function:     f,
			CurrentBlock: f.Blocks[0],
			LocalMemory:  make(map[string]symbolic.SymbolicExpression),
		}},
		Analyser: &a,
		Heap:     mem,
	}
	for _, param := range f.Params {
		initialState.CallStack[0].LocalMemory[param.Name()] = symbolic.NewSymbolicVariable(param.Name(), symbolic.IntType)
	}
	a.StatesQueue.Push(&Item{
		value:    initialState,
		priority: a.PathSelector.CalculatePriority(initialState),
	})

	a.run(initialState)
	return a.Results
}

// NOTE: stop when nothing in q and at end
func (analyzer *Analyser) run(initialState Interpreter) {
	INSTRCOUNTER := 0
	var prevState = initialState
	for analyzer.StatesQueue.Len() > 0 {
		item := analyzer.StatesQueue.Pop().(*Item)
		currentState := item.value
		currentBlock := currentState.CallStack[len(currentState.CallStack)-1].CurrentBlock

		prevBB := prevState.CallStack[len(prevState.CallStack)-1].CurrentBlock
		prevBBIndex := prevBB.Index

		if prevBBIndex != currentBlock.Index {
			INSTRCOUNTER = 0
		}

		if INSTRCOUNTER >= len(currentBlock.Instrs) {
			analyzer.Results = append(analyzer.Results, currentState)
			INSTRCOUNTER = 0
			continue
		}

		nextInstr := currentBlock.Instrs[INSTRCOUNTER]
		newStates := currentState.interpretDynamically(nextInstr)

		for _, newState := range newStates {
			heap.Push(&analyzer.StatesQueue, &Item{
				value:    newState,
				priority: analyzer.PathSelector.CalculatePriority(newState),
			})
		}

		INSTRCOUNTER += 1
		prevState = currentState
	}
}
