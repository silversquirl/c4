package main

type Program []Toplevel

type Toplevel interface {
	FormattableCode
	GenToplevel(c *Compiler)
}

type NamespaceTL struct {
	Name string
	Body []Toplevel
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
	Extern bool
	Names  []string
	Ty     TypeExpr
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
type TypeAlias struct {
	Name string
	Ty   TypeExpr
}

type Statement interface {
	FormattableCode
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

type BreakStmt struct{}
type ContinueStmt struct{}

type ReturnStmt struct {
	Value Expression
}

type ExprStmt struct{ Expression }
type Expression interface {
	FormattableCode
	TypeOf(c *Compiler) Type
	GenExpression(c *Compiler) Operand
}
type LValue interface {
	Expression
	GenPointer(c *Compiler) Operand
	genPointer(c *Compiler) (Operand, Type)
}

type VarExpr string
type RefExpr struct{ V LValue }
type DerefExpr struct{ V Expression }

type AccessExpr struct {
	L LValue
	R string
}

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

type CastExpr struct {
	V  Expression
	Ty TypeExpr
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

type BooleanExpr struct {
	Op   BooleanOperator
	L, R Expression
}
type BooleanOperator int

type IntegerExpr string
type FloatExpr string
type StringExpr string
type RuneExpr rune

type TypeExpr interface {
	FormattableCode
	Get(c *Compiler) ConcreteType
}

type NamedTypeExpr string
type NamespaceTypeExpr []string
type PointerTypeExpr struct{ To TypeExpr }
type ArrayTypeExpr struct {
	Ty TypeExpr
	N  int
}
type FuncTypeExpr struct {
	Var   bool // true if the function uses C-style varags
	Param []TypeExpr
	Ret   TypeExpr
}
type StructTypeExpr []VarDecl
type UnionTypeExpr []VarDecl
