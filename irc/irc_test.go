package irc

import (
	"log"
	"strings"
	"time"
)

func Example() {
	send, recv, done := /*irc.*/ Dial("chopstick:6667")

	registered := false
	go func() {
		for {
			select {
			case <-done:
				log.Println("we get signal (main screen turn on)")
				return
			case msg := <-recv:
				log.Println("->", msg)
				if registered && strings.HasPrefix(msg, "PING") {
					send <- "PONG" + strings.TrimPrefix(msg, "PING")
				}
			}
		}
	}()

	<-time.After(time.Second)
	send <- "USER tso tso tso :hi there"
	send <- "NICK tso"
	registered = true
	<-time.After(time.Second)
	send <- "JOIN #test"
	<-time.After(time.Second * 5)
	close(done)

	select {}
}
