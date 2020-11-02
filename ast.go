package main

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
	Format() string
	GenStatement(c *Compiler)
}

type VarDecl struct {
	Name string
	Ty   TypeExpr
}

type ReturnStmt struct {
	Value Expression
}

type ExprStmt struct{ Expression }
type Expression interface {
	Format() string
	TypeOf(c *Compiler) Type
	GenExpression(c *Compiler) Operand
}
type LValue interface {
	Expression
	GenPointer(c *Compiler) Operand
}

type VarExpr string
type RefExpr struct{ V LValue }
type DerefExpr struct{ V Expression }

type AssignExpr struct {
	L LValue
	R Expression
}
type MutateExpr struct {
	Op BinaryOperator
	L  LValue
	R  Expression
}

type CallExpr struct {
	Func Expression
	Args []Expression
}

type PrefixExpr struct {
	Op PrefixOperator
	V  Expression
}
type PrefixOperator int

type BinaryExpr struct {
	Op   BinaryOperator
	L, R Expression
}
type BinaryOperator int

type IntegerExpr string
type FloatExpr string
type StringExpr string

type TypeExpr interface {
	Get(c *Compiler) ConcreteType
	Format() string
}

type NamedTypeExpr string
type PointerTypeExpr struct{ To TypeExpr }
type FuncTypeExpr struct {
	Param []TypeExpr
	Ret   TypeExpr
}
