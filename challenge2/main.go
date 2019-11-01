package main

import (
	"crypto/rand"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"

	"golang.org/x/crypto/nacl/box"
)

// A SecureReader reads and decrypts encrypted messages.
type SecureReader struct {
	io.Reader
	priv *[32]byte
	pub  *[32]byte
}

// NewSecureReader creates a new SecureReader.
func NewSecureReader(r io.Reader, priv *[32]byte, pub *[32]byte) io.Reader {
	return &SecureReader{r, priv, pub}
}

// Read will read the given encrypted message and attempt to decrypt it
func (sr *SecureReader) Read(message []byte) (int, error) {
	var nonce [24]byte

	readSize, err := io.ReadFull(sr.Reader, nonce[:])
	if err != nil {
		return readSize, fmt.Errorf("read nonce: %w", err)
	}

	readerMessage := make([]byte, len(message)+box.Overhead)
	readSize, err = sr.Reader.Read(readerMessage)
	if err != nil {
		return readSize, fmt.Errorf("read message: %w", err)
	}

	dec, ok := box.Open(message[:0], readerMessage[:readSize], &nonce, sr.pub, sr.priv)
	if !ok {
		return readSize, fmt.Errorf("open message: %w", err)
	}

	return len(dec), nil
}

// A SecureWriter writes encrypted messages.
type SecureWriter struct {
	io.Writer
	priv *[32]byte
	pub  *[32]byte
}

// NewSecureWriter creates a new SecureWriter
func NewSecureWriter(w io.Writer, priv *[32]byte, pub *[32]byte) io.Writer {
	return &SecureWriter{w, priv, pub}
}

// Write will encrypt the given bytes to the writer
func (sw *SecureWriter) Write(message []byte) (int, error) {
	var nonce [24]byte
	if _, err := io.ReadFull(rand.Reader, nonce[:]); err != nil {
		return 0, err
	}

	encryptedMessage := box.Seal(nonce[:], message, &nonce, sw.pub, sw.priv)

	if writeSize, err := sw.Writer.Write(encryptedMessage); err != nil {
		return writeSize, err
	}

	return len(message), nil
}

// Dial creates a secure connection on the given address
func Dial(addr string) (io.ReadWriteCloser, error) {
	pub, priv, err := box.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("generate key pair: %w", err)
	}

	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("dial address: %w", err)
	}

	if _, err = conn.Write(pub[:]); err != nil {
		return nil, fmt.Errorf("write public key: %w", err)
	}

	var publicKey [32]byte
	if _, err = io.ReadFull(conn, publicKey[:]); err != nil {
		return nil, fmt.Errorf("read public key: %w", err)
	}

	dialer := struct {
		io.Reader
		io.Writer
		io.Closer
	}{
		NewSecureReader(conn, priv, &publicKey),
		NewSecureWriter(conn, priv, &publicKey),
		conn,
	}

	return &dialer, nil
}

// Serve starts a secure echo server on the given listener.
func Serve(l net.Listener) error {
	pub, priv, err := box.GenerateKey(rand.Reader)
	if err != nil {
		return fmt.Errorf("generate keys: %w", err)
	}

	for {
		conn, err := l.Accept()
		if err != nil {
			return fmt.Errorf("create connection: %w", err)
		}

		go func(conn net.Conn) {
			defer conn.Close()

			if _, err := conn.Write(pub[:]); err != nil {
				log.Fatalf("writing public key: %v", err)
			}
			var publicKey [32]byte
			if _, err := io.ReadFull(conn, publicKey[:]); err != nil {
				log.Fatalf("reading public key: %v", err)
			}

			secureWriter := NewSecureWriter(conn, priv, &publicKey)
			secureReader := NewSecureReader(conn, priv, &publicKey)

			if _, err := io.Copy(secureWriter, secureReader); err != nil {
				log.Fatalf("starting echo: %v", err)
			}
		}(conn)
	}
}

func main() {
	port := flag.Int("l", 0, "Listen mode. Specify port")
	flag.Parse()

	if *port != 0 {
		l, err := net.Listen("tcp", fmt.Sprintf(":%d", *port))
		if err != nil {
			log.Fatal(err)
		}
		defer l.Close()

		log.Fatal(Serve(l))
	}

	if len(os.Args) != 3 {
		log.Fatalf("Usage: %s <port> <message>", os.Args[0])
	}

	conn, err := Dial("localhost:" + os.Args[1])
	if err != nil {
		log.Fatal(err)
	}

	if _, err := conn.Write([]byte(os.Args[2])); err != nil {
		log.Fatal(err)
	}

	buf := make([]byte, len(os.Args[2]))
	n, err := conn.Read(buf)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("%s\n", buf[:n])
}
