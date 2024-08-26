package main

import (
    amqp "github.com/rabbitmq/amqp091-go"
    "fmt"
    "log"
    //"os"
    //"os/signal"
    "github.com/navivan123/learn-pub-sub-starter/internal/pubsub"
    "github.com/navivan123/learn-pub-sub-starter/internal/routing"
    "github.com/navivan123/learn-pub-sub-starter/internal/gamelogic"
)

func main() {
	gamelogic.PrintServerHelp()	
	const conPrt = "5672"
	const conStr = "amqp://guest:guest@localhost:" + conPrt + "/"

	con, err := amqp.Dial(conStr)
	if err != nil {
		log.Fatalf("Could not connect to Rabbit %v", err)
	}
	
	defer con.Close()
	fmt.Println("Connection Succesful!")
        
	// Begin Assignment:
	ch, err := con.Channel()
	if err != nil {
		log.Fatalf("Could not create a channel to Rabbit %v", err)
	}
	
	err = pubsub.SubscribeGob(con, routing.ExchangePerilTopic, routing.GameLogSlug, routing.GameLogSlug + ".*", pubsub.SimpleQueueDurable, handlerLogs(),)  
	if err != nil {
		log.Fatalf("Could not consume logs %v", err)
	}
	
	gamelogic.PrintServerHelp()

	for {
                menuOption := gamelogic.GetInput()
		if len(menuOption) == 0 {
			continue
		}
		
		if menuOption[0] == "pause" {
			err = pubsub.PublishJSON(ch, routing.ExchangePerilDirect, routing.PauseKey,  routing.PlayingState{IsPaused: true,})
			if err!= nil {
				log.Fatalf("Could not Publish a JSON message using PublishJSON %v", err)
			}
		
		} else if menuOption[0] == "resume" {
			err = pubsub.PublishJSON(ch, routing.ExchangePerilDirect, routing.PauseKey,  routing.PlayingState{IsPaused: false,})
			if err!= nil {
				log.Fatalf("Could not Publish a JSON message using PublishJSON %v", err)
			}
		
		} else if menuOption[0] == "quit" {
			fmt.Println("Exiting!")
			break
		
		} else {
			fmt.Println("Command not understood")
		}
	}
}

func handlerLogs() func(gamelog routing.GameLog) pubsub.Acktype {
	return func(gamelog routing.GameLog) pubsub.Acktype {
		defer fmt.Print("> ")

		err := gamelogic.WriteLog(gamelog)
		if err != nil {
			fmt.Printf("error writing log: %v\n", err)
			return pubsub.NackRequeue
		}
		return pubsub.Ack
	}
}
