package main

import (
        amqp "github.com/rabbitmq/amqp091-go"
	"github.com/navivan123/learn-pub-sub-starter/internal/gamelogic"
	"github.com/navivan123/learn-pub-sub-starter/internal/pubsub"
	"github.com/navivan123/learn-pub-sub-starter/routing"
	"fmt"
	"log"
)

func main() {
	fmt.Println("Starting Peril client...")
	
	const conPrt = "5672"
	const conStr = "amqp://guest:guest@localhost:" + conPrt + "/"

	con, err := amqp.Dial(conStr)
	if err != nil {
		log.Fatalf("Could not connect to Rabbit %v", err)
	}
	
	defer con.Close()
	fmt.Println("Connection Succesful!")

	usr, err := gamelogic.ClientWelcome()
	if err != nil {
		log.Fatalf("Failure getting username! %v", err)
	}
	
	ch, que, err := DeclareAndBind(con, "peril_direct", "pause." + usr, "pause", pubsub.Transient)  
	if err != nil {
		log.Fatalf("Could not declare and bind queue to Rabbit %v", err)
	}
	defer ch.Close()

	fmt.Printf("Queue %s has been declared and bound.\n", que.Name)
}
