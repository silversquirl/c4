package main

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

type Program []Toplevel

type Toplevel interface {
	GenToplevel(c *Compiler)
}

type Function struct {
	Pub   bool
	Name  string
	Param []VarDecl
	Ret   TypeExpr
	Code  []Statement
}

type Statement interface {
	Code() string
	GenStatement(c *Compiler)
}

type VarDecl struct {
	Name string
	Ty   TypeExpr
}

func (d VarDecl) Code() string {
	return "var " + d.Name + " " + d.Ty.Code()
}

type ReturnStmt struct {
	Value Expression
}

func (r ReturnStmt) Code() string {
	return "return " + r.Value.Code()
}

type ExprStmt struct{ Expression }

type Expression interface {
	Code() string
	TypeOf(c *Compiler) Type
	GenExpression(c *Compiler) Operand
}

type AssignExpr struct {
	L LValue
	R Expression
}

func (e AssignExpr) Code() string {
	return e.L.Code() + " = " + e.R.Code()
}
func (e AssignExpr) TypeOf(c *Compiler) Type {
	ltyp, ok := e.L.TypeOf(c).(ConcreteType)
	if !ok {
		panic("Lvalue of non-concrete type")
	}
	rtyp := e.R.TypeOf(c)
	if !Compatible(ltyp, rtyp) {
		panic("Operands of assignment are incompatible")
	}
	return ltyp
}

type CallExpr struct {
	Func Expression
	Args []Expression
}

func (e CallExpr) typeOf(c *Compiler) (t FuncType, ptr bool) {
	switch t := e.Func.TypeOf(c).(type) {
	case FuncType:
		return t, false
	case PointerType:
		return t.To.(FuncType), true
	}
	panic("Invalid function type")
}
func (e CallExpr) TypeOf(c *Compiler) Type {
	t, _ := e.typeOf(c)
	return t.Ret
}
func (e CallExpr) Code() string {
	args := make([]string, len(e.Args))
	for i, arg := range e.Args {
		args[i] = arg.Code()
	}
	return e.Func.Code() + "(" + strings.Join(args, ", ") + ")"
}

type LValue interface {
	Expression
	GenPointer(c *Compiler) Operand
}

type VarExpr string

func (e VarExpr) TypeOf(c *Compiler) Type {
	return c.Variable(string(e)).Ty
}
func (e VarExpr) Code() string {
	return string(e)
}

type RefExpr struct{ V LValue }

func (e RefExpr) TypeOf(c *Compiler) Type {
	return PointerType{e.V.TypeOf(c).(ConcreteType)}
}
func (e RefExpr) Code() string {
	return "&" + e.V.Code()
}

type DerefExpr struct{ V Expression }

func (e DerefExpr) TypeOf(c *Compiler) Type {
	if t, ok := e.V.TypeOf(c).(PointerType); ok {
		return t.To
	} else {
		panic("Dereference of non-pointer type")
	}
}
func (e DerefExpr) Code() string {
	return "[" + e.V.Code() + "]"
}

type BinaryExpr struct {
	Op   BinaryOperator
	L, R Expression
}

// FIXME: all operators other than add, sub, div and mul require integer types
// FIXME: lsh and rsh require their second argument to be an I32 or smaller
func (e BinaryExpr) TypeOf(c *Compiler) Type {
	ltyp := e.L.TypeOf(c)
	rtyp := e.R.TypeOf(c)
	if !Compatible(ltyp, rtyp) {
		panic("Operands of binary expression are incompatible")
	}
	ctyp := ltyp.Concrete()
	if !ltyp.IsConcrete() && rtyp.IsConcrete() {
		ctyp = rtyp.Concrete()
	}
	typ, ok := ctyp.(NumericType)
	if !ok {
		panic("Operand of binary expression is of non-numeric type")
	}
	return typ
}

func (e BinaryExpr) Code() string {
	// TODO: smarter spacing/parenthesizing
	return fmt.Sprintf("(%s %s %s)", e.L.Code(), e.Op.Operator(), e.R.Code())
}

type BinaryOperator int

func (op BinaryOperator) Operator() string {
	switch op {
	case BOpAdd:
		return "+"
	case BOpSub:
		return "-"
	case BOpMul:
		return "*"
	case BOpDiv:
		return "/"
	case BOpMod:
		return "%"

	case BOpOr:
		return "|"
	case BOpXor:
		return "^"
	case BOpAnd:
		return "&"
	case BOpShl:
		return "<<"
	case BOpShr:
		return ">>"
	}
	panic("Invalid binary operator")
}

const (
	BOpAdd BinaryOperator = iota
	BOpSub
	BOpMul
	BOpDiv
	BOpMod

	BOpOr
	BOpXor
	BOpAnd
	BOpShl
	BOpShr

	BinaryOperatorMax
)

type IntegerExpr string

func (_ IntegerExpr) TypeOf(c *Compiler) Type {
	return IntLitType{}
}
func (e IntegerExpr) Code() string {
	return string(e)
}

type FloatExpr string

func (_ FloatExpr) TypeOf(c *Compiler) Type {
	return FloatLitType{}
}
func (e FloatExpr) Code() string {
	return string(e)
}

type StringExpr string

func (_ StringExpr) TypeOf(c *Compiler) Type {
	// TODO: immutable types
	return PointerType{TypeI8}
}
func (e StringExpr) Code() string {
	b := &strings.Builder{}
	b.WriteRune('"')
	str := []byte(e)
	for len(str) > 0 {
		r, size := utf8.DecodeRune(str)
		if r == utf8.RuneError {
			fmt.Fprintf(b, `\x%02x`, str[0])
		} else if r == '"' {
			b.WriteString(`\"`)
		} else if ' ' <= r && r <= '~' { // Printable ASCII range
			b.WriteRune(r)
		} else if r <= 0x7F {
			fmt.Fprintf(b, `\x%02x`, r)
		} else if r <= 0xFFFF {
			fmt.Fprintf(b, `\u%04x`, r)
		} else {
			fmt.Fprintf(b, `\U%08x`, r)
		}
		str = str[size:]
	}
	b.WriteRune('"')
	return b.String()
}

type TypeExpr interface {
	Get(c *Compiler) ConcreteType
	Code() string
}

type NamedTypeExpr string
type PointerTypeExpr struct{ To TypeExpr }
type FuncTypeExpr struct {
	Param []TypeExpr
	Ret   TypeExpr
}

func (name NamedTypeExpr) Get(c *Compiler) ConcreteType {
	return c.Type(string(name))
}
func (ptr PointerTypeExpr) Get(c *Compiler) ConcreteType {
	return PointerType{ptr.To.Get(c)}
}
func (fun FuncTypeExpr) Get(c *Compiler) ConcreteType {
	params := make([]ConcreteType, len(fun.Param))
	for i, param := range fun.Param {
		params[i] = param.Get(c)
	}
	var ret ConcreteType
	if fun.Ret != nil {
		ret = fun.Ret.Get(c)
	}
	return FuncType{params, ret}
}

func (name NamedTypeExpr) Code() string {
	return string(name)
}
func (ptr PointerTypeExpr) Code() string {
	return "[" + ptr.To.Code() + "]"
}
func (fun FuncTypeExpr) Code() string {
	params := make([]string, len(fun.Param))
	for i, param := range fun.Param {
		params[i] = param.Code()
	}
	var ret string
	if fun.Ret != nil {
		ret = " " + fun.Ret.Code()
	}
	return "fn(" + strings.Join(params, ", ") + ")" + ret
}
