package main

import (
	"flag"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func main() {
	log.SetFlags(0)

	out := flag.String("o", "", "output `filename`")
	as := flag.String("as", "as", "`name` of assembler to use")
	ld := flag.String("ld", "cc", "`name` of linker to use")
	verbose := flag.Bool("v", false, "verbose output")
	flag.Parse()

	if flag.NArg() < 1 {
		log.Fatal("usage")
	}

	if *out == "" {
		file := flag.Arg(0)
		*out = strings.TrimSuffix(file, filepath.Ext(file))
	}

	tmpDir, err := ioutil.TempDir("", "c4-build-*")
	if err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	var objs []string
	for _, file := range flag.Args() {
		if *verbose {
			log.Print("C4 ", file)
		}

		data, err := ioutil.ReadFile(file)
		prog, err := Parse(string(data))
		if err != nil {
			log.Fatalln("Parse error:", err)
		}

		qbeR, qbeW, err := os.Pipe()
		if err != nil {
			log.Fatal(err)
		}

		asR, asW, err := os.Pipe()
		if err != nil {
			log.Fatal(err)
		}

		objFile, err := ioutil.TempFile(tmpDir, "*.o")
		if err != nil {
			log.Fatal(err)
		}
		objFile.Close() // I wish I could pass the fd directly to as

		qbeCmd := exec.Command("qbe")
		qbeCmd.Stdin = qbeR
		qbeCmd.Stdout = asW
		qbeCmd.Stderr = os.Stderr
		if err := qbeCmd.Start(); err != nil {
			log.Fatal(err)
		}

		asCmd := exec.Command(*as, "-o", objFile.Name(), "-")
		asCmd.Stdin = asR
		asCmd.Stdout = objFile
		asCmd.Stderr = os.Stderr
		if err := asCmd.Start(); err != nil {
			log.Fatal(err)
		}

		if err := NewCompiler(qbeW).Compile(prog); err != nil {
			log.Fatal(err)
		}
		qbeW.Close()
		if err := qbeCmd.Wait(); err != nil {
			log.Fatal(err)
		}
		asW.Close()
		if err := asCmd.Wait(); err != nil {
			log.Fatal(err)
		}

		objs = append(objs, objFile.Name())
	}

	if *verbose {
		log.Print("LD ", *out)
	}
	ldCmd := exec.Command(*ld, append(objs, "-o", *out)...)
	ldCmd.Stderr = os.Stderr
	if err := ldCmd.Run(); err != nil {
		log.Fatal(err)
	}
}
