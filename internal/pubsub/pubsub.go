package pubsub


import (
	"context"
	"encoding/json"

	amqp "github.com/rabbitmq/amqp091-go"
)

const (
    Durable   = 1
    Transient = 0
)

func DeclareAndBind(
	con *amqp.Connection,
	exchange,
	queueName,
	key string,
	simpleQueueType int, // an enum to represent "durable" or "transient"
) (*amqp.Channel, amqp.Queue, error) {
	
	ch, err := con.Channel()
	if err != nil {
		return nil, amqp.Queue{}, err
	}

	var durable, autoDel, exclusive bool
	
	if simpleQueueType == Durable {
		durable   = true
		autoDel   = false
		exclusive = false
	} else if simpleQueueType == Transient {
		durable   = false
		autoDel   = true
		exclusive = true
	}

	que, err := ch.QueueDeclare(queueName, durable, autoDel, exclusive ,false, nil)
	if err != nil {
		ch.Close()
		return nil, amqp.Queue{}, err
	}
	
	err = ch.QueueBind(queueName, key, exchange, false, nil)
	if err != nil {
		ch.Close()
		return nil, amqp.Queue{}, err
	}

	return ch, que, nil
}

func PublishJSON[T any](ch *amqp.Channel, exchange, key string, val T) error {
	
	body, err := json.Marshal(val)
	if err != nil {
		return err
	}

	err = ch.PublishWithContext(
		context.Background(),
		exchange,
		key,
		false,
		false,
		amqp.Publishing{
			ContentType: "application/json",
			Body: body,
		},
	)
	if err != nil {
		return err
	}
	return nil
}
