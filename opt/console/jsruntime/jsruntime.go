package jsruntime

import "github.com/robertkrimen/otto"

type (
	FunctionCall = otto.FunctionCall
	Value        = otto.Value
	Otto         = otto.Otto
	Script       = otto.Script
	Object       = otto.Object
	Error        = otto.Error
)

var (
	New            = otto.New
	NullValue      = otto.NullValue
	UndefinedValue = otto.UndefinedValue
	TrueValue      = otto.TrueValue
	FalseValue     = otto.FalseValue
	ToValue        = otto.ToValue
)
