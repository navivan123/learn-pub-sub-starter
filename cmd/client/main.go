package main

import (
        amqp "github.com/rabbitmq/amqp091-go"
	"github.com/navivan123/learn-pub-sub-starter/internal/gamelogic"
	"github.com/navivan123/learn-pub-sub-starter/internal/pubsub"
	"github.com/navivan123/learn-pub-sub-starter/internal/routing"
    	//"os"
    	//"os/signal"
	"fmt"
        "time"
	"log"
        "strconv"
)


func handlerMove(gs *gamelogic.GameState, ch *amqp.Channel) func(gamelogic.ArmyMove) pubsub.Acktype {
        return func(move gamelogic.ArmyMove) pubsub.Acktype {
                
                defer fmt.Printf("> ")
                outcome := gs.HandleMove(move)
                
                if outcome == gamelogic.MoveOutComeSafe  || outcome == gamelogic.MoveOutcomeSamePlayer{
                        return pubsub.Ack
                }

                if outcome == gamelogic.MoveOutcomeMakeWar {
                        err := pubsub.PublishJSON(ch, routing.ExchangePerilTopic, routing.WarRecognitionsPrefix+"."+gs.GetUsername(), gamelogic.RecognitionOfWar{
					                                                                                              Attacker: move.Player,
					                                                                                              Defender: gs.GetPlayerSnap(),},)
			if err != nil {
				log.Fatalf("Could not Publish a JSON message using PublishJSON %v", err)
                                return pubsub.NackRequeue
			}
                        return pubsub.Ack
                }

                fmt.Println("Error, Unknown Move Command!")
                return pubsub.NackDiscard
        }
}


func handlerPause(gs *gamelogic.GameState) func(routing.PlayingState) pubsub.Acktype {
        return func(state routing.PlayingState) pubsub.Acktype {
                
                defer fmt.Printf("> ")
                gs.HandlePause(state)
                return pubsub.Ack
        }
}

func handlerWar(gs *gamelogic.GameState, ch *amqp.Channel) func(gamelogic.RecognitionOfWar) pubsub.Acktype {
        return func(rw gamelogic.RecognitionOfWar) pubsub.Acktype {
                
                defer fmt.Printf("> ")
                outcome, winner, loser := gs.HandleWar(rw)
                
                if outcome == gamelogic.WarOutcomeNotInvolved {
                        fmt.Println("Not involved in the war!")
                        return pubsub.NackRequeue

                } else if outcome == gamelogic.WarOutcomeNoUnits {
                        fmt.Println("No units left!")
                        return pubsub.NackDiscard
                
                } else if outcome == gamelogic.WarOutcomeOpponentWon || outcome == gamelogic.WarOutcomeYouWon {
                        fmt.Println("Debug: WarOutcomeOpponentWon or WarOutcomeYouWon")
                        err := publishGameLog(ch, gs.GetUsername(), fmt.Sprintf("War complete! Winner: %s | Loser: %s", winner, loser))
                        if err != nil {
                                fmt.Printf("Error\n", err)
                                return pubsub.NackRequeue
                        }
                        return pubsub.Ack
                } else if outcome == gamelogic.WarOutcomeDraw {
                        err := publishGameLog(ch, gs.GetUsername(), fmt.Sprintf("So disappointing... a war between %s and %s ended in a draw.", winner, loser))
                        if err != nil {
                                fmt.Printf("Error\n", err)
                                return pubsub.NackRequeue
                        }
                        return pubsub.Ack
                }

                fmt.Println("Error, Unknown War Results!")
                return pubsub.NackDiscard
        }
}

func publishGameLog(ch *amqp.Channel, username, msg string) error {
        return pubsub.PublishGob(ch, routing.ExchangePerilTopic, routing.GameLogSlug+"."+username, routing.GameLog{ Username: username,
                                                                                                                    CurrentTime: time.Now(),
                                                                                                                    Message: msg,},)
}

func main() {
	fmt.Println("Starting Peril client...")
	
	const conPrt = "5672"
	const conStr = "amqp://guest:guest@localhost:" + conPrt + "/"

        // Dial to Rabbit
	con, err := amqp.Dial(conStr)
	if err != nil {
		log.Fatalf("Could not connect to Rabbit %v", err)
	}
	defer con.Close()
	fmt.Println("Connection Succesful!")
       
        // Create channel to publish messages
        ch, err := con.Channel()
	if err != nil {
		log.Fatalf("could not create channel: %v", err)
	}

        // Welcome user and get user name
	usr, err := gamelogic.ClientWelcome()
	if err != nil {
		log.Fatalf("Failure getting username! %v", err)
	}
	
        // Create Gamestate
	gs := gamelogic.NewGameState(usr)

        // We will Subscribe and declare the queue for pausing inside this function now!
        err = pubsub.SubscribeJSON(con, routing.ExchangePerilDirect, routing.PauseKey+"."+gs.GetUsername(), routing.PauseKey, pubsub.SimpleQueueTransient, handlerPause(gs),)
        if err != nil {
                log.Fatalf("could not subscribe to pause: %v", err)
        }

        // We will Subscribe and declare the queue for moving to army_moves.*
        err = pubsub.SubscribeJSON(con, routing.ExchangePerilTopic, routing.ArmyMovesPrefix+"."+gs.GetUsername(), routing.ArmyMovesPrefix + ".*", pubsub.SimpleQueueTransient, handlerMove(gs, ch),)
        if err != nil {
                log.Fatalf("could not subscribe to moves: %v", err)
        }
        
        // We will Subscribe and declare the queue for war exchange
        err = pubsub.SubscribeJSON(con, routing.ExchangePerilTopic, routing.WarRecognitionsPrefix, routing.WarRecognitionsPrefix + ".*", pubsub.SimpleQueueDurable, handlerWar(gs, ch),)
        if err != nil {
                log.Fatalf("could not subscribe to moves: %v", err)
        }
        
	var menuOption []string
	
	for {
		menuOption = gamelogic.GetInput()
                if len(menuOption) == 0 {
			continue
		}
		if menuOption[0] == "spawn" {
			err = gs.CommandSpawn(menuOption)
			if err != nil {
				fmt.Println("Unknown entry when spawning, please try again!")
				continue
			}
			
		} else if menuOption[0] == "move" {
			mv, err := gs.CommandMove(menuOption)
			if err != nil {
				fmt.Println("Unknown entry when moving, please try again!")
				continue	
			}
                        
                        // Publish move
			err = pubsub.PublishJSON(ch, routing.ExchangePerilTopic, routing.ArmyMovesPrefix+"."+mv.Player.Username, mv)
			if err!= nil {
				log.Fatalf("Could not Publish a JSON message using PublishJSON %v", err)
			}
		        
		} else if menuOption[0] == "status" {
			gs.CommandStatus()

		} else if menuOption[0] == "help" {
			gamelogic.PrintClientHelp()	
		
		} else if menuOption[0] == "spam" {
	                if len(menuOption) > 1 {
                                n, err := strconv.Atoi(menuOption[1])
			        if err != nil {
				        fmt.Printf("error: %s is not a valid number\n", menuOption[1])
				        continue
			        }
                                spam(ch, usr, n)
                                continue
                        }
                        fmt.Println("Not enough arguments! Usage: spam # (example: spam 100)")
		} else if menuOption[0] == "quit" {
			gamelogic.PrintQuit()
			break
		
		} else {
			fmt.Println("Dude, where's my car?")
		}

	}


	// Wait for keyboard interrupt (ctrl+c signal)
	//sChan := make(chan os.Signal, 1)
	//signal.Notify(sChan, os.Interrupt)
	//<-sChan
}

func spam(ch *amqp.Channel, username string, n int) {
        msg := ""
        for i := 0 ; i < n ; i++ {
                msg = gamelogic.GetMaliciousLog()
                err := publishGameLog(ch, username, msg)
                if err != nil {
                        fmt.Printf("Error publishing stinky log: %v", err)
                }
        }
}
