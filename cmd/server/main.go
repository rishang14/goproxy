package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"strings"
	"time"

	"github.com/rishang14/goproxy/internal/httpx"
)

func main() {
	listener, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Fatal("Error listening", err)
	}
	defer listener.Close()

	fmt.Println("TCP Server started at port:8080")

	for {
		conn, err := listener.Accept()

		if err != nil {
			log.Println("Error while connecting", err)
			continue
		}

		go handleConnection(conn)

	}
}

// it will accept the input from the incoming request parse it  and set the header and all and  parse it
// and it will also read the content body and everythig
func handleConnection(conn net.Conn) {

	defer conn.Close()

	reader := bufio.NewReader(conn)
	for {
		err := conn.SetReadDeadline(time.Now().Add(10 * time.Second))
		if err != nil {
			log.Printf("Timeout Error : %s", err)
			return
		}
		requestLine, err := reader.ReadString('\n')

		if err != nil {
			log.Printf("Reading error %v", err)
			return
		}

		method, target, version, err := httpx.ParseRequest(requestLine)
		if err != nil {
			httpx.WriteErrorFor(conn, err)
			return
		}
		fmt.Printf("Method is %s  target is %s  version is %s", method, target, version)
		for {
			header, err := reader.ReadString('\n')
			if err != nil {
				log.Printf("Reading error %v", err)
				return
			}
			if strings.TrimSpace(header) == "" {
				fmt.Printf("End of the header \n")
				_, err = conn.Write([]byte(method))

				if err != nil {
					log.Printf("Server write error: %v", err)
				}
				break
			}
			fmt.Println("Rest of the header", strings.TrimSpace(header))
		}
	}

}
