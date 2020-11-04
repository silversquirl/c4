package main

import "strings"

func Compatible(a, b Type) bool {
	if a.Equals(b) || b.Equals(a) {
		return true
	}
	switch a.(type) {
	case IntLitType, FloatLitType:
		switch b.(type) {
		case IntLitType, FloatLitType, PrimitiveType:
			return true
		}
	case PrimitiveType:
		switch b.(type) {
		case IntLitType, FloatLitType:
			return true
		}
	}
	return false
}

type Type interface {
	Equals(other Type) bool
	IsConcrete() bool
	Concrete() ConcreteType
}

type ConcreteType interface {
	Type
	Metrics() TypeMetrics
	// Source code representing the type
	Format() string
	// The QBE name of the base, extended or aggregate type corresponding to this type
	IRTypeName(c *Compiler) string
	// The QBE name of the base type closest to this type, if any
	IRBaseTypeName() byte
	// Generate code to zero a value of the type
	GenZero(c *Compiler, loc Operand)
}

// TypeMetrics stores the size and alignment of a type. If a type's metrics are zero, a value of that type cannot be created.
type TypeMetrics struct {
	Size, Align int
}

type NumericType interface {
	ConcreteType
	Signed() bool
}

// The type of integral numeric literals
type IntLitType struct{}

func (_ IntLitType) Equals(other Type) bool {
	_, ok := other.(IntLitType)
	return ok
}
func (_ IntLitType) IsConcrete() bool {
	return false
}
func (_ IntLitType) Concrete() ConcreteType {
	return TypeI64
}

// The type of decimal numeric literals
type FloatLitType struct{}

func (_ FloatLitType) Equals(other Type) bool {
	_, ok := other.(FloatLitType)
	return ok
}
func (_ FloatLitType) IsConcrete() bool {
	return false
}
func (_ FloatLitType) Concrete() ConcreteType {
	return TypeF64
}

type PrimitiveType int

func (a PrimitiveType) Equals(other Type) bool {
	b, ok := other.(PrimitiveType)
	return ok && a == b
}

func (p PrimitiveType) Signed() bool {
	switch p {
	case TypeI64, TypeI32, TypeI16, TypeI8:
		return true
	case TypeU64, TypeU32, TypeU16, TypeU8:
		return false
	case TypeF64, TypeF32:
		return true
	case TypeBool:
		return false
	}
	panic("Invalid primitive type")
}

func (t PrimitiveType) IsConcrete() bool {
	return true
}
func (t PrimitiveType) Concrete() ConcreteType {
	return t
}

func (p PrimitiveType) Metrics() TypeMetrics {
	switch p {
	case TypeI64, TypeU64:
		return TypeMetrics{8, 8}
	case TypeI32, TypeU32:
		return TypeMetrics{4, 4}
	case TypeI16, TypeU16:
		return TypeMetrics{2, 2}
	case TypeI8, TypeU8, TypeBool:
		return TypeMetrics{1, 1}
	case TypeF64:
		return TypeMetrics{8, 8}
	case TypeF32:
		return TypeMetrics{4, 4}
	}
	panic("Invalid primitive type")
}

func (p PrimitiveType) Format() string {
	switch p {
	case TypeI64:
		return "I64"
	case TypeI32:
		return "I32"
	case TypeI16:
		return "I16"
	case TypeI8:
		return "I8"

	case TypeU64:
		return "U64"
	case TypeU32:
		return "U32"
	case TypeU16:
		return "U16"
	case TypeU8:
		return "U8"

	case TypeF64:
		return "F64"
	case TypeF32:
		return "F32"

	case TypeBool:
		return "Bool"
	}
	panic("Invalid primitive type")
}

func (p PrimitiveType) IRTypeName(c *Compiler) string {
	switch p {
	case TypeI64, TypeU64:
		return "l"
	case TypeI32, TypeU32:
		return "w"
	case TypeI16, TypeU16:
		return "h"
	case TypeI8, TypeU8, TypeBool:
		return "b"
	case TypeF64:
		return "d"
	case TypeF32:
		return "s"
	}
	panic("Invalid primitive type")
}

func (p PrimitiveType) IRBaseTypeName() byte {
	switch p {
	case TypeI64, TypeU64:
		return 'l'
	case TypeI32, TypeU32, TypeI16, TypeU16, TypeI8, TypeU8, TypeBool:
		return 'w'
	case TypeF64:
		return 'd'
	case TypeF32:
		return 's'
	}
	panic("Invalid primitive type")
}

const (
	TypeI64 PrimitiveType = iota
	TypeI32
	TypeI16
	TypeI8

	TypeU64
	TypeU32
	TypeU16
	TypeU8

	TypeF64
	TypeF32

	TypeBool
)

type PointerType struct {
	To ConcreteType
}

func (a PointerType) Equals(other Type) bool {
	b, ok := other.(PointerType)
	return ok && a.To.Equals(b.To)
}
func (_ PointerType) Signed() bool {
	return false
}
func (_ PointerType) IsConcrete() bool {
	return true
}
func (p PointerType) Concrete() ConcreteType {
	return p
}
func (_ PointerType) Metrics() TypeMetrics {
	return TypeMetrics{8, 8}
}
func (p PointerType) Format() string {
	return "*" + p.To.Format()
}
func (_ PointerType) IRTypeName(c *Compiler) string {
	return "l"
}
func (_ PointerType) IRBaseTypeName() byte {
	return 'l'
}

type FuncType struct {
	Var   bool // true if the function uses C-style varags
	Param []ConcreteType
	Ret   ConcreteType
}

func (a FuncType) Equals(other Type) bool {
	b, ok := other.(FuncType)
	if !ok {
		return false
	}
	if a.Ret != b.Ret && !a.Ret.Equals(b.Ret) {
		return false
	}
	if len(a.Param) != len(b.Param) {
		return false
	}
	for i := range a.Param {
		if !a.Param[i].Equals(b.Param[i]) {
			return false
		}
	}
	return true
}
func (_ FuncType) IsConcrete() bool {
	return true
}
func (f FuncType) Concrete() ConcreteType {
	return f
}
func (f FuncType) Metrics() TypeMetrics {
	return TypeMetrics{}
}
func (f FuncType) Format() string {
	params := make([]string, len(f.Param))
	for i, param := range f.Param {
		params[i] = param.Format()
	}
	return "func(" + strings.Join(params, ", ") + ") " + f.Ret.Format()
}
func (_ FuncType) IRTypeName(c *Compiler) string {
	return ""
}
func (_ FuncType) IRBaseTypeName() byte {
	return 0
}

type NamedType struct {
	ConcreteType
	Name string
}

func (a NamedType) Equals(other Type) bool {
	b, ok := other.(NamedType)
	return ok && a.Name == b.Name
}

type Field struct {
	Name string
	Ty   ConcreteType
}
type CompositeType []Field
type StructType struct{ CompositeType }
type UnionType struct{ CompositeType }

func (comp CompositeType) composite() CompositeType { return comp }

func (a CompositeType) equals(b CompositeType) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if !a[i].Ty.Equals(b[i].Ty) {
			return false
		}
	}
	return true

}
func (comp CompositeType) IsConcrete() bool {
	return true
}
func (comp CompositeType) format() string {
	b := &strings.Builder{}
	b.WriteString("{\n")
	for _, field := range comp {
		b.WriteByte('\t')
		b.WriteString(field.Name)
		b.WriteByte(' ')
		b.WriteString(field.Ty.Format())
		b.WriteByte('\n')
	}
	b.WriteByte('}')
	return b.String()
}
func (comp CompositeType) IRBaseTypeName() byte {
	return 0
}

func (a StructType) Equals(other Type) bool {
	b, ok := other.(StructType)
	return ok && a.equals(b.CompositeType)
}
func (s StructType) Concrete() ConcreteType {
	return s
}
func (s StructType) Metrics() (m TypeMetrics) {
	for _, field := range s.CompositeType {
		fm := field.Ty.Metrics()
		if m.Align < fm.Align {
			m.Align = fm.Align
		}
		m.Size += fm.Size
	}
	return
}
func (s StructType) Format() string {
	return "struct " + s.format()
}
func (s StructType) IRTypeName(c *Compiler) string {
	return c.CompositeType(s.layout(c))
}
func (s StructType) layout(c *Compiler) CompositeLayout {
	var ent CompositeEntry
	var layout CompositeLayout
	for _, field := range s.CompositeType {
		ty := field.Ty.IRTypeName(c)
		if ent.N > 0 && ent.Ty != ty {
			layout = append(layout, ent)
			ent.N = 0
		}
		ent.Ty = ty
		ent.N++
	}
	if ent.N > 0 {
		layout = append(layout, ent)
	}
	return layout
}

func (a UnionType) Equals(other Type) bool {
	b, ok := other.(UnionType)
	return ok && a.equals(b.CompositeType)
}
func (u UnionType) Concrete() ConcreteType {
	return u
}
func (u UnionType) Metrics() TypeMetrics {
	return u.largest().Ty.Metrics()
}
func (u UnionType) Format() string {
	return "union " + u.format()
}
func (u UnionType) IRTypeName(c *Compiler) string {
	return c.CompositeType(u.layout(c))
}
func (u UnionType) layout(c *Compiler) CompositeLayout {
	return CompositeLayout{{u.largest().Ty.IRTypeName(c), 1}}
}
func (u UnionType) largest() (f Field) {
	fs := 0
	for _, field := range u.CompositeType {
		fsiz := field.Ty.Metrics().Size
		if fsiz > fs {
			f = field
			fs = fsiz
		}
	}
	return
}
