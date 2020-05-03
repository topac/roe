package roe

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// errBasenameNotValid is an error used by newSplittedName when the given file
// does not resemple an encrypted .bmp image that is part of larger original file.
var errBasenameNotValid = fmt.Errorf("not a splitted file")

type splitName struct {
	base  string
	index int
	count int
}

func (s splitName) String() string {
	return encryptedFilename(s.base, s.index, s.count)
}

func newSplittedName(fp string) (*splitName, error) {
	base := filepath.Base(fp)

	if !HasBmpExt(fp) {
		return nil, errBasenameNotValid
	}

	r := regexp.MustCompile("^(.+)\\.(\\d+)-(\\d+)\\.bmp$")
	parts := r.FindAllStringSubmatch(base, 3)

	if len(parts) != 1 || len(parts[0]) != 4 {
		return nil, errBasenameNotValid
	}

	index, err := strconv.Atoi(parts[0][2])
	if err != nil {
		return nil, errBasenameNotValid
	}
	index--

	count, err := strconv.Atoi(parts[0][3])
	if err != nil {
		return nil, errBasenameNotValid
	}

	if index >= count || index < 0 {
		return nil, errBasenameNotValid
	}

	return &splitName{
		base:  parts[0][1],
		index: index,
		count: count,
	}, nil
}

// isSplittedName returns true if the given filename ends with "{n}-{total}.bmp"
// where {n} and {total} are numbers, for e.g. "/tmp/foobar.2-10.bmp".
// {n} must be less than or equal to {total} and cannot be less than or equal to zero.
func isSplittedName(fp string) bool {
	_, err := newSplittedName(fp)
	return err == nil
}

// DecryptedFilename returns the filename that will be used for the decrypted (the original) version of a file
func DecryptedFilename(fp string) string {
	if !HasBmpExt(fp) {
		log.Fatal("expected .bmp ext")
	}

	base := filepath.Base(fp)

	if isSplittedName(fp) {
		parts := strings.Split(base, ".")
		return strings.Join(parts[0:len(parts)-2], ".")
	}

	return base[0 : len(base)-4]
}

// HasBmpExt returns true when the given filename ends with .bmp
func HasBmpExt(fp string) bool {
	return strings.EqualFold(filepath.Ext(fp), ".bmp")
}

func encryptedFilename(base string, index, count int) string {
	if count == 1 {
		return fmt.Sprintf("%s.bmp", base)
	}
	return fmt.Sprintf("%s.%d-%d.bmp", base, index+1, count)
}

func findSplitNames(fp string) ([]splitName, error) {
	ary := make([]splitName, 0)

	// get the slitName of fp
	sn, err := newSplittedName(fp)
	if err != nil {
		return ary, fmt.Errorf("'%s' is not valid: %v", fp, err)
	}

	// find the other parts in the same folder
	files, err := ioutil.ReadDir(filepath.Dir(fp))
	if err != nil {
		return ary, err
	}
	for _, f := range files {
		sn2, err := newSplittedName(f.Name())
		if err == nil && sn2.base == sn.base {
			ary = append(ary, *sn2)
		}
	}

	// verify we have all the parts
	if len(ary) == 0 {
		return ary, fmt.Errorf("there should be other parts of '%s'", fp)
	}
	if len(ary) != ary[0].count {
		return ary, fmt.Errorf("there should be %d parts of '%s', founded %d", ary[0].count, fp, len(ary))
	}

	// sort them
	for i := 0; i < len(ary); i++ {
		b1 := ary[i]
		b2 := ary[b1.index]
		ary[b1.index] = b1
		ary[i] = b2
	}
	return ary, nil
}

type byteRange struct {
	off   int64
	len   int64
	index int
}

func getByteRanges(total int64, step int64) []byteRange {
	var off, len int64
	output := make([]byteRange, 0)
	index := 0

	for off = 0; off < total; off += step {
		if off+step > total {
			len = total - off
		} else {
			len = step
		}

		output = append(output, byteRange{
			off:   off,
			len:   len,
			index: index,
		})

		index++
	}

	return output
}

// GetFileSize returns the size of fp or panic in case of error
func GetFileSize(fp string) int64 {
	info, err := os.Stat(fp)
	if err != nil {
		log.Fatal(err)
	}
	return info.Size()
}
