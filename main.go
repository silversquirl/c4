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
	irOut := flag.Bool("i", false, "output intermediate representation of the program")
	obj := flag.Bool("c", false, "output an object file")
	flag.Parse()

	if flag.NArg() < 1 {
		log.Fatal("usage")
	}

	if *out == "" {
		file := flag.Arg(0)
		file = strings.TrimSuffix(file, filepath.Ext(file))
		if *obj {
			*out = file + ".o"
		} else {
			*out = file
		}
	}

	if *obj && *irOut {
		log.Fatal("-c and -i are incompatible")
	}
	link := !(*obj || *irOut)

	if *obj && flag.NArg() > 1 {
		log.Fatal("-c requires only one source file")
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
			log.Fatal(err)
		}

		qbeR, qbeW, err := os.Pipe()
		if err != nil {
			log.Fatal(err)
		}

		asR, asW, err := os.Pipe()
		if err != nil {
			log.Fatal(err)
		}

		var objFile string
		if *obj {
			objFile = *out
		} else {
			f, err := ioutil.TempFile(tmpDir, "*.o")
			if err != nil {
				log.Fatal(err)
			}
			f.Close() // I wish I could pass the fd directly to as
			objFile = f.Name()
		}

		var qbeCmd, asCmd *exec.Cmd
		if !*irOut {
			qbeCmd = exec.Command("qbe")
			qbeCmd.Stdin = qbeR
			qbeCmd.Stdout = asW
			qbeCmd.Stderr = os.Stderr
			if err := qbeCmd.Start(); err != nil {
				log.Fatal(err)
			}

			if *obj {
				asCmd = exec.Command(*as, "-c", "-o", objFile, "-")
			} else {
				asCmd = exec.Command(*as, "-o", objFile, "-")
			}
			asCmd.Stdin = asR
			asCmd.Stdout = os.Stdout
			asCmd.Stderr = os.Stderr
			if err := asCmd.Start(); err != nil {
				log.Fatal(err)
			}
		}

		if r, err := NewCompiler().Compile(prog); err != nil {
			log.Fatal(err)
		} else if *irOut {
			r.WriteTo(os.Stdout)
		} else {
			r.WriteTo(qbeW)
		}
		qbeW.Close()
		if !*irOut {
			if err := qbeCmd.Wait(); err != nil {
				log.Fatal(err)
			}
			asW.Close()
			if err := asCmd.Wait(); err != nil {
				log.Fatal(err)
			}
		}

		objs = append(objs, objFile)
	}

	if link {
		if *verbose {
			log.Print("LD ", *out)
		}
		ldCmd := exec.Command(*ld, append(objs, "-o", *out)...)
		ldCmd.Stderr = os.Stderr
		if err := ldCmd.Run(); err != nil {
			log.Fatal(err)
		}
	}
}
