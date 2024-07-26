package main

import (
    amqp "github.com/rabbitmq/amqp091-go"
    "fmt"
    "log"
    "os"
    "os/signal"
    "github.com/navivan123/learn-pub-sub-starter/internal/pubsub"
    "github.com/navivan123/learn-pub-sub-starter/internal/routing"
)

func main() {
	const conPrt = "5672"
	const conStr = "amqp://guest:guest@localhost:" + conPrt + "/"

	con, err := amqp.Dial(conStr)
	if err != nil {
		log.Fatalf("Could not connect to Rabbit %v", err)
	}
	
	defer con.Close()
	fmt.Println("Connection Succesful!")
	// Begin Assignment:
	con.Channel()
	
	ch, err := con.Channel()
	if err != nil {
		log.Fatalf("Could not create a channel to Rabbit %v", err)
	}
	defer ch.Close()

	goObj := routing.PlayingState{IsPaused: true,}
	err = pubsub.PublishJSON(ch, routing.ExchangePerilDirect, routing.PauseKey,  goObj)
	if err!= nil {
		log.Fatalf("Could not Publish a JSON message using PublishJSON %v", err)
	}

	// Wait for keyboard interrupt (ctrl+c signal)
	sChan := make(chan os.Signal, 1)
	signal.Notify(sChan, os.Interrupt)
	<-sChan
}
