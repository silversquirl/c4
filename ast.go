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
	Body  []Statement
}

type VarDecl struct {
	Name string
	Ty   TypeExpr
}
type VarsDecl struct {
	Names []string
	Ty    TypeExpr
}

func (d VarsDecl) Decls() []VarDecl {
	ds := make([]VarDecl, len(d.Names))
	for i, name := range d.Names {
		ds[i] = VarDecl{name, d.Ty}
	}
	return ds
}

type TypeDef struct {
	Name string
	Ty   TypeExpr
}

type Statement interface {
	Format() string
	GenStatement(c *Compiler)
}

type IfStmt struct {
	Cond       Expression
	Then, Else []Statement
}
type ForStmt struct {
	Init       Statement
	Cond, Step Expression
	Body       []Statement
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
	Var   bool // true if the function uses C-style varags
	Param []TypeExpr
	Ret   TypeExpr
}
