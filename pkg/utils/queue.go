package utils

import (
	"fmt"
	"log"

	amqp "github.com/rabbitmq/amqp091-go"
)

func DeclareQueue(name string, ch *amqp.Channel) (queue amqp.Queue, err error) {
	queue, err = ch.QueueDeclare(
		name,  // name
		false, // durable
		false, // delete when unused
		false, // exclusive
		false, // no-wait
		nil,   // arguments
	)
	if err != nil {
		return
	}
	if err = ch.Qos(1, 0, false); err != nil {
		return
	}
	return
}

func FailOnNack(d amqp.Delivery, err error) {
	fmt.Printf("Could not marshal result: %v", err)
	// Message will be re-added to the queue
	if err = d.Nack(false, true); err != nil {
		log.Fatalf("Could not NACK to message queue: %v", err)
	}
}
