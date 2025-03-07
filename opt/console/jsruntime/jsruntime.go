package jsruntime

import "github.com/robertkrimen/otto"

type (
	FunctionCall = otto.FunctionCall
)

var NullValue = otto.NullValue

var UndefinedValue = otto.UndefinedValue

var TrueValue = otto.TrueValue

var FalseValue = otto.FalseValue

type Value = otto.Value

type Otto = otto.Otto

var New = otto.New

type Script = otto.Script

var ToValue = otto.ToValue

type Object = otto.Object

type Error = otto.Error
