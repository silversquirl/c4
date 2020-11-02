package main

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

func (d VarDecl) Format() string {
	return "var " + d.Name + " " + d.Ty.Format()
}

func (r ReturnStmt) Format() string {
	return "return " + r.Value.Format()
}
func (e AssignExpr) Format() string {
	return e.L.Format() + " = " + e.R.Format()
}
func (e MutateExpr) Format() string {
	return fmt.Sprintf("%s %s= %s", e.L.Format(), e.Op, e.R.Format())
}

func (e CallExpr) Format() string {
	args := make([]string, len(e.Args))
	for i, arg := range e.Args {
		args[i] = arg.Format()
	}
	return e.Func.Format() + "(" + strings.Join(args, ", ") + ")"
}

func (e VarExpr) Format() string {
	return string(e)
}

func (e RefExpr) Format() string {
	return "&" + e.V.Format()
}

func (e DerefExpr) Format() string {
	return "[" + e.V.Format() + "]"
}

func (e PrefixExpr) Format() string {
	return e.Op.String() + e.V.Format()
}
func (e BinaryExpr) Format() string {
	// TODO: smarter spacing/parenthesizing
	return fmt.Sprintf("(%s %s %s)", e.L.Format(), e.Op, e.R.Format())
}

func (e IntegerExpr) Format() string {
	return string(e)
}
func (e FloatExpr) Format() string {
	return string(e)
}
func (e StringExpr) Format() string {
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

func (name NamedTypeExpr) Format() string {
	return string(name)
}
func (ptr PointerTypeExpr) Format() string {
	return "[" + ptr.To.Format() + "]"
}
func (fun FuncTypeExpr) Format() string {
	params := make([]string, len(fun.Param))
	for i, param := range fun.Param {
		params[i] = param.Format()
	}
	var ret string
	if fun.Ret != nil {
		ret = " " + fun.Ret.Format()
	}
	return "fn(" + strings.Join(params, ", ") + ")" + ret
}
