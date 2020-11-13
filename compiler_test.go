package main

import (
	"fmt"
	"runtime/debug"
	"testing"
)

func testCompile(t *testing.T, code, ir string) {
	toks := make(chan Token)
	go Tokenize(code, toks)

	phase := "Parse"
	p := parser{<-toks, toks}
	defer func() {
		switch e := recover().(type) {
		case nil:
		case string:
			t.Fatalf("%s error: %s\n%s", phase, e, debug.Stack())
		default:
			panic(e)
		}
	}()
	prog := p.parseProgram()

	phase = "Compile"
	c := NewCompiler()
	c.compile(prog)

	gen := c.r.String()
	if eq, ai, bi := CodeCompare(gen, ir); !eq {
		t.Fatalf("Generated and expected IRs do not match at bytes %d, %d\n%s!!%s", ai, bi, gen[:ai], gen[ai:])
	}
}

func testCompileFailure(t *testing.T, err, code string) {
	toks := make(chan Token)
	go Tokenize(code, toks)

	p := parser{<-toks, toks}
	prog := p.parseProgram()

	defer func() {
		switch e := recover().(type) {
		case nil:
			t.Fatal("No error")
		case string:
			if e != err {
				t.Fatal("Incorrect error:", e)
			}
		default:
			panic(e)
		}
	}()
	NewCompiler().compile(prog)
}

func testMainCompile(t *testing.T, code, ir string) {
	code = "pub fn main() I32 {\n" + code + "\n\treturn 0\n}\n"
	ir = "export function w $main() {\n@start\n" + ir + "\n\tret 0\n}\n"
	testCompile(t, code, ir)
}

func TestFunctionArgs(t *testing.T) {
	testCompile(t, `
		fn foo(a, b I32, c U64) U64 {
			a = b
			return c
		}
	`, `
		function l $foo(w %t1, w %t2, l %t3) {
		@start
			%t4 =l alloc4 4
			storew %t1, %t4
			%t5 =l alloc4 4
			storew %t2, %t5
			%t6 =l alloc8 8
			storel %t3, %t6

			%t7 =w loadw %t5
			storew %t7, %t4
			%t8 =l loadl %t6
			ret %t8
		}
	`)
}

func TestRecursiveFunction(t *testing.T) {
	testCompile(t, `
		fn foo() {
			foo()
		}
	`, `
		function $foo() {
		@start
			call $foo()
			ret
		}
	`)
}

func TestVariadicFunction(t *testing.T) {
	testCompile(t, `
		variadic fn foo(a I32)
		fn bar() {
			foo(1, 2)
		}
	`, `
		function $bar() {
		@start
			call $foo(l 1, l 2)
			ret
		}
	`)
}

func TestNamespace(t *testing.T) {
	testCompile(t, `
		ns foo {
			fn bar() {}
			type Bar I8
		}
		fn bar() {
			var x foo.Bar
			foo.bar()
		}
	`, `
		function $foo.bar() {
		@start
			ret
		}
		function $bar() {
		@start
			%t1 =l alloc4 1
			storeb 0, %t1
			call $foo.bar()
			ret
		}
	`)
}

func TestScope(t *testing.T) {
	testCompile(t, `
		fn foo() {}
		fn bar() {
			var foo I32
		}
		fn baz() {
			var foo I32
		}
	`, `
		function $foo() {
		@start
			ret
		}
		function $bar() {
		@start
			%t1 =l alloc4 4
			storew 0, %t1
			ret
		}
		function $baz() {
		@start
			%t1 =l alloc4 4
			storew 0, %t1
			ret
		}
	`)
}

func TestReturn0(t *testing.T) {
	testMainCompile(t, "", "")
}

func TestPrefixExpr(t *testing.T) {
	testMainCompile(t, `
		_ = !3
		_ = ^3
		_ = -(3)
		_ = +(3)
	`, `
		%t1 =l ceql 0, 3
		%t2 =l xor -1, 3
		%t3 =l sub 0, 3
		%t4 =l copy 3
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
	`, `
		%t1 =l alloc4 4
		storew 0, %t1`+
		m("add")+m("sub")+m("mul")+m("div")+
		m("rem")+m("or")+m("xor")+m("and")+
		m("shl")+m("sar"))
}

func TestIncrDecr(t *testing.T) {
	testMainCompile(t, `
		var a I32
		a++
		a--
	`, `
		%t1 =l alloc4 4
		storew 0, %t1

		%t2 =w loadw %t1
		%t3 =w add %t2, 1
		storew %t3, %t1

		%t4 =w loadw %t1
		%t5 =w sub %t4, 1
		storew %t5, %t1
	`)
}

// TODO: test unsigned div, mod and shr
func TestArithmetic(t *testing.T) {
	testMainCompile(t, `
		_ = 4 + 2
		_ = 4 - 2
		_ = 4 * 2
		_ = 4 / 2
		_ = 4 % 2

		_ = 4 | 2
		_ = 4 ^ 2
		_ = 4 & 2
		_ = 4 << 2
		_ = 4 >> 2
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
	`)
}

func TestComparison(t *testing.T) {
	testMainCompile(t, `
		_ = 4 == 2
		_ = 4 != 2
		_ = 4 < 2
		_ = 4 > 2
		_ = 4 <= 2
		_ = 4 >= 2
	`, `
		%t1 =l ceql 4, 2
		%t2 =l cnel 4, 2
		%t3 =l csltl 4, 2
		%t4 =l csgtl 4, 2
		%t5 =l cslel 4, 2
		%t6 =l csgel 4, 2
	`)
}

func TestBoolean(t *testing.T) {
	testMainCompile(t, `
		_ = 4 && 2
		_ = 4 || 2
	`, `
		%t1 =l copy 4
		jnz %t1, @b1, @b2
	@b1
		%t1 =l copy 2
	@b2

		%t2 =l copy 4
		jnz %t2, @b4, @b3
	@b3
		%t2 =l copy 2
	@b4
	`)
}

func TestCast(t *testing.T) {
	testMainCompile(t, `
		var i I32
		var u U64
		i = cast(u, I32)
		u = cast(i, U64)
	`, `
		%t1 =l alloc4 4
		storew 0, %t1
		%t2 =l alloc8 8
		storel 0, %t2

		%t3 =l loadl %t2
		storew %t3, %t1

		%t4 =w loadw %t1
		%t5 =l extsw %t4
		storel %t5, %t2
	`)
}

func TestNestedArithmetic(t *testing.T) {
	testMainCompile(t, `_ = (1 + 10*2) * 2`, `
		%t1 =l mul 10, 2
		%t2 =l add 1, %t1
		%t3 =l mul %t2, 2
	`)
}

func TestLocalVariables(t *testing.T) {
	testMainCompile(t, `
		var i, j I32
		i = 7
		j = 5
		i = i + j
	`, `
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
	`)
}

func TestGlobalVariables(t *testing.T) {
	testCompile(t, `
		extern var foo I32
		var bar I32
		pub fn main() I32 {
			return foo + bar
		}
	`, `
		export function w $main() {
		@start
			%t1 =w loadw $foo
			%t2 =w loadw $bar
			%t3 =w add %t1, %t2
			ret %t3
		}
		data $bar = align 4 { z 4 }
	`)
}

func TestTypeDef(t *testing.T) {
	testCompile(t, `
		type Foo I32
		type Bar U64
		type Baz [I8]
		pub fn main() I32 {
			var foo Foo
			_ = foo / foo

			var bar Bar
			_ = bar / bar

			var baz Baz
			_ = [baz + 3]

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

			%t9 =l alloc8 8
			storel 0, %t9
			%t10 =l loadl %t9
			%t11 =l add %t10, 3
			%t12 =w loadsb %t11

			ret 0
		}
	`)
}

func TestTypeAlias(t *testing.T) {
	testCompile(t, `
		type Foo = I32
		pub fn main() I32 {
			var foo Foo
			var bar I32
			_ = foo / bar

			return 0
		}
	`, `
		export function w $main() {
		@start
			%t1 =l alloc4 4
			storew 0, %t1
			%t2 =l alloc4 4
			storew 0, %t2
			%t3 =w loadw %t1
			%t4 =w loadw %t2
			%t5 =w div %t3, %t4

			ret 0
		}
	`)
}

func TestStruct(t *testing.T) {
	testCompile(t, `
		type Foo struct { a, b I32; c I64 }
		type Bar struct { a, b, c I8 }
		type Baz struct { a I8; b I64; c I8 }
		fn fooFn(_ Foo)
		fn barFn(_ Bar)
		fn bazFn(_ Baz)
		pub fn main() I32 {
			var foo Foo
			fooFn(foo)
			var bar Bar
			barFn(bar)
			var baz Baz
			bazFn(baz)
			return 0
		}
	`, `
		type :b3 = { b 3 }
		type :blb = { b, l, b }
		type :w2l = { w 2, l }
		export function w $main() {
		@start
			%t1 =l alloc8 16
			storew 0, %t1
			%t2 =l add %t1, 4
			storew 0, %t2
			%t3 =l add %t1, 8
			storel 0, %t3

			call $fooFn(:w2l %t1)

			%t4 =l alloc4 3
			storeb 0, %t4
			%t5 =l add %t4, 1
			storeb 0, %t5
			%t6 =l add %t4, 2
			storeb 0, %t6

			call $barFn(:b3 %t4)

			%t7 =l alloc8 24
			storeb 0, %t7
			%t8 =l add %t7, 8
			storel 0, %t8
			%t9 =l add %t7, 16
			storeb 0, %t9

			call $bazFn(:blb %t7)

			ret 0
		}
	`)
}

func TestUnion(t *testing.T) {
	testCompile(t, `
		type Foo union { a, b I32; c I64 }
		type Bar union { a, b, c I8 }
		fn fooFn(_ Foo)
		fn barFn(_ Bar)
		pub fn main() I32 {
			var foo Foo
			fooFn(foo)
			var bar Bar
			barFn(bar)
			return 0
		}
	`, `
		type :b = { b }
		type :l = { l }
		export function w $main() {
		@start
			%t1 =l alloc8 8
			storel 0, %t1
			call $fooFn(:l %t1)

			%t2 =l alloc4 1
			storeb 0, %t2
			call $barFn(:b %t2)

			ret 0
		}
	`)
}

func TestCompositeReturn(t *testing.T) {
	testCompile(t, `
		type S struct { a I32 }
		type U struct { a I32 }
		fn sf() S {
			var s S
			return s
		}
		fn uf() U {
			var u U
			return u
		}
	`, `
		type :w = { w }
		function :w $sf() {
		@start
			%t1 =l alloc4 4
			storew 0, %t1
			ret %t1
		}
		function :w $uf() {
		@start
			%t1 =l alloc4 4
			storew 0, %t1
			ret %t1
		}
	`)
}

func TestFieldAccess(t *testing.T) {
	testCompile(t, `
		type Foo struct { a, b I32; c I64 }
		type Bar union { a, b I32; c I64 }
		fn f() {
			var foo Foo
			_ = foo.a
			_ = foo.b
			_ = foo.c

			var bar Bar
			_ = bar.a
			_ = bar.b
			_ = bar.c
		}
	`, `
		function $f() {
		@start
			%t1 =l alloc8 16
			storew 0, %t1
			%t2 =l add %t1, 4
			storew 0, %t2
			%t3 =l add %t1, 8
			storel 0, %t3

			%t4 =w loadw %t1
			%t5 =l add %t1, 4
			%t6 =w loadw %t5
			%t7 =l add %t1, 8
			%t8 =l loadl %t7

			%t9 =l alloc8 8
			storel 0, %t9

			%t10 =w loadw %t9
			%t11 =w loadw %t9
			%t12 =l loadl %t9

			ret
		}
	`)
}

func TestAccessPointer(t *testing.T) {
	testCompile(t, `
		type Foo struct { a, b I32; c I64 }
		fn f() {
			var foo [Foo]
			_ = foo.a
			_ = foo.b
			_ = foo.c
		}
	`, `
		function $f() {
		@start
			%t1 =l alloc8 8
			storel 0, %t1

			%t2 =l loadl %t1
			%t3 =w loadw %t2

			%t4 =l loadl %t1
			%t5 =l add %t4, 4
			%t6 =w loadw %t5

			%t7 =l loadl %t1
			%t8 =l add %t7, 8
			%t9 =l loadl %t8

			ret
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

func TestSmallReturnType(t *testing.T) {
	testCompile(t, `
		fn bool(b Bool) Bool {
			return b
		}
		fn i8(i I8) I8 {
			return i
		}
		fn i16(i I16) I16 {
			return i
		}
	`, `
		function w $bool(w %t1) {
		@start
			%t2 =l alloc4 1
			storeb %t1, %t2
			%t3 =w loadub %t2
			ret %t3
		}
		function w $i8(w %t1) {
		@start
			%t2 =l alloc4 1
			storeb %t1, %t2
			%t3 =w loadsb %t2
			ret %t3
		}
		function w $i16(w %t1) {
		@start
			%t2 =l alloc4 2
			storeh %t1, %t2
			%t3 =w loadsh %t2
			ret %t3
		}
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
	@b2
	@b3

		jnz 0, @b4, @b5
	@b4
		ret 1
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
	@b2
		ret 1
	@b3

		jnz 0, @b4, @b5
	@b4
		ret 2
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
	@b2
		jnz 2, @b4, @b5
	@b4
		ret 1
	@b5
		jnz 3, @b7, @b8
	@b7
		ret 2
	@b8
		ret 3
	@b9
	@b6
	@b3
	`)
}

func TestFor0(t *testing.T) {
	testMainCompile(t, `
		var a I32
		for {a = 0}
		for ;; {a = 1}
	`, `
		%t1 =l alloc4 4
		storew 0, %t1

	@b1
	@b2
		storew 0, %t1
		jmp @b1
	@b3

	@b4
	@b5
		storew 1, %t1
		jmp @b4
	@b6
	`)
}

func TestFor1(t *testing.T) {
	testMainCompile(t, `
		var a I32
		for 1 {a = 0}
		for ; 2; {a = 1}
	`, `
		%t1 =l alloc4 4
		storew 0, %t1

	@b1
		jnz 1, @b2, @b3
	@b2
		storew 0, %t1
		jmp @b1
	@b3

	@b4
		jnz 2, @b5, @b6
	@b5
		storew 1, %t1
		jmp @b4
	@b6
	`)
}

func TestFor1Other(t *testing.T) {
	testMainCompile(t, `
		var a I32
		for a = 1;; {a = 0}
		for ;; a = 2 {a = 1}
	`, `
		%t1 =l alloc4 4
		storew 0, %t1

		storew 1, %t1
	@b1
	@b2
		storew 0, %t1
		jmp @b1
	@b3

	@b4
	@b5
		storew 1, %t1
		storew 2, %t1
		jmp @b4
	@b6
	`)
}

func TestFor2(t *testing.T) {
	testMainCompile(t, `
		var a I32
		for a = 0; 1; {a = 0}
		for ; 0; a = 1 {a = 1}
	`, `
		%t1 =l alloc4 4
		storew 0, %t1

		storew 0, %t1
	@b1
		jnz 1, @b2, @b3
	@b2
		storew 0, %t1
		jmp @b1
	@b3

	@b4
		jnz 0, @b5, @b6
	@b5
		storew 1, %t1
		storew 1, %t1
		jmp @b4
	@b6
	`)
}

func TestFor3(t *testing.T) {
	testMainCompile(t, `
		var a I32
		for a = 0; 1; a = 1 {a = 0}
	`, `
		%t1 =l alloc4 4
		storew 0, %t1

		storew 0, %t1
	@b1
		jnz 1, @b2, @b3
	@b2
		storew 0, %t1
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
	`, `
		%t1 =l alloc4 4
		storew 0, %t1
		%t2 =l alloc8 8
		storel 0, %t2
		storel %t1, %t2
	`)
}

func TestDereferencePointer(t *testing.T) {
	testMainCompile(t, `
		var p [I32]
		_ = [p]
	`, `
		%t1 =l alloc8 8
		storel 0, %t1

		%t2 =l loadl %t1
		%t3 =w loadw %t2
	`)
}

func TestPointerArithmetic(t *testing.T) {
	testMainCompile(t, `
		var p [I32]
		p += 1
		var i I32
		p += i
		var bp [I8]
		bp += 1
	`, `
		%t1 =l alloc8 8
		storel 0, %t1

		%t2 =l loadl %t1
		%t3 =l mul 4, 1
		%t4 =l add %t2, %t3
		storel %t4, %t1

		%t5 =l alloc4 4
		storew 0, %t5

		%t6 =l loadl %t1
		%t7 =w loadw %t5
		%t8 =l extsw %t7
		%t9 =l mul 4, %t8
		%t10 =l add %t6, %t9
		storel %t10, %t1

		%t11 =l alloc8 8
		storel 0, %t11

		%t12 =l loadl %t11
		%t13 =l add %t12, 1
		storel %t13, %t11
	`)
}

func TestGenericPointer(t *testing.T) {
	testMainCompile(t, `
		var p []
		p += 1
		var ip [I32]
		p = ip
		ip = p
	`, `
		%t1 =l alloc8 8
		storel 0, %t1

		%t2 =l loadl %t1
		%t3 =l add %t2, 1
		storel %t3, %t1

		%t4 =l alloc8 8
		storel 0, %t4

		%t5 =l loadl %t4
		storel %t5, %t1

		%t6 =l loadl %t1
		storel %t6, %t4
	`)

	testCompileFailure(t, "Generic pointer may not be dereferenced", `
		fn f() {
			var p []
			_ = [p]
		}
	`)
}

func TestFunctionCall(t *testing.T) {
	testCompile(t, `
		fn foo(i I64)
		fn bar(i I64) {
			foo(i)
		}
		pub fn main() I32 {
			foo(42)
			bar(42)
			return 0
		}
	`, `
		function $bar(l %t1) {
		@start
			%t2 =l alloc8 8
			storel %t1, %t2
			%t3 =l loadl %t2
			call $foo(l %t3)
			ret
		}
		export function w $main() {
		@start
			call $foo(l 42)
			call $bar(l 42)
			ret 0
		}
	`)
}

func TestStringLiteral(t *testing.T) {
	testCompile(t, `
		fn puts(s [I8]) I32
		pub fn main() I32 {
			_ = puts("str0")
			_ = puts("str0")
			_ = puts("str1")
			_ = puts("str1")
			_ = puts("str2")
			_ = puts("str2")
			_ = puts("\e\n\r\t\\\"")
			_ = puts("\x00\xab\xff")
			_ = puts("\u0100\U00010000")
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
			%t7 =w call $puts(l $str3)
			%t8 =w call $puts(l $str4)
			%t9 =w call $puts(l $str5)
			ret 0
		}
		data $str0 = { b "str0", b 0 }
		data $str1 = { b "str1", b 0 }
		data $str2 = { b "str2", b 0 }
		data $str3 = { b 27, b 10, b 13, b 9, b "\"", b 0 }
		data $str4 = { b 0, b 171, b 255, b 0 }
		data $str5 = { b 196, b 128, b 240, b 144, b 128, b 128, b 0 }
	`)
}

func TestRuneLiteral(t *testing.T) {
	testMainCompile(t, `
		var r I32
		r = 'a'
		r = '\e'
		r = '\n'
		r = '\r'
		r = '\t'
		r = '\\'
		r = '\''
	`, `
		%t1 =l alloc4 4
		storew 0, %t1
		storew 97, %t1
		storew 27, %t1
		storew 10, %t1
		storew 13, %t1
		storew 9, %t1
		storew 92, %t1
		storew 39, %t1
	`)
}

func TestTypeCheck(t *testing.T) {
	testCompileFailure(t, "Expression returning non-void cannot be used as statement", `
		fn f() {
			4 + 2
		}
	`)

	testCompileFailure(t, "Type error in call to f: [I8] is not I32", `
		fn f(x I32)
		fn g() {
			f("")
		}
	`)
}
