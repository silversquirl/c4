package main

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

type FormattableCode interface {
	Format(indent int) string
}

func (prog Program) Format(indent int) string {
	b := &strings.Builder{}
	for _, tl := range prog {
		b.WriteString(tl.Format(indent))
		b.WriteString(newLine(indent))
	}
	return b.String()
}

func (ns NamespaceTL) Format(indent int) string {
	b := &strings.Builder{}
	b.WriteString("ns ")
	b.WriteString(ns.Name)
	b.WriteString(" {")
	for _, tl := range ns.Body {
		b.WriteString(newLine(indent + 1))
		b.WriteString(tl.Format(indent + 1))
	}
	b.WriteString(newLine(indent))
	b.WriteString("}")
	return b.String()
}

func (f Function) Format(indent int) string {
	b := &strings.Builder{}
	if f.Pub {
		b.WriteString("pub ")
	}
	b.WriteString("fn ")
	b.WriteString(f.Name)

	b.WriteByte('(')
	for i, param := range f.Param {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString(param.Name)
		b.WriteByte(' ')
		b.WriteString(param.Ty.Format(indent))
	}
	b.WriteByte(')')

	if f.Ret != nil {
		b.WriteByte(' ')
		b.WriteString(f.Ret.Format(indent))
	}

	b.WriteByte(' ')
	b.WriteString(fmtBlock(indent, f.Body))

	return b.String()
}

func (d VarDecl) Format(indent int) string {
	return "var " + d.Name + " " + d.Ty.Format(indent)
}
func (d VarsDecl) Format(indent int) string {
	return "var " + strings.Join(d.Names, ", ") + " " + d.Ty.Format(indent)
}

func (t TypeDef) Format(indent int) string {
	return "type " + t.Name + " " + t.Ty.Format(indent)
}
func (t TypeAlias) Format(indent int) string {
	return "type " + t.Name + " = " + t.Ty.Format(indent)
}

func (i IfStmt) Format(indent int) string {
	s := "if " + i.Cond.Format(indent) + " " + fmtBlock(indent, i.Then)
	if i.Else != nil {
		s += " else " + fmtBlock(indent, i.Else)
	}
	return s
}

func (f ForStmt) Format(indent int) string {
	b := &strings.Builder{}
	b.WriteString("for ")
	if f.Init != nil || f.Step != nil {
		if f.Init != nil {
			b.WriteString(f.Init.Format(indent))
		}
		b.WriteByte(';')
		if f.Cond != nil {
			b.WriteByte(' ')
			b.WriteString(f.Cond.Format(indent))
		}
		b.WriteByte(';')
		if f.Step != nil {
			b.WriteByte(' ')
			b.WriteString(f.Step.Format(indent))
		}
		b.WriteByte(' ')
	} else if f.Cond != nil {
		b.WriteString(f.Cond.Format(indent))
		b.WriteByte(' ')
	}
	b.WriteString(fmtBlock(indent, f.Body))
	return b.String()
}

func (r ReturnStmt) Format(indent int) string {
	return "return " + r.Value.Format(indent)
}

func (e AccessExpr) Format(indent int) string {
	return e.L.Format(indent) + "." + e.R
}
func (e AssignExpr) Format(indent int) string {
	return "(" + e.L.Format(indent) + " = " + e.R.Format(indent) + ")"
}
func (e MutateExpr) Format(indent int) string {
	return fmt.Sprintf("%s %s= %s", e.L.Format(indent), e.Op, e.R.Format(indent))
}

func (e CallExpr) Format(indent int) string {
	args := make([]string, len(e.Args))
	for i, arg := range e.Args {
		args[i] = arg.Format(indent)
	}
	return e.Func.Format(indent) + "(" + strings.Join(args, ", ") + ")"
}

func (e CastExpr) Format(indent int) string {
	return "cast(" + e.V.Format(0) + ", " + e.Ty.Format(0) + ")"
}

func (e VarExpr) Format(indent int) string {
	return string(e)
}

func (e RefExpr) Format(indent int) string {
	return "&" + e.V.Format(indent)
}

func (e DerefExpr) Format(indent int) string {
	return "[" + e.V.Format(indent) + "]"
}

func (e PrefixExpr) Format(indent int) string {
	return e.Op.String() + "(" + e.V.Format(indent) + ")"
}
func (e BinaryExpr) Format(indent int) string {
	return fmt.Sprintf("(%s %s %s)", e.L.Format(indent), e.Op, e.R.Format(indent))
}
func (e BooleanExpr) Format(indent int) string {
	return fmt.Sprintf("(%s %s %s)", e.L.Format(indent), e.Op, e.R.Format(indent))
}

func (e IntegerExpr) Format(indent int) string {
	return string(e)
}
func (e FloatExpr) Format(indent int) string {
	return string(e)
}
func escapeRune(r, sep rune) string {
	switch r {
	case '\x1b':
		return `\e`
	case '\n':
		return `\n`
	case '\r':
		return `\r`
	case '\t':
		return `\t`
	case '\\', sep:
		return `\` + string(r)
	}
	switch {
	case ' ' <= r && r <= '~': // Printable ASCII range
		return string(r)
	case sep == '\'' && r <= 0x7F:
		return fmt.Sprintf(`\x%02x`, r)
	case r <= 0xFFFF:
		return fmt.Sprintf(`\u%04x`, r)
	default:
		return fmt.Sprintf(`\U%08x`, r)
	}
}
func (e StringExpr) Format(indent int) string {
	b := &strings.Builder{}
	b.WriteRune('"')
	str := []byte(e)
	for len(str) > 0 {
		r, size := utf8.DecodeRune(str)
		if r == utf8.RuneError {
			fmt.Fprintf(b, `\x%02x`, str[0])
		} else {
			b.WriteString(escapeRune(r, '"'))
		}
		str = str[size:]
	}
	b.WriteRune('"')
	return b.String()
}
func (e RuneExpr) Format(indent int) string {
	return "'" + escapeRune(rune(e), '\'') + "'"
}

func (name NamedTypeExpr) Format(indent int) string {
	return string(name)
}
func (ns NamespaceTypeExpr) Format(indent int) string {
	return strings.Join([]string(ns), ".")
}
func (ptr PointerTypeExpr) Format(indent int) string {
	return "[" + ptr.To.Format(indent) + "]"
}
func (fun FuncTypeExpr) Format(indent int) string {
	params := make([]string, len(fun.Param))
	for i, param := range fun.Param {
		params[i] = param.Format(indent)
	}
	var ret string
	if fun.Ret != nil {
		ret = " " + fun.Ret.Format(indent)
	}
	return "fn(" + strings.Join(params, ", ") + ")" + ret
}
func (s StructTypeExpr) Format(indent int) string {
	return s.Get(nil).Format(indent)
}
func (u UnionTypeExpr) Format(indent int) string {
	return u.Get(nil).Format(indent)
}

func fmtBlock(indent int, body []Statement) string {
	b := &strings.Builder{}
	b.WriteByte('{')
	for _, s := range body {
		b.WriteString(newLine(indent))
		b.WriteString(s.Format(indent + 1))
	}
	b.WriteString(newLine(indent))
	b.WriteByte('}')
	return b.String()
}

func newLine(indent int) string {
	b := make([]byte, indent+1)
	for indent > 0 {
		indent--
		b[indent] = '\t'
	}
	b[len(b)-1] = '\n'
	return string(b)
}
