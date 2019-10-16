package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/howeyc/gopass"
)

const splitDefVal = 24000000

// CLIOpts describes all possible arguments and options available in the command-line interface.
type CLIOpts struct {
	Input     []string
	InputDir  string
	Outdir    string
	Encrypt   bool
	Decrypt   bool
	Recursive bool
	Password  string
	Split     int
}

// StartCLI init the command line interface, returning Opts and any validation errors of the Opts.
// When error is not nil, the Opts are not valid and the program should not rely on them.
func StartCLI() (CLIOpts, error) {
	var outdir string
	var encrypt, decrypt, recursive bool
	var password string
	var split int

	flag.StringVar(&outdir, "outdir", ".", "Output directory")
	flag.StringVar(&password, "p", "", "Password")
	flag.BoolVar(&encrypt, "encrypt", false, "Encrypt mode")
	flag.BoolVar(&decrypt, "decrypt", false, "Decrypt mode")
	flag.BoolVar(&recursive, "recursive", false, "Traverse directories recursively")
	flag.IntVar(&split, "split", splitDefVal, "Split every N bytes")
	setUsage(flag.CommandLine)
	flag.Parse()

	opts := CLIOpts{
		Input:     flag.Args(),
		Outdir:    outdir,
		Encrypt:   encrypt,
		Decrypt:   decrypt,
		Recursive: recursive,
		Password:  password,
		Split:     split,
	}

	return opts, validate(&opts)
}

func validate(opts *CLIOpts) error {
	// validate -encrypt, -decrypt flags
	if opts.Decrypt == false && opts.Encrypt == false {
		return fmt.Errorf("unknown action, choose between -decrypt or -encrypt flags")
	}
	if opts.Decrypt && opts.Encrypt {
		return fmt.Errorf("-decrypt and -encrypt flags are mutually exclusive")
	}

	// validate -output flag
	output, err := absPath(opts.Outdir)
	if err != nil {
		return fmt.Errorf("-outdir flag is invalid: %s", err)
	}
	opts.Outdir = output

	// ensure at least an input file is given
	if len(opts.Input) == 0 {
		return fmt.Errorf("invalid usage, the last arg should be the input file(s)")
	}

	// validate input files
	for _, item := range opts.Input {
		stat, err := os.Stat(item)

		if err != nil {
			return fmt.Errorf("'%s' cannot be supplied as input: %v", item, err)
		}

		if stat.IsDir() {
			if opts.Recursive == false {
				return fmt.Errorf("'%s' cannot be supplied as input because is a directory, use -recursive flag", item)
			}
			if len(opts.Input) != 1 {
				return fmt.Errorf("only one input is allowed with -recursive flag")
			}
			a, _ := absPath(item)
			// verify that outdir and input are not the same folder
			if a == opts.Outdir {
				return fmt.Errorf("'%s' cannot be supplied both as input and to -outdir", item)
			}
			// verify that outdir is not included in this input
			r, _ := filepath.Rel(a, opts.Outdir)
			if strings.Split(r, string(filepath.Separator))[0] != ".." {
				return fmt.Errorf("-outdir is invalid because is a sub-directory of the input '%s'", a)
			}
			opts.InputDir = item
		}
	}

	// validate -slipt flag
	if opts.Split < 1000000 {
		return fmt.Errorf("-split flag is invalid: cannot be less than 1MB")
	}
	if opts.Split != splitDefVal && opts.Decrypt {
		return fmt.Errorf("-split flag is accepted only with -encrypt")
	}

	// read the password
	if opts.Password == "" {
		readPasswordLoop(&opts.Password)
	}

	return nil
}

func readPasswordLoop(password *string) {
	for {
		fmt.Printf("Type the password: ")
		pwd, err := gopass.GetPasswd()
		if err != nil {
			os.Exit(1)
		}
		if len(pwd) == 0 {
			continue
		}

		fmt.Printf("Confirm the password: ")
		pwd2, err := gopass.GetPasswd()
		if err != nil {
			os.Exit(1)
		}
		if bytes.Compare(pwd, pwd2) != 0 {
			fmt.Printf("Error: Passwords don't match\n\n")
			continue
		}
		*password = string(pwd)
		break
	}
}

func setUsage(f *flag.FlagSet) {
	f.Usage = func() {
		exe := path.Base(os.Args[0])
		fmt.Printf("Usage: %s [options] input\n", exe)
		fmt.Println("\nExamples:")
		fmt.Printf("  %s -encrypt -outdir /tmp/ jazz.mp3\n", exe)
		fmt.Printf("  %s -encrypt *.pdf\n", exe)
		fmt.Printf("  %s -encrypt -recursive -outdir /tmp/ /home/John/Movies\n", exe)
		fmt.Printf("  %s -decrypt invoice.pdf.bmp\n", exe)
		fmt.Printf("  %s -decrypt -recursive -outdir /tmp/ /home/John/Cloud\n", exe)
		fmt.Println("\nOptions:")
		flag.PrintDefaults()
		os.Exit(2)
	}
}

func absPath(dir string) (string, error) {
	stat, err := os.Stat(dir)
	if err != nil {
		return "", err
	}
	if !stat.IsDir() {
		return "", fmt.Errorf("not a directory")
	}

	if filepath.IsAbs(dir) {
		return filepath.Clean(dir), nil
	}

	s, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}
	return filepath.Clean(s), nil
}
