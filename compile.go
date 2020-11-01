package main

import (
	"fmt"
	"io"
	"strconv"
	"strings"
)

type Compiler struct {
	w    io.Writer
	temp Temporary
	vars map[string]Variable
	strs map[IRString]Global
}

func NewCompiler(w io.Writer) *Compiler {
	return &Compiler{
		w, 0,
		make(map[string]Variable),
		make(map[IRString]Global),
	}
}

func (c *Compiler) Compile(prog Program) {
	prog.GenIR(c)
	c.Finish()
}

func (c *Compiler) Writef(format string, args ...interface{}) {
	fmt.Fprintf(c.w, format, args...)
}

func (c *Compiler) Insn(retVar Temporary, retType rune, opcode string, operands ...Operand) {
	b := &strings.Builder{}
	b.WriteString(opcode)
	for i, operand := range operands {
		if i > 0 {
			b.WriteRune(',')
		}
		b.WriteRune(' ')
		b.WriteString(operand.Operand())
	}

	if retVar.IsZero() {
		c.Writef("\t%s\n", b)
	} else {
		c.Writef("\t%s =%c %s\n", retVar, retType, b)
	}
}

func (c *Compiler) StartFunction(export bool, name string, params []IRParam, retType string) {
	prefix := ""
	if export {
		prefix = "export "
	}

	pbuild := &strings.Builder{}
	for i, param := range params {
		if i > 0 {
			pbuild.WriteString(", ")
		}
		pbuild.WriteString(param.Type)
		pbuild.WriteRune(' ')
		pbuild.WriteString(param.Name)
	}

	c.Writef("%sfunction %s $%s(%s) {\n@start\n", prefix, retType, name, pbuild)
}

func (c *Compiler) EndFunction() {
	c.Writef("}\n")

	// Reset temporaries
	c.temp = 0
}

type IRParam struct {
	Name string
	Type string
}

func (c *Compiler) Temporary() Temporary {
	c.temp++
	return c.temp
}

func (c *Compiler) DeclareGlobal(name string, typ ConcreteType) Variable {
	if _, ok := c.vars[name]; ok {
		panic("Variable already exists")
	}
	v := Variable{Global(name), typ}
	c.vars[name] = v
	return v
}
func (c *Compiler) DeclareLocal(name string, typ ConcreteType) Variable {
	if _, ok := c.vars[name]; ok {
		panic("Variable already exists")
	}
	loc := c.Temporary()
	v := Variable{loc, typ}
	c.vars[name] = v

	m := typ.Metrics()
	op := ""
	switch {
	case m.Align <= 4:
		op = "alloc4"
	case m.Align <= 8:
		op = "alloc8"
	case m.Align <= 16:
		op = "alloc16"
	default:
		panic("Invalid alignment")
	}
	c.Insn(loc, 'l', op, IRInt(m.Size))

	return v
}
func (c *Compiler) Variable(name string) Variable {
	v, ok := c.vars[name]
	if !ok {
		panic("Undefined variable")
	}
	return v
}

func (c *Compiler) String(str string) Global {
	istr := IRString(str)
	g, ok := c.strs[istr]
	if !ok {
		g = Global(fmt.Sprintf("str%d", len(c.strs)))
		c.strs[istr] = g
	}
	return g
}

func (c *Compiler) Finish() {
	// Write all strings
	for str, name := range c.strs {
		c.Writef("data %s = %s\n", name, str)
	}
}

type Operand interface {
	Operand() string
}

type Temporary uint

func (t Temporary) IsZero() bool {
	return t == 0
}
func (t Temporary) Operand() string {
	return fmt.Sprintf("%%t%d", t)
}
func (t Temporary) String() string {
	return t.Operand()
}

type Global string

func (g Global) Operand() string {
	return "$" + string(g)
}
func (g Global) String() string {
	return g.Operand()
}

type IRInteger string

func IRInt(i int) IRInteger {
	return IRInteger(strconv.Itoa(i))
}
func (i IRInteger) Operand() string {
	return string(i)
}

type IRString string

func (s IRString) String() string {
	b := &strings.Builder{}
	b.WriteRune('{')
	inStr := false
	for i, ch := range append([]byte(s), 0) {
		if ' ' <= ch && ch <= '~' { // Printable ASCII range
			if !inStr {
				if i > 0 {
					b.WriteRune(',')
				}
				b.WriteString(` b "`)
				inStr = true
			}
			b.WriteByte(ch)
		} else {
			if inStr {
				b.WriteRune('"')
				inStr = false
			}
			if i > 0 {
				b.WriteRune(',')
			}
			fmt.Fprintf(b, " b %d", ch)
		}
	}
	b.WriteString(" }")
	return b.String()
}

type CallOperand struct {
	Func Operand
	Args []TypedOperand
}
type TypedOperand struct {
	Ty string
	Op Operand
}

func (c CallOperand) Operand() string {
	b := &strings.Builder{}
	b.WriteString(c.Func.Operand())
	b.WriteRune('(')
	for i, arg := range c.Args {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString(arg.Ty)
		b.WriteRune(' ')
		b.WriteString(arg.Op.Operand())
	}
	b.WriteRune(')')
	return b.String()
}

type Variable struct {
	Loc  Operand
	Type ConcreteType
}
