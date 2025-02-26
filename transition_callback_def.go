package statemachine

import (
	"fmt"
	"reflect"
)

type TransitionCallbackFuncDef struct {
	Label          string                 `json:",omitempty" hcl:"label" hcle:"omitempty"`
	RegisteredFunc string                 `json:",omitempty" hcl:"registered_func" hcle:"omitempty"`
	Func           TransitionCallbackFunc `json:"-" hcle:"omit"`
}

type TransitionCallbackDef struct {
	From        []string                     `json:",omitempty" hcl:"from" hcle:"omitempty"`
	ExceptFrom  []string                     `json:",omitempty" hcl:"except_from" hcle:"omitempty"`
	To          []string                     `json:",omitempty" hcl:"to" hcle:"omitempty"`
	ExceptTo    []string                     `json:",omitempty" hcl:"except_to" hcle:"omitempty"`
	Do          []*TransitionCallbackFuncDef `json:",omitempty" hcl:"do" hcle:"omitempty"`
	ExitToState string                       `json:",omitempty" hcl:"exit_to_state" hcle:"omitempty"`

	validateFor string `json:"-" hcle:"omit"`
}

func (s *TransitionCallbackDef) Matches(from, to string) bool {
	// except from
	for _, exceptState := range s.ExceptFrom {
		if from == exceptState {
			return false
		}
	}

	// except to
	for _, exceptState := range s.ExceptTo {
		if to == exceptState {
			return false
		}
	}

	matchesFrom := len(s.From) == 0
	matchesTo := len(s.To) == 0

	if !matchesFrom {
		for _, state := range s.From {
			if from == state {
				matchesFrom = true
			}
		}
	}

	if !matchesTo {
		for _, state := range s.To {
			if to == state {
				matchesTo = true
			}
		}
	}

	if matchesFrom && matchesTo {
		return true
	}

	return false
}

func (s *TransitionCallbackDef) SetFrom(states ...string) {
	for _, state := range states {
		s.From = append(s.From, state)
	}
}

func (s *TransitionCallbackDef) SetFromAnyExcept(exceptStates ...string) {
	for _, exceptState := range exceptStates {
		s.ExceptFrom = append(s.ExceptFrom, exceptState)
	}
}

func (s *TransitionCallbackDef) SetTo(states ...string) {
	for _, state := range states {
		s.To = append(s.To, state)
	}
}

func (s *TransitionCallbackDef) SetSame() {
	s.To = s.From
}

func (s *TransitionCallbackDef) SetToAnyExcept(exceptStates ...string) {
	for _, exceptState := range exceptStates {
		s.ExceptTo = append(s.ExceptTo, exceptState)
	}
}

func (s *TransitionCallbackDef) SetExitToState(supermachineState string) {
	s.ExitToState = supermachineState
}

func (s *TransitionCallbackDef) AddCallbackFunc(callbackFuncs ...TransitionCallbackFunc) {
	for _, callbackFunc := range callbackFuncs {
		s.assertCallbackKind(callbackFunc)
		s.Do = append(s.Do, &TransitionCallbackFuncDef{Func: callbackFunc})
	}
}

func (s *TransitionCallbackDef) assertCallbackKind(callbackFunc TransitionCallbackFunc) {
	t := reflect.TypeOf(callbackFunc)
	switch t.Kind() {
	case reflect.Func:
		requiredReturns := 0
		switch s.validateFor {
		case "BeforeTransition":
			requiredReturns = 1
		case "AroundTransition":
			requiredReturns = 1
		case "AfterTransition":
			requiredReturns = 0
		}
		returnTypeError := reflect.TypeOf(new(error))
		if t.NumOut() != requiredReturns ||
			(requiredReturns == 1 && reflect.PtrTo(t.Out(0)) != returnTypeError) {
			panic("callback func return type wrong")
		}

		optionalArgs := make(map[reflect.Type]struct{})
		requiredArgs := make(map[reflect.Type]struct{})

		optionalArgs[reflect.TypeOf(new(Machine))] = struct{}{}

		switch s.validateFor {
		case "BeforeTransition":
			optionalArgs[reflect.TypeOf(new(Transition))] = struct{}{}
			optionalArgs[reflect.TypeOf(new(string))] = struct{}{}

		case "AroundTransition":
			optionalArgs[reflect.TypeOf(new(Transition))] = struct{}{}
			optionalArgs[reflect.TypeOf(new(string))] = struct{}{}
			requiredArgs[reflect.TypeOf(new(func()))] = struct{}{}

		case "AfterTransition":
			optionalArgs[reflect.TypeOf(new(Transition))] = struct{}{}
			optionalArgs[reflect.TypeOf(new(string))] = struct{}{}
		}

		// ensure all args are of expected types, whether optional or required
		for i := 0; i < t.NumIn(); i++ {
			argType := t.In(i)
			if _, ok := optionalArgs[reflect.PtrTo(argType)]; ok {
				continue
			}
			if _, ok := requiredArgs[reflect.PtrTo(argType)]; ok {
				continue
			}
			panic(fmt.Sprintf("unexpected argument with type '%s' in %s callback", argType, s.validateFor))
		}

	outer:
		// ensure required args are present
		for requiredArg := range requiredArgs {
			for i := 0; i < t.NumIn(); i++ {
				if requiredArg.Elem() == t.In(i) {
					continue outer
				}
			}

			panic(fmt.Sprintf("missing required arg '%s' in %s callback", requiredArg.Elem().String(), s.validateFor))
		}
		return
	}
	panic("callback must be a compatible func")
}
