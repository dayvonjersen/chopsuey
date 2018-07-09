package irc

import (
	"bufio"
	"io"
	"log"
	"net"
)

func Dial(addr string) (send, recv chan string, done chan struct{}) {

	done = make(chan struct{})
	send, recv = make(chan string), make(chan string)

	conn, err := net.Dial("tcp", addr)
	if err != nil {
		log.Fatal(err)
	}
	scanner := bufio.NewScanner(conn)
	go func() {
		for scanner.Scan() {
			recv <- scanner.Text()
		}
		select {
		case <-done:
			return
		default:
			log.Println("got EOF, initializing shutdown")
			close(done)
		}
	}()
	go func() {
		for {
			select {
			case msg := <-send:
				log.Println("<-", msg)
				io.WriteString(conn, msg+"\r\n")
			case <-done:
				log.Println("sender is exiting")
				close(send)
				return
			}
		}
	}()
	go func() {
		<-done
		log.Println("we get signal (shutting down...)")
		log.Println("waiting for sender to close")
		for {
			_, ok := <-send
			if !ok {
				break
			}
		}
		log.Println("closing connection")
		conn.Close()
		log.Println("done.")
		return
	}()
	return
}
