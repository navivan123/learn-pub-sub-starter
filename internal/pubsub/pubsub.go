package pubsub


import (
	"context"
	"encoding/json"
        "fmt"
	amqp "github.com/rabbitmq/amqp091-go"
        "encoding/gob"
        "bytes"
)

const (
    Durable   = 1
    Transient = 0
)

type Acktype int

type SimpleQueueType int

const (
	SimpleQueueDurable SimpleQueueType = iota
	SimpleQueueTransient
)

const (
        Ack Acktype = iota
        NackRequeue 
        NackDiscard 
)

func DeclareAndBind(
	con *amqp.Connection,
	exchange,
	queueName,
	key string,
	simpleQueueType SimpleQueueType,
) (*amqp.Channel, amqp.Queue, error) {
	
	ch, err := con.Channel()
	if err != nil {
                return nil, amqp.Queue{}, fmt.Errorf("Couldn't create channel: %v", err)
	}

	que, err := ch.QueueDeclare(queueName, 
                                    simpleQueueType == SimpleQueueDurable,
                                    simpleQueueType != SimpleQueueDurable,
                                    simpleQueueType != SimpleQueueDurable,
                                    false,
                                    amqp.Table{"x-dead-letter-exchange": "peril_dlx"},)
	if err != nil {
		return nil, amqp.Queue{}, err
	}
	
	err = ch.QueueBind(queueName, key, exchange, false, nil)
	if err != nil {
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

func PublishGob[T any](ch *amqp.Channel, exchange, key string, val T) error {
	
        var data bytes.Buffer
        enc := gob.NewEncoder(&data)


	err := enc.Encode(val)
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
			ContentType: "application/gob",
			Body: data.Bytes(),
		},
	)
	if err != nil {
		return err
	}
	return nil
}

func SubscribeJSON[T any](
	conn *amqp.Connection,
	exchange,
	queueName,
	key string,
	simpleQueueType SimpleQueueType,
	handler func(T) Acktype,
) error {
        return subscribe[T](
		conn,
		exchange,
		queueName,
		key,
		simpleQueueType,
		handler,
		func(data []byte) (T, error) {
			var target T
			err := json.Unmarshal(data, &target)
			return target, err },)
}

func SubscribeGob[T any](
	conn *amqp.Connection,
	exchange,
	queueName,
	key string,
	simpleQueueType SimpleQueueType,
	handler func(T) Acktype,
) error {
        return subscribe[T](
		conn,
		exchange,
		queueName,
		key,
		simpleQueueType,
		handler,
		func(data []byte) (T, error) {
                        buffer  := bytes.NewBuffer(data)
                        decoder := gob.NewDecoder(buffer)
                        var target T
			err := decoder.Decode(&target)
			return target, err },)
}


func subscribe[T any](
	conn *amqp.Connection,
	exchange,
	queueName,
	key string,
	simpleQueueType SimpleQueueType,
	handler func(T) Acktype,
	unmarshaller func([]byte) (T, error),
) error {
        ch, queue, err := DeclareAndBind(conn,
                                         exchange, 
                                         queueName, 
                                         key, 
                                         simpleQueueType)
        if err != nil {
                return fmt.Errorf("Error when declaring and binding connection and queue: %v", err)
        }

        err = ch.Qos(10, 0, false)
        if err != nil {
                return fmt.Errorf("Error when setting Qos %v", err)
        }

        deliveries, err := ch.Consume(queue.Name, "", false, false, false, false, nil)
        if err != nil {
                return fmt.Errorf("Error when Consuming Deliveries: %v", err)
        }
        
        go func() {
                defer ch.Close()
                for delivery := range deliveries {
                        msg, err := unmarshaller(delivery.Body)
                        if err != nil {
                                fmt.Printf("Error when Unmarshalling message: %v\n", err)
                                continue
                        }
                        switch handler(msg) {
                        case Ack:
                                delivery.Ack(false)
                                fmt.Println("Acked!")
                        case NackRequeue:
                                delivery.Nack(false, true)
                                fmt.Println("Nacked!  Retrying...")
                        case NackDiscard:
                                delivery.Nack(false, false)
                                fmt.Println("Nacked!  Discarding ;-; womp womp")
                        }
                }
        }()
        
        return nil

}
