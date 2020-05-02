package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/topac/roe/pkg/roe"
)

func fatalf(err error) {
	if err == nil {
		os.Exit(0)
	}
	fmt.Printf("ERROR: %v\n", err)
	os.Exit(1)
}

func main() {
	opts, err := StartCLI()

	if err != nil {
		fatalf(err)
	}

	key := roe.KeyFromPassword(opts.Password)

	if opts.Encrypt {
		if opts.InputDir != "" {
			fatalf(roe.EncryptDir(opts.InputDir, opts.Outdir, key, opts.Split))
		}

		for _, input := range opts.Input {
			if roe.GetFileSize(input) == 0 {
				continue
			}
			if err := roe.EncryptFile(input, opts.Outdir, key, opts.Split); err != nil {
				fatalf(err)
			}
		}
	}

	if opts.Decrypt {
		if opts.InputDir != "" {
			fatalf(roe.DecryptDir(opts.InputDir, opts.Outdir, key))
		}

		// dict is used to avoid decrypting twice the same file, for e.g.
		// when Input is []string{"foo.mp4.1-3.bmp", "foo.mp4.2-3.bmp", "foo.mp4.3-3.bmp"}
		// no matter what file is used as arg, DecryptFile is going to generate
		// always the same file: "foo.mp4"
		dict := make(map[string]bool)

		for _, input := range opts.Input {
			if roe.GetFileSize(input) == 0 {
				continue
			}

			dp := filepath.Join(opts.Outdir, roe.DecryptedFilename(input))
			if dict[dp] {
				continue
			}
			dict[dp] = true

			if err := roe.DecryptFile(input, opts.Outdir, key); err != nil {
				fatalf(err)
			}
		}
	}
}
