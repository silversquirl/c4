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
	StmtCode() string
	GenIR(c *Compiler)
}

type ReturnStmt struct {
	Value Expression
}

func (r ReturnStmt) StmtCode() string {
	return "return " + r.Value.ExprCode()
}
func (r ReturnStmt) GenIR(c *Compiler) {
	v := r.Value.GenIR(c)
	c.Insn(0, 0, "ret", v)
}

type ExprStmt struct{ Expression }

func (e ExprStmt) StmtCode() string {
	return e.ExprCode()
}
func (e ExprStmt) GenIR(c *Compiler) {
	e.GenIR(c)
}

type Expression interface {
	TypeOf() Type
	ExprCode() string
	GenIR(c *Compiler) Operand
}

type BinaryExpr struct {
	Op   BinaryOperator
	L, R Expression
}

func (e BinaryExpr) TypeOf() Type {
	ltyp := e.L.TypeOf()
	rtyp := e.R.TypeOf()
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

func (e BinaryExpr) ExprCode() string {
	// TODO: smarter spacing/parenthesizing
	return fmt.Sprintf("(%s %s %s)", e.L.ExprCode(), e.Op.Operator(), e.R.ExprCode())
}

func (e BinaryExpr) GenIR(c *Compiler) Operand {
	t := e.TypeOf().(NumericType)
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

func (_ IntegerExpr) TypeOf() Type {
	return NumberType{}
}
func (e IntegerExpr) ExprCode() string {
	return string(e)
}
func (e IntegerExpr) GenIR(c *Compiler) Operand {
	return IRInteger(e)
}
