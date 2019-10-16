package roe

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"io/ioutil"
	mrand "math/rand"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func randInt(min, max int) int {
	mrand.Seed(time.Now().UnixNano())
	return mrand.Intn(max-min+1) + min
}

func createRandomFile(fp string, size int) []byte {
	f, _ := os.Create(fp)
	cleartext := make([]byte, size)
	rand.Read(cleartext)
	f.Write(cleartext)
	f.Close()
	return cleartext
}

// encryptAndDecryptRand tests the encrypt/decrypt functions using a slice of random bytes of size n.
func encryptAndDecryptRand(n int) error {
	// encryption/decryption key
	key := KeyFromPassword("foobar")

	// cleartext, a buffer filled with random bytes
	cleartext := make([]byte, n)
	rand.Read(cleartext)

	// multipurpose buffer
	buffer := bytes.NewBuffer(make([]byte, 0))

	// encrypt
	if err := encrypt(bytes.NewReader(cleartext), buffer, key, n); err != nil {
		return err
	}

	// copy the encrypted data to enc and reset buffer
	enc := buffer.Bytes()
	buffer.Reset()

	// decrypt
	if err := decrypt(bytes.NewReader(enc), buffer, key); err != nil {
		return err
	}

	// compare
	if comp := bytes.Compare(buffer.Bytes(), cleartext); comp != 0 {
		return fmt.Errorf("comparison failed: %d", comp)
	}

	return nil
}

func Test_encryptAndDecrypt(t *testing.T) {
	clearsizes := make([]int, 0)

	for i := 0; i <= 2048; i++ {
		clearsizes = append(clearsizes, i)
	}

	for _, clearsize := range clearsizes {
		f := func(t2 *testing.T) {
			if err := encryptAndDecryptRand(clearsize); err != nil {
				t2.Error(err)
			}
		}
		t.Run(fmt.Sprintf("encryptAndDecryptRand(%d)", clearsize), f)
	}
}

func Test_encryptFileAndDecryptFile(t *testing.T) {
	// create a temporary folder and make sure it is going to be delete at the end
	tmpdir, _ := ioutil.TempDir("", "roe")
	defer os.RemoveAll(tmpdir)
	defer os.Remove(tmpdir)

	for n := 1; n <= 1024; n++ {
		// create a key
		key := KeyFromPassword(fmt.Sprintf("my secret password %d", n))

		// ensure these temporary folders exist
		cleandir := filepath.Join(tmpdir, "clean") // folder for clean files
		encdir := filepath.Join(tmpdir, "enc")     // folder for encrypted files
		decdir := filepath.Join(tmpdir, "dec")     // folder for decrypted files
		os.MkdirAll(cleandir, os.ModePerm)
		os.MkdirAll(encdir, os.ModePerm)
		os.MkdirAll(decdir, os.ModePerm)

		// create a random file of size n
		cleanpath := filepath.Join(cleandir, fmt.Sprintf("testfile.no.%d", n+1))
		clearbuf := createRandomFile(cleanpath, n)

		// select a random split value, this is going to be passed to the EncryptFile func
		split := n * 2
		if n > 80 && randInt(1, 2) == 1 {
			split = n/randInt(2, 4) + randInt(1, 10)
		}

		// call EncryptFile
		if err := EncryptFile(cleanpath, encdir, key, split); err != nil {
			t.Error(err)
			return
		}

		// call DecryptFile
		files, _ := ioutil.ReadDir(encdir)
		if len(files) == 0 {
			t.Errorf("EncryptFile failed to create a file, %d files founded in %v", len(files), encdir)
			return
		}
		encpath := filepath.Join(encdir, files[0].Name())
		if err := DecryptFile(encpath, decdir, key); err != nil {
			t.Error(err)
			return
		}

		// verify that the decrypted file is equal to the original file
		files, _ = ioutil.ReadDir(decdir)
		if len(files) != 1 {
			t.Errorf("DecryptFile failed to create a file, %d files founded in %v", len(files), decdir)
			return
		}
		decpath := filepath.Join(decdir, files[0].Name())
		decbuf, _ := ioutil.ReadFile(decpath)
		if bytes.Compare(clearbuf, decbuf) != 0 {
			t.Errorf("decrypted file and original file differs")
			return
		}

		// clean up the temporary folder
		os.RemoveAll(tmpdir)
	}
}
