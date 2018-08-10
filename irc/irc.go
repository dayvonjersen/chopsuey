package irc

import (
	"bufio"
	"crypto/tls"
	"io"
	"log"
	"net"
	"os"
	"time"
)

func MockConnection(filename string) (send, recv chan string, done chan struct{}) {
	done = make(chan struct{})
	send, recv = make(chan string), make(chan string)

	go func() {
		f, err := os.Open(filename)
		if err != nil {
			log.Fatal(err)
		}
		defer f.Close()
		scanner := bufio.NewScanner(f)
		for {
			select {
			case <-send:
				// do nothing
			case <-done:
				return
			case <-time.After(time.Millisecond * 50):
				if scanner.Scan() {
					recv <- scanner.Text()
				} else {
					recv <- ":go!~gopher@golang.org QUIT :EOF (the test is over now, goodbye!)\n"
					close(done)
					return
				}
			}
		}
	}()

	return send, recv, done
}

func Dial(addr string) (send, recv chan string, done chan struct{}) {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		log.Fatal(err)
	}
	return dial(conn)
}

func DialTLS(addr string, cfg *tls.Config) (send, recv chan string, done chan struct{}) {
	if cfg == nil {
		cfg = &tls.Config{
			InsecureSkipVerify: true,
		}
	}
	conn, err := tls.Dial("tcp", addr, cfg)
	if err != nil {
		log.Fatal(err)
	}
	return dial(conn)
}

func dial(conn net.Conn) (send, recv chan string, done chan struct{}) {
	done = make(chan struct{})
	send, recv = make(chan string), make(chan string)

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
