package main

import (
	"bufio"
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"time"

	"github.com/quic-go/quic-go"
)

func main() {

	hostName := flag.String("hostname", "localhost", "hostname/ip of the server")
	portNum := flag.String("port", "4242", "port number of the server")
	//numEcho := flag.Int("necho", 100, "number of echos")
	timeoutDuration := flag.Int("rtt", 1000, "timeout duration (in ms)")

	flag.Parse()

	addr := *hostName + ":" + *portNum

	tlsConf := &tls.Config{
		InsecureSkipVerify: true,
		NextProtos:         []string{"quic-echo"},
	}

	session, err := quic.DialAddr(context.Background(), addr, tlsConf, nil)
	if err != nil {
		fmt.Println(session, "\n", addr, "\n", tlsConf)
	}

	stream, err := session.OpenStreamSync(context.Background())
	if err != nil {
		panic(err)
	}

	timeout := time.Duration(*timeoutDuration) * time.Millisecond

	resp := make(chan string)

	ips, _ := GetLocalIPs()

	fmt.Printf("Client %s is running \n", ips[len(ips)-1])
	for {
		fmt.Print("Message: ")
		message, _ := bufio.NewReader(os.Stdin).ReadString('\n')
		_, err = stream.Write([]byte(message))
		if err != nil {
			panic(err)
		}

		//log.Println("Done. Waiting for echo")

		go func() {
			buff := make([]byte, len(message))
			_, err = io.ReadFull(stream, buff)
			if err != nil {
				panic(err)
			}

			resp <- string(buff)
		}()

		select {
		case reply := <-resp:
			log.Printf("Сервер прислал: %s", reply)
		case <-time.After(timeout):
			log.Printf("Client: Timed out\n")
		}

		/*if counter == *numEcho {
			break
		}*/
	}
}
// небольшая безделушка по определению IP пользователя
func GetLocalIPs() ([]net.IP, error) {
	var ips []net.IP
	addresses, err := net.InterfaceAddrs()
	if err != nil {
		return nil, err
	}
	for _, addr := range addresses {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				ips = append(ips, ipnet.IP)
			}
		}
	}
	return ips, nil
}
