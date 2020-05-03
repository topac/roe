package roe

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"math"
	"os"
	"path/filepath"
)

// KeyFromPassword derives a 256 bits key from the given passphrase.
func KeyFromPassword(p string) []byte {
	h := sha256.New()
	h.Write([]byte(p))
	for i := 0; i < 256; i++ {
		h.Write(h.Sum(nil))
	}
	return h.Sum(nil)
}

// randBuf fills the given buf with random bytes starting from start.
// With start = 0 the whole buf is filled.
func randBuf(buf []byte, start int) {
	_ = buf[start] // guarantee safety of writes below
	size := len(buf)
	padding := make([]byte, size-start)
	_, err := rand.Read(padding)
	if err != nil {
		log.Fatal(err)
	}
	for i := 0; i < len(padding); i++ {
		buf[start+i] = padding[i]
	}
}

func decryptSplittedFile(srcpath string, outdir string, key []byte) error {
	// search all the other parts
	names, err := findSplitNames(srcpath)
	if err != nil {
		return err
	}

	// create the new file
	base := DecryptedFilename(srcpath)
	dst, err := os.Create(filepath.Join(outdir, base))
	if err != nil {
		return err
	}
	defer dst.Close()

	for _, n := range names {
		fp := filepath.Join(filepath.Dir(srcpath), n.String())
		src, err := os.Open(fp)
		if err != nil {
			return err
		}
		log.Printf("decrypt %s -> %s\n", fp, dst.Name())
		if err := decrypt(src, dst, key); err != nil {
			src.Close()
			os.Remove(dst.Name())
			return fmt.Errorf("failed to decrypt '%s': %v", fp, err)
		}
		src.Close()
	}

	return nil
}

// DecryptFile decrypts the given .bmp file into outdir.
// If the .bmp file is part of a larger original file,
// DecryptFile automatically searches for all the other parts
// in order to combine them.
func DecryptFile(srcpath string, outdir string, key []byte) error {
	if isSplittedName(srcpath) {
		return decryptSplittedFile(srcpath, outdir, key)
	}

	// open the src file
	src, err := os.Open(srcpath)
	if err != nil {
		return err
	}
	defer src.Close()

	// contruct the absolute destination path
	base := DecryptedFilename(srcpath)
	dst, err := os.Create(filepath.Join(outdir, base))
	if err != nil {
		return err
	}
	defer dst.Close()

	// write decrypted data
	log.Printf("decrypt %s -> %s\n", srcpath, dst.Name())
	if err := decrypt(src, dst, key); err != nil {
		os.Remove(dst.Name())
		return fmt.Errorf("failed to decrypt '%s': %v", srcpath, err)
	}

	return nil
}

// DecryptDir walks srcdir and calls DecryptFile on each file.
func DecryptDir(srcdir string, outdir string, key []byte) error {
	// dict is used to avoid decrypting twice the same file, for e.g.
	// when Input is []string{"foo.mp4.1-3.bmp", "foo.mp4.2-3.bmp", "foo.mp4.3-3.bmp"}
	// no matter what file is used as arg, DecryptFile is going to generate
	// always the same file: "foo.mp4"
	dict := make(map[string]bool)

	walkFn := func(fp string, fi os.FileInfo, err error) error {
		if err != nil || fi.IsDir() || fi.Size() == 0 {
			return nil
		}
		rel, err := filepath.Rel(srcdir, filepath.Dir(fp))
		if err != nil {
			return err
		}

		reloutdir := filepath.Join(outdir, rel)

		dp := filepath.Join(reloutdir, DecryptedFilename(fi.Name()))
		if dict[dp] {
			return nil
		}
		dict[dp] = true

		if err := os.MkdirAll(reloutdir, os.ModePerm); err != nil {
			return err
		}
		return DecryptFile(fp, reloutdir, key)
	}

	return filepath.Walk(srcdir, walkFn)
}

// EncryptDir walks srcdir and calls EncryptFile on each file.
func EncryptDir(srcdir string, outdir string, key []byte, split int) error {
	walkFn := func(fp string, fi os.FileInfo, err error) error {
		if err != nil || fi.IsDir() || fi.Size() == 0 {
			return nil
		}
		rel, err := filepath.Rel(srcdir, filepath.Dir(fp))
		if err != nil {
			return err
		}
		return EncryptFile(fp, filepath.Join(outdir, rel), key, split)
	}

	return filepath.Walk(srcdir, walkFn)
}

// EncryptFile encrypts the given file into outdir, writing a new valid .bmp image.
// Empty files are ignored.
func EncryptFile(src string, outdir string, key []byte, split int) error {
	f, err := os.Open(src)
	if err != nil {
		return err
	}
	defer f.Close()

	// eventually split the file into many; each file will be a valid .bmp image
	list := getByteRanges(GetFileSize(src), int64(split))

	for _, r := range list {
		// create the destination file
		dstfile := filepath.Join(outdir, encryptedFilename(filepath.Base(src), r.index, len(list)))
		os.MkdirAll(filepath.Dir(dstfile), os.ModePerm)
		dst, err := os.Create(dstfile)
		if err != nil {
			return err
		}

		// write the encrypted data
		log.Printf("encrypt %s -> %s (%d bytes)\n", src, dstfile, r.len)
		if err := encrypt(io.NewSectionReader(f, r.off, r.len), dst, key, int(r.len)); err != nil {
			dst.Close()
			return err
		}
		dst.Close()
	}

	return nil
}

func encrypt(src io.Reader, dst io.Writer, key []byte, clearsize int) error {
	// iv + clearsize + data (padded) + hash
	encsize := 16 + 16 + clearsize + (16 - clearsize%16) + 32

	// prepare the cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return err
	}

	// init the sha256 hash digest that will be appended at the end
	// of the encrypted payload
	h := sha256.New()

	// write the bitmap header
	dim := int(math.Ceil(math.Sqrt(float64(encsize) / 4.0)))
	bmpHeader := newBmpHeader(dim, dim)
	binary.Write(dst, binary.LittleEndian, bmpHeader)

	// allocate 2 buffers of 16 bytes
	buf := make([]byte, aes.BlockSize)
	encBuf := make([]byte, aes.BlockSize)

	// get a random IV and write it
	randBuf(buf, 0)
	mode := cipher.NewCBCEncrypter(block, buf)
	dst.Write(buf)

	// write the filesize
	randBuf(buf, 0)
	binary.LittleEndian.PutUint32(buf, uint32(clearsize))
	mode.CryptBlocks(encBuf, buf)
	dst.Write(encBuf)

	// write the rest of the data
	readed := 0
	for {
		n, err := src.Read(buf)

		if err != nil && err != io.EOF {
			return err
		}

		if err == io.EOF && n == 0 {
			break
		}

		if n == 0 {
			continue
		}

		readed += n

		if n < len(buf) && readed != clearsize {
			return fmt.Errorf("failed to read %d bytes", len(buf))
		}

		if n < len(buf) {
			randBuf(buf, n)
		}

		h.Write(buf[0:n])
		mode.CryptBlocks(encBuf, buf)
		dst.Write(encBuf)
	}

	// write the sha256 hash (32 bytes)
	dst.Write(h.Sum(nil))

	// write the remaining bytes to fill the bmp data-section with random bytes
	if left := int(bmpHeader.ImageSize) - encsize; left > 0 {
		_, err := io.CopyN(dst, rand.Reader, int64(left))
		if err != nil {
			return err
		}
	}

	return nil
}

func decrypt(src io.Reader, dst io.Writer, key []byte) error {
	block, err := aes.NewCipher(key)
	if err != nil {
		return err
	}

	// init the sha256 hash digest that will be used to verify if decryption is ok
	h := sha256.New()

	// allocate 2 buffers of 16 bytes
	buf := make([]byte, aes.BlockSize)
	clearBuf := make([]byte, aes.BlockSize)

	// read the bitmap header
	hBuf := make([]byte, 54)
	src.Read(hBuf)

	// read the iv and initialize the cipher
	n, err := src.Read(buf)
	if err != nil {
		return err
	}
	if n != len(buf) {
		return fmt.Errorf("failed to read %d bytes", len(buf))
	}
	mode := cipher.NewCBCDecrypter(block, buf)

	// decrypt the filesize
	src.Read(buf)
	mode.CryptBlocks(clearBuf, buf)
	clearsize := int(binary.LittleEndian.Uint32(clearBuf))

	// decrypt the data-seciton
	written := 0
	for {
		if clearsize == 0 {
			break
		}

		n, err := src.Read(buf)

		if err != nil && err != io.EOF {
			return err
		}

		if err == io.EOF && n == 0 {
			break
		}

		if n != len(buf) {
			return fmt.Errorf("failed to read %d bytes, %d bytes were written", len(buf), written)
		}

		mode.CryptBlocks(clearBuf, buf)
		written += n

		if written >= clearsize {
			n = len(buf) - (written - clearsize)
			dst.Write(clearBuf[0:n])
			h.Write(clearBuf[0:n])
			break
		} else {
			dst.Write(clearBuf)
			h.Write(clearBuf[0:n])
		}
	}

	// compare the checksum
	expectedHash := make([]byte, 32)
	src.Read(expectedHash)
	if !bytes.Equal(expectedHash, h.Sum(nil)) {
		return fmt.Errorf("sha256 hash check failed")
	}
	return nil
}
