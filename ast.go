package main

import (
	"fmt"
)

type Program []Function

func (p Program) GenIR(c *Compiler) {
	for _, f := range p {
		f.GenIR(c)
	}
}

type Function struct {
	Export bool
	Name   string
	Return ConcreteType
	Params []VarDecl
	Code   []Statement
}

type VarDecl struct {
	Name string
	Type ConcreteType
}

func (f Function) GenIR(c *Compiler) {
	params := make([]IRParam, len(f.Params))
	for i, param := range f.Params {
		params[i].Name = param.Name
		params[i].Type = param.Type.IRTypeName()
	}

	c.StartFunction(f.Export, f.Name, params, f.Return.IRTypeName())
	defer c.EndFunction()

	for _, stmt := range f.Code {
		stmt.GenIR(c)
	}
}

type Statement interface {
	Code() string
	GenIR(c *Compiler)
}

type DeclStmt struct {
	Name string
	Type ConcreteType
}

func (d DeclStmt) Code() string {
	return "var " + d.Name + " " + d.Type.Code()
}
func (d DeclStmt) GenIR(c *Compiler) {
	c.NewVariable(d.Name, d.Type)
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

type LValue interface {
	Expression
	PtrTo(c *Compiler) Operand
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

type VarExpr string

func (e VarExpr) TypeOf(c *Compiler) Type {
	return c.Variable(string(e)).Type
}
func (e VarExpr) Code() string {
	return string(e)
}
func (e VarExpr) GenIR(c *Compiler) Operand {
	v := c.Variable(string(e))
	t := c.Temporary()
	// TODO: make extensible
	c.Insn(t, v.Type.IRBaseTypeName(), "load"+v.Type.IRTypeName(), v.Loc)
	return t
}
func (e VarExpr) PtrTo(c *Compiler) Operand {
	return c.Variable(string(e)).Loc
}

type BinaryExpr struct {
	Op   BinaryOperator
	L, R Expression
}

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
	}
	panic("Invalid binary operator")
}

const (
	BOpAdd BinaryOperator = iota
	BOpSub
	BOpMul
	BOpDiv
	// TODO: more
)

type IntegerExpr string

func (_ IntegerExpr) TypeOf(c *Compiler) Type {
	return NumberType{}
}
func (e IntegerExpr) Code() string {
	return string(e)
}
func (e IntegerExpr) GenIR(c *Compiler) Operand {
	return IRInteger(e)
}
