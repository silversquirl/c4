package main

import (
	"fmt"
	"os"
)

func main() {
	prog, err := Parse(`
	// Very amazing program to sum two inputted numbers
	fn puts(str [I8]) I32
	fn scanf(fmt [I8], i [I32]) I32
	fn printf(fmt [I8], a, b, c I32) I32
	pub fn main() I32 {
		puts("Enter two numbers:")

		var a, b I32
		scanf("%d", &a)
		scanf("%d", &b)

		var c I32
		c = a + b

		printf("%d + %d = %d\n", a, b, c)
		return 0
	}
	`)

	if err != nil {
		fmt.Println("Parse error:", err)
	}

	NewCompiler(os.Stdout).Compile(prog)
}
