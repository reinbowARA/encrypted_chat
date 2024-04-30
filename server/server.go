package main

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"flag"
	"fmt"
	"io"
	"log"
	"math/big"
	"os"
	"sync"
	"time"

	"github.com/quic-go/quic-go"
)
// this is Ilon Mask
func main() {
	hostName := flag.String("hostname", "localhost", "hostname/ip of the server")
	portNum := flag.String("port", "4242", "port number of the server")

	flag.Parse()

	addr := *hostName + ":" + *portNum

	log.Println("Server running @", addr)

	listener, err := quic.ListenAddr(addr, generateTLSConfig(), nil)
	if err != nil {
		log.Fatal(err)
	}
	defer listener.Close()

	messageCh := make(chan []byte)

	go broadcastMessages(messageCh)

	for {
		sess, err := listener.Accept(context.Background())
		if err != nil {
			log.Println(err)
			continue
		}
		fmt.Printf("Пользователь %s подключился к серверу\n",listener.Addr().String())
		go handleSession(sess, messageCh)
	}
}

func handleSession(sess quic.Connection, messageCh chan<- []byte) {
	defer sess.CloseWithError(0, "")
	defer log.Println("Session closed")

	stream, err := sess.AcceptStream(context.Background())
	if err != nil {
		log.Println(err)
		return
	}
	defer stream.Close()

	log.Println("Stream opened")

	// Add the client to the list of clients
	clientMu.Lock()
	clients[stream] = struct{}{}
	clientMu.Unlock()

	// Read from the stream and send messages to all clients
	for {
		buf := make([]byte, 1024)
		n, err := stream.Read(buf)
		if err != nil {
			if err == io.EOF {
				log.Println("Client disconnected")
				return
			}
			log.Println(err)
			return
		}

		message := buf[:n]
		log.Printf("Received message: %s\n", message)

		// Send the received message to all clients
		messageCh <- message
		os.Stdin.WriteString("")
	}
}

func broadcastMessages(messageCh <-chan []byte) {
	for {
		select {
		case message := <-messageCh:
			clientMu.Lock()
			for client := range clients {
				if _, err := client.Write(message); err != nil {
					log.Println("Error writing message to client:", err)
				}

			}
			clientMu.Unlock()
		}
	}
}

var (
	clients  = make(map[quic.Stream]struct{})
	clientMu sync.Mutex
)

// Setup a bare-bones TLS config for the server
func generateTLSConfig() *tls.Config {
	key, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		log.Fatal(err)
	}
	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:     x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
	}
	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	if err != nil {
		log.Fatal(err)
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})

	tlsCert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		log.Fatal(err)
	}
	return &tls.Config{
		Certificates: []tls.Certificate{tlsCert},
		NextProtos:   []string{"quic-echo"},
	}
}
