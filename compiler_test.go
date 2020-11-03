package main

import (
	"fmt"
	"runtime/debug"
	"strings"
	"testing"
)

func spc(ch byte) bool {
	return ch == ' ' || ch == '\t' || ch == '\n'
}

func testCompile(t *testing.T, code, ir string) {
	toks := make(chan Token)
	go Tokenize(code, toks)
	p := parser{<-toks, toks}
	defer func() {
		switch e := recover().(type) {
		case nil:
		case string:
			t.Fatalf("Parse error: %s\n%s", e, debug.Stack())
		default:
			panic(e)
		}
	}()
	prog := p.parseProgram()

	b := &strings.Builder{}
	c := NewCompiler(b)
	c.Compile(prog)

	// Compare without taking into account indentation
	ir0 := []byte(b.String())
	ir1 := []byte(ir)
	var i0, i1 int
	for {
		spc0 := false
		for i0 < len(ir0) && spc(ir0[i0]) {
			i0++
			spc0 = true
		}
		spc1 := false
		for i1 < len(ir1) && spc(ir1[i1]) {
			i1++
			spc1 = true
		}

		end0 := i0 >= len(ir0)
		end1 := i1 >= len(ir1)
		if end0 && end1 {
			return
		} else if end0 || end1 {
			t.Fatal("Generated and expected IRs are of different length")
		}

		if i0 == 0 {
			spc0 = true
		}
		if i1 == 0 {
			spc1 = true
		}

		if spc0 != spc1 || ir0[i0] != ir1[i1] {
			t.Fatalf("Generated and expected IRs do not match at bytes %d, %d\n%s!!%s", i0, i1, ir0[:i0], ir0[i0:])
		}
		i0++
		i1++
	}
}

func testMainCompile(t *testing.T, code, ir string) {
	code = "pub fn main() I32 {\n" + code + "\n}\n"
	ir = "export function w $main() {\n@start\n" + ir + "\n}\n"
	testCompile(t, code, ir)
}

func TestReturn0(t *testing.T) {
	testMainCompile(t, `return 0`, `ret 0`)
}

func TestPrefixExpr(t *testing.T) {
	testMainCompile(t, `
		!3
		^3
		-(3)
		+(3)

		return 0
	`, `
		%t1 =l ceql 0, 3
		%t2 =l xor -1, 3
		%t3 =l sub 0, 3
		%t4 =l copy 3

		ret 0
	`)
}

func TestMutate(t *testing.T) {
	n := 0
	m := func(op string) string {
		n += 2
		return fmt.Sprintf(`
			%%t%[1]d =w loadw %%t1
			%%t%[2]d =w %[3]s %%t%[1]d, 1
			storew %%t%[2]d, %%t1
		`, n, n+1, op)
	}
	testMainCompile(t, `
		var a I32
		a += 1; a -= 1; a *= 1; a /= 1
		a %= 1; a |= 1; a ^= 1; a &= 1
		a <<= 1; a >>= 1
		// a &&= 1
		// a ||= 1
	`, `
		%t1 =l alloc4 4
		storew 0, %t1`+
		m("add")+m("sub")+m("mul")+m("div")+
		m("rem")+m("or")+m("xor")+m("and")+
		m("shl")+m("sar"))
}

// TODO: test unsigned div, mod and shr
func TestArithmetic(t *testing.T) {
	testMainCompile(t, `
		4 + 2
		4 - 2
		4 * 2
		4 / 2
		4 % 2

		4 | 2
		4 ^ 2
		4 & 2
		4 << 2
		4 >> 2

		return 0
	`, `
		%t1 =l add 4, 2
		%t2 =l sub 4, 2
		%t3 =l mul 4, 2
		%t4 =l div 4, 2
		%t5 =l rem 4, 2

		%t6  =l or  4, 2
		%t7  =l xor 4, 2
		%t8  =l and 4, 2
		%t9  =l shl 4, 2
		%t10 =l sar 4, 2

		ret 0
	`)
}

func TestNestedArithmetic(t *testing.T) {
	testMainCompile(t, `return (1 + 10*2) * 2`, `
		%t1 =l mul 10, 2
		%t2 =l add 1, %t1
		%t3 =l mul %t2, 2
		ret %t3
	`)
}

func TestVariables(t *testing.T) {
	testCompile(t, `
		var global I32
		pub fn main() I32 {
			var i, j I32
			i = 7
			j = 5
			i = i + j
			return i + global
		}
	`, `
		export function w $main() {
		@start
			%t1 =l alloc4 4
			storew 0, %t1
			%t2 =l alloc4 4
			storew 0, %t2

			storew 7, %t1
			storew 5, %t2

			%t3 =w loadw %t1
			%t4 =w loadw %t2
			%t5 =w add %t3, %t4
			storew %t5, %t1

			%t6 =w loadw %t1
			%t7 =w loadw $global
			%t8 =w add %t6, %t7
			ret %t8
		}
	`)
}

func TestTypeDef(t *testing.T) {
	testCompile(t, `
		type Foo I32
		type Bar U64
		pub fn main() I32 {
			var foo Foo
			foo / foo

			var bar Bar
			bar / bar

			return 0
		}
	`, `
		export function w $main() {
		@start
			%t1 =l alloc4 4
			storew 0, %t1
			%t2 =w loadw %t1
			%t3 =w loadw %t1
			%t4 =w div %t2, %t3

			%t5 =l alloc8 8
			storel 0, %t5
			%t6 =l loadl %t5
			%t7 =l loadl %t5
			%t8 =l udiv %t6, %t7

			ret 0
		}
	`)
}

func TestSmallTypes(t *testing.T) {
	testMainCompile(t, `
		var i, j I16
		i = 7
		j = 5
		i = i + j

		var k, l U8
		k = 7
		l = 5
		k = k + l
	`, `
		%t1 =l alloc4 2
		storeh 0, %t1
		%t2 =l alloc4 2
		storeh 0, %t2

		storeh 7, %t1
		storeh 5, %t2

		%t3 =w loadsh %t1
		%t4 =w loadsh %t2
		%t5 =w add %t3, %t4
		storeh %t5, %t1


		%t6 =l alloc4 1
		storeb 0, %t6
		%t7 =l alloc4 1
		storeb 0, %t7

		storeb 7, %t6
		storeb 5, %t7

		%t8 =w loadub %t6
		%t9 =w loadub %t7
		%t10 =w add %t8, %t9
		storeb %t10, %t6
	`)
}

func TestIf(t *testing.T) {
	testMainCompile(t, `
		if 1 {
			return 0
		}
		if 0 {
			return 1
		}
	`, `
		jnz 1, @b1, @b2
	@b1
		ret 0
		jmp @b3
	@b2
	@b3

		jnz 0, @b4, @b5
	@b4
		ret 1
		jmp @b6
	@b5
	@b6
	`)
}

func TestIfElse(t *testing.T) {
	testMainCompile(t, `
		if 1 {
			return 0
		} else {
			return 1
		}
		if 0 {
			return 2
		} else {
			return 3
		}
	`, `
		jnz 1, @b1, @b2
	@b1
		ret 0
		jmp @b3
	@b2
		ret 1
	@b3

		jnz 0, @b4, @b5
	@b4
		ret 2
		jmp @b6
	@b5
		ret 3
	@b6
	`)
}

func TestElseIf(t *testing.T) {
	testMainCompile(t, `
		if 1 {
			return 0
		} else if 2 {
			return 1
		} else if 3 {
			return 2
		} else {
			return 3
		}
	`, `
		jnz 1, @b1, @b2
	@b1
		ret 0
		jmp @b3
	@b2
		jnz 2, @b4, @b5
	@b4
		ret 1
		jmp @b6
	@b5
		jnz 3, @b7, @b8
	@b7
		ret 2
		jmp @b9
	@b8
		ret 3
	@b9
	@b6
	@b3
	`)
}

func TestFor0(t *testing.T) {
	testMainCompile(t, `
		for {return 0}
		for ;; {return 1}
	`, `
	@b1
	@b2
		ret 0
		jmp @b1
	@b3

	@b4
	@b5
		ret 1
		jmp @b4
	@b6
	`)
}

func TestFor1(t *testing.T) {
	testMainCompile(t, `
		for 1 {return 0}
		for ; 2; {return 1}
	`, `
	@b1
		jnz 1, @b2, @b3
	@b2
		ret 0
		jmp @b1
	@b3

	@b4
		jnz 2, @b5, @b6
	@b5
		ret 1
		jmp @b4
	@b6
	`)
}

func TestFor1Other(t *testing.T) {
	testMainCompile(t, `
		var a I32
		for a = 1;; {return 0}
		for ;; a = 2 {return 1}
	`, `
		%t1 =l alloc4 4
		storew 0, %t1

		storew 1, %t1
	@b1
	@b2
		ret 0
		jmp @b1
	@b3

	@b4
	@b5
		ret 1
		storew 2, %t1
		jmp @b4
	@b6
	`)
}

func TestFor2(t *testing.T) {
	testMainCompile(t, `
		var a I32
		for a = 0; 1; {return 0}
		for ; 0; a = 1 {return 1}
	`, `
		%t1 =l alloc4 4
		storew 0, %t1

		storew 0, %t1
	@b1
		jnz 1, @b2, @b3
	@b2
		ret 0
		jmp @b1
	@b3

	@b4
		jnz 0, @b5, @b6
	@b5
		ret 1
		storew 1, %t1
		jmp @b4
	@b6
	`)
}

func TestFor3(t *testing.T) {
	testMainCompile(t, `
		var a I32
		for a = 0; 1; a = 1 {return 0}
	`, `
		%t1 =l alloc4 4
		storew 0, %t1

		storew 0, %t1
	@b1
		jnz 1, @b2, @b3
	@b2
		ret 0
		storew 1, %t1
		jmp @b1
	@b3
	`)
}

func TestReferenceVariable(t *testing.T) {
	testMainCompile(t, `
		var i I32
		var p [I32]
		p = &i
		return 0
	`, `
		%t1 =l alloc4 4
		storew 0, %t1
		%t2 =l alloc8 8
		storel 0, %t2
		storel %t1, %t2
		ret 0
	`)
}

func TestDereferencePointer(t *testing.T) {
	testMainCompile(t, `
		var p [I32]
		return [p]
	`, `
		%t1 =l alloc8 8
		storel 0, %t1
		%t2 =l loadl %t1
		%t3 =w loadw %t2
		ret %t3
	`)
}

func TestFunctionCall(t *testing.T) {
	testCompile(t, `
		fn printi(i I64)
		pub fn main() I32 {
			printi(42)
			return 0
		}
	`, `
		export function w $main() {
		@start
			call $printi(l 42)
			ret 0
		}
	`)
}

func TestStringLiteral(t *testing.T) {
	testCompile(t, `
		fn puts(s [I8]) I32
		pub fn main() I32 {
			puts("str0")
			puts("str0")
			puts("str1")
			puts("str1")
			puts("str2")
			puts("str2")
			return 0
		}
	`, `
		export function w $main() {
		@start
			%t1 =w call $puts(l $str0)
			%t2 =w call $puts(l $str0)
			%t3 =w call $puts(l $str1)
			%t4 =w call $puts(l $str1)
			%t5 =w call $puts(l $str2)
			%t6 =w call $puts(l $str2)
			ret 0
		}
		data $str0 = { b "str0", b 0 }
		data $str1 = { b "str1", b 0 }
		data $str2 = { b "str2", b 0 }
	`)
}
