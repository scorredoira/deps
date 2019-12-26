// command deps lists and executes actions on package imports.
package main

import (
	"flag"
	"fmt"
	"go/parser"
	"go/token"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"golang.org/x/tools/go/packages"
)

var stdPackages = make(map[string]bool)

func main() {
	e := flag.String("e", "", "exec command")
	p := flag.String("p", "", "pattern to match")
	v := flag.Bool("v", false, "verbose")
	t := flag.Bool("t", false, "include tests")

	flag.Usage = func() {
		fmt.Printf("Usage: %s path\n", os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()

	var path string

	switch flag.NArg() {
	case 0:
		path = "."
	case 1:
		path = flag.Arg(0)
	default:
		flag.Usage()
		return
	}

	if *e == "" {
		// if no command is passed then just print the packages
		*v = true
	}

	abs, err := filepath.Abs(path)
	if err != nil {
		log.Fatal(err)
	}

	gopath := filepath.Join(os.Getenv("GOPATH"), "src")

	loadStd()

	imps, err := Imports(abs, gopath, *t)
	if err != nil {
		log.Fatal(err)
	}

	var rx *regexp.Regexp
	if *p != "" {
		if rx, err = regexp.Compile(*p); err != nil {
			log.Fatal(err)
		}
	}

	for _, imp := range imps {
		if rx != nil && !rx.MatchString(imp) {
			continue
		}

		if *v {
			fmt.Println(imp)
		}

		if *e != "" {
			parts := strings.Split(*e, " ")
			cmd := exec.Command(parts[0], parts[1:]...)
			cmd.Dir = imp
			cmd.Env = os.Environ()

			b, err := cmd.CombinedOutput()
			if err != nil {
				fmt.Println(err)
			}
			if len(b) > 0 {
				fmt.Println(string(b))
			}
		}
	}
}

func loadStd() {
	std, err := packages.Load(nil, "std")
	if err != nil {
		panic(err)
	}

	for _, pkg := range std {
		stdPackages[pkg.PkgPath] = true
	}
}

func Imports(path, gopath string, includeTests bool) ([]string, error) {
	imports := make(map[string]bool)

	if err := getImports(path, gopath, includeTests, imports); err != nil {
		return nil, err
	}

	s := make([]string, len(imports))
	i := 0
	for k, _ := range imports {
		s[i] = k
		i++
	}

	sort.Strings(s)
	return s, nil
}

func getImports(path, gopath string, includeTests bool, imports map[string]bool) error {
	fset := token.NewFileSet()

	filter := func(f os.FileInfo) bool {
		n := f.Name()

		if strings.IndexAny(n, "._") == 0 {
			// ignore hidden files or that start with _
			return false
		}

		if !includeTests && strings.HasSuffix(n, "_test.go") {
			return false
		}
		return true
	}

	pkgs, err := parser.ParseDir(fset, path, filter, parser.ImportsOnly)
	if err != nil {
		fmt.Println(err)
		return nil
	}

	for _, pkg := range pkgs {
		for _, f := range pkg.Files {
			for _, imp := range f.Imports {
				// search only if the package exists under GOPATH
				p := strings.Trim(imp.Path.Value, `"`)

				if stdPackages[p] {
					continue
				}

				switch p {
				case "C":
					continue
				}

				if _, ok := imports[p]; !ok {
					imports[p] = true
					p = filepath.Join(gopath, p)
					if err := getImports(p, gopath, includeTests, imports); err != nil {
						return err
					}
				}

			}
		}
	}

	return nil
}
