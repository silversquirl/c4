package main

import (
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
)

type Compiler struct {
	w io.Writer

	blk  Block
	temp Temporary

	typs map[string]ConcreteType // Type names
	comp []CompositeLayout       // Composite types
	vars map[string]Variable     // Variable names
	strs []IRString              // String constants
	strM map[string]int          // Map from string to index of entry in strs
}

func NewCompiler(w io.Writer) *Compiler {
	return &Compiler{
		w, 0, 0,
		map[string]ConcreteType{
			"I64": TypeI64,
			"I32": TypeI32,
			"I16": TypeI16,
			"I8":  TypeI8,

			"U64": TypeU64,
			"U32": TypeU32,
			"U16": TypeU16,
			"U8":  TypeU8,

			"F64": TypeF64,
			"F32": TypeF32,

			"Bool": TypeBool,
		},
		nil,
		make(map[string]Variable),
		nil,
		make(map[string]int),
	}
}

func (c *Compiler) Compile(prog Program) (err error) {
	defer func() {
		switch e := recover().(type) {
		case nil:
		case string:
			err = errors.New(e)
		default:
			panic(e)
		}
	}()
	c.compile(prog)
	return
}

func (c *Compiler) compile(prog Program) {
	prog.GenProgram(c)
	c.Finish()
}

func (c *Compiler) Writef(format string, args ...interface{}) {
	fmt.Fprintf(c.w, format, args...)
}

func (c *Compiler) Insn(retVar Temporary, retType byte, opcode string, operands ...Operand) {
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
	ptemps := make([]Temporary, len(params))
	for i, param := range params {
		if i > 0 {
			pbuild.WriteString(", ")
		}
		pbuild.WriteString(param.Ty.IRTypeName(c))
		pbuild.WriteRune(' ')
		ptemps[i] = c.Temporary()
		pbuild.WriteString(ptemps[i].Operand())
	}

	c.Writef("%sfunction %s $%s(%s) {\n@start\n", prefix, retType, name, pbuild)

	// Add args to locals
	for i, param := range params {
		loc := c.Temporary()
		c.vars[param.Name] = Variable{loc, param.Ty}
		if param.Ty.IRBaseTypeName() != 0 {
			// If it's a primitive, we need to alloc and copy
			c.allocLocal(loc, param.Ty)
			c.Insn(0, 0, "store"+param.Ty.IRTypeName(c), ptemps[i], loc)
		}
	}
}

func (c *Compiler) EndFunction() {
	c.Writef("}\n")

	// Reset counters
	c.temp = 0
	c.blk = 0
}

type IRParam struct {
	Name string
	Ty   ConcreteType
}

func (c *Compiler) StartBlock(block Block) {
	c.Writef("%s\n", block)
}
func (c *Compiler) Block() Block {
	c.blk++
	return c.blk
}

func (c *Compiler) Temporary() Temporary {
	c.temp++
	return c.temp
}

func (c *Compiler) AliasType(name string, ty ConcreteType) {
	if _, ok := c.typs[name]; ok {
		panic("Type already exists")
	}
	c.typs[name] = ty
}
func (c *Compiler) DefineType(name string, typ ConcreteType) NamedType {
	ty := NamedType{typ, name}
	c.AliasType(name, ty)
	return ty
}
func (c *Compiler) Type(name string) ConcreteType {
	return c.typs[name]
}

func (c *Compiler) CompositeType(layout CompositeLayout) string {
	ident := layout.Ident()
	for i, layout_ := range c.comp {
		switch strings.Compare(layout_.Ident(), ident) {
		case 0:
			// Return
			return ident
		case 1:
			// Insert
			c.comp = append(c.comp, nil)
			copy(c.comp[i+1:], c.comp[i:])
			c.comp[i] = layout
			return ident
		}
	}
	// Append
	c.comp = append(c.comp, layout)
	return ident
}

func (c *Compiler) DeclareGlobal(name string, typ ConcreteType) Variable {
	if _, ok := c.vars[name]; ok {
		panic("Variable already exists")
	}
	v := Variable{Global(name), typ}
	c.vars[name] = v
	return v
}
func (c *Compiler) allocLocal(loc Temporary, ty ConcreteType) {
	m := ty.Metrics()
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
}
func (c *Compiler) DeclareLocal(name string, ty ConcreteType) Variable {
	if _, ok := c.vars[name]; ok {
		panic("Variable already exists")
	}
	loc := c.Temporary()
	v := Variable{loc, ty}
	c.vars[name] = v

	c.allocLocal(loc, ty)
	ty.GenZero(c, loc)
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
	i, ok := c.strM[str]
	if !ok {
		i = len(c.strs)
		c.strM[str] = i
		c.strs = append(c.strs, IRString(str))
	}
	return Global(fmt.Sprintf("str%d", i))
}

func (c *Compiler) Finish() {
	// Write all strings
	for i, str := range c.strs {
		c.Writef("data $str%d = %s\n", i, str)
	}

	// Write all composite types
	for _, layout := range c.comp {
		layout.GenType(c)
	}
}

type CompositeLayout []CompositeEntry
type CompositeEntry struct {
	Ty string
	N  int
}

func (l CompositeLayout) Ident() string {
	b := &strings.Builder{}
	b.WriteByte(':')
	for _, entry := range l {
		if len(entry.Ty) > 1 {
			// X and Y act as parentheses
			b.WriteByte('X')
			b.WriteString(entry.Ty)
			b.WriteByte('Y')
		} else {
			b.WriteString(entry.Ty)
		}
		if entry.N > 1 {
			fmt.Fprintf(b, "%d", entry.N)
		}
	}
	return b.String()
}

func (l CompositeLayout) GenType(c *Compiler) {
	c.Writef("type %s = { ", l.Ident())
	for i, entry := range l {
		if i > 0 {
			c.Writef(", ")
		}
		c.Writef("%s", entry.Ty)
		if entry.N > 1 {
			c.Writef(" %d", entry.N)
		}
	}
	c.Writef(" }\n")
}

type Operand interface {
	Operand() string
}

type Block uint

func (b Block) Operand() string {
	return fmt.Sprintf("@b%d", b)
}
func (b Block) String() string {
	return b.Operand()
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
	Loc Operand
	Ty  ConcreteType
}
