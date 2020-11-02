package main

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

type Program []Toplevel

func (p Program) GenIR(c *Compiler) {
	for _, f := range p {
		f.ToplevelIR(c)
	}
}

type Toplevel interface {
	ToplevelIR(c *Compiler)
}

type Function struct {
	Pub   bool
	Name  string
	Param []VarDecl
	Ret   ConcreteType
	Code  []Statement
}

func (f Function) ToplevelIR(c *Compiler) {
	params := make([]IRParam, len(f.Param))
	for i, param := range f.Param {
		params[i].Name = param.Name
		params[i].Ty = param.Ty.IRTypeName()
	}

	c.StartFunction(f.Pub, f.Name, params, f.Ret.IRTypeName())
	defer c.EndFunction()

	for _, stmt := range f.Code {
		stmt.GenIR(c)
	}
}

type Statement interface {
	Code() string
	GenIR(c *Compiler)
}

type VarDecl struct {
	Name string
	Ty   ConcreteType
}

func (d VarDecl) Code() string {
	return "var " + d.Name + " " + d.Ty.Code()
}
func (d VarDecl) GenIR(c *Compiler) {
	c.DeclareLocal(d.Name, d.Ty)
}
func (d VarDecl) ToplevelIR(c *Compiler) {
	c.DeclareGlobal(d.Name, d.Ty)
}

type ReturnStmt struct {
	Value Expression
}

func (r ReturnStmt) Code() string {
	return "return " + r.Value.Code()
}
func (r ReturnStmt) GenIR(c *Compiler) {
	v := r.Value.GenIR(c)
	c.Insn(0, 0, "ret", v)
}

type ExprStmt struct{ Expression }

func (e ExprStmt) GenIR(c *Compiler) {
	e.Expression.GenIR(c)
}

type Expression interface {
	TypeOf(c *Compiler) Type
	Code() string
	GenIR(c *Compiler) Operand
}

type AssignExpr struct {
	L LValue
	R Expression
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

func (e AssignExpr) Code() string {
	return e.L.Code() + " = " + e.R.Code()
}

func (e AssignExpr) GenIR(c *Compiler) Operand {
	t := e.TypeOf(c).(ConcreteType)
	l := e.L.PtrTo(c)
	r := e.R.GenIR(c)
	// TODO: make extensible
	c.Insn(0, 0, "store"+t.IRTypeName(), r, l)
	return l
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
func (e CallExpr) GenIR(c *Compiler) Operand {
	t, ptr := e.typeOf(c)
	var f Operand
	if ptr {
		f = e.Func.GenIR(c)
	} else {
		f = e.Func.(LValue).PtrTo(c)
	}

	call := CallOperand{f, make([]TypedOperand, len(e.Args))}
	for i, arg := range e.Args {
		// TODO: type-check arguments
		call.Args[i].Ty = arg.TypeOf(c).Concrete().IRTypeName()
		call.Args[i].Op = arg.GenIR(c)
	}

	if t.Ret == nil {
		c.Insn(0, 0, "call", call)
		return nil
	} else {
		v := c.Temporary()
		c.Insn(v, t.Ret.IRBaseTypeName(), "call", call)
		return v
	}
}

type LValue interface {
	Expression
	PtrTo(c *Compiler) Operand
}

func genLValueIR(lv LValue, c *Compiler) Operand {
	ty, ok := lv.TypeOf(c).(NumericType)
	if !ok {
		panic("Attempted load of non-numeric type")
	}

	ptr := lv.PtrTo(c)
	op := "load"
	if ty.IRTypeName() != string(ty.IRBaseTypeName()) {
		if ty.(NumericType).Signed() {
			op += "s"
		} else {
			op += "u"
		}
	}
	op += ty.IRTypeName()

	tmp := c.Temporary()
	c.Insn(tmp, ty.IRBaseTypeName(), op, ptr)
	return tmp
}

type VarExpr string

func (e VarExpr) TypeOf(c *Compiler) Type {
	return c.Variable(string(e)).Ty
}
func (e VarExpr) Code() string {
	return string(e)
}
func (e VarExpr) GenIR(c *Compiler) Operand {
	return genLValueIR(e, c)
}
func (e VarExpr) PtrTo(c *Compiler) Operand {
	return c.Variable(string(e)).Loc
}

type RefExpr struct{ V LValue }

func (e RefExpr) TypeOf(c *Compiler) Type {
	return PointerTo(e.V.TypeOf(c).(ConcreteType))
}
func (e RefExpr) Code() string {
	return "&" + e.V.Code()
}
func (e RefExpr) GenIR(c *Compiler) Operand {
	return e.V.PtrTo(c)
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
func (e DerefExpr) GenIR(c *Compiler) Operand {
	return genLValueIR(e, c)
}
func (e DerefExpr) PtrTo(c *Compiler) Operand {
	return e.V.GenIR(c)
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

func (e BinaryExpr) GenIR(c *Compiler) Operand {
	t := e.TypeOf(c).(NumericType)
	l := e.L.GenIR(c)
	r := e.R.GenIR(c)
	v := c.Temporary()
	c.Insn(v, t.IRBaseTypeName(), e.Op.Instruction(t), l, r)
	return v
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

func (op BinaryOperator) Instruction(typ NumericType) string {
	switch op {
	case BOpAdd:
		return "add"
	case BOpSub:
		return "sub"
	case BOpMul:
		return "mul"
	case BOpDiv:
		if typ.Signed() {
			return "div"
		} else {
			return "udiv"
		}
	case BOpMod:
		if typ.Signed() {
			return "rem"
		} else {
			return "urem"
		}

	case BOpOr:
		return "or"
	case BOpXor:
		return "xor"
	case BOpAnd:
		return "and"
	case BOpShl:
		return "shl"
	case BOpShr:
		if typ.Signed() {
			return "sar"
		} else {
			return "shr"
		}
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
)

type IntegerExpr string

func (_ IntegerExpr) TypeOf(c *Compiler) Type {
	return IntLitType{}
}
func (e IntegerExpr) Code() string {
	return string(e)
}
func (e IntegerExpr) GenIR(c *Compiler) Operand {
	return IRInteger(e)
}

type StringExpr string

func (_ StringExpr) TypeOf(c *Compiler) Type {
	// TODO: immutable types
	return PointerTo(TypeI8)
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
func (e StringExpr) GenIR(c *Compiler) Operand {
	return c.String(string(e))
}
