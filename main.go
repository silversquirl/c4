package main

import "fmt"

func main() {
	prog := `
	// Very amazing program to sum two inputted numbers
	pub fn main() I32 {
		puts("Enter two numbers:")

		var a, b I32
		scanf("%d", &a)
		scanf("%d", &b)

		printf("%d + %d = %d\n", a, b, a+b)
		return 0
	}
	`

	toks := make(chan Token)
	go Tokenize(prog, toks)
	for tok := range toks {
		fmt.Println(tok)
	}
}
