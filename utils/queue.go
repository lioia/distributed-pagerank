package utils

import (
	"fmt"
	"log"

	rabbithole "github.com/michaelklishin/rabbit-hole/v2"
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

func EmptyQueue(ch *amqp.Channel, name string) {
	msgs, err := ch.Consume(
		name,  // queue
		"",    // consumer
		false, // auto-ack
		false, // exclusive
		false, // no-local
		false, // no-wait
		nil,   // args
	)
	FailOnError("Could not register a consumer", err)

	for msg := range msgs {
		// Acknowledge the message to remove it from the queue
		err := msg.Ack(false)
		if err != nil {
			log.Printf("Failed to acknowledge message: %v", err)
		}
	}
}

// Contact RabbitMQ management API to query the number of messages
// NOTE: this can be probably removed if the master reads the messages
// but does not ack them until it received #jobs messages
// This can be achieved by saving the DeliveryTag in an array
func GetNumberOfQueueMessages(queueName string) int {
	host, err := ReadStringEnvVar("RABBIT_HOST")
	FailOnError("Failed to read environment variables", err)
	user := ReadStringEnvVarOr("RABBIT_USER", "guest")
	pass := ReadStringEnvVarOr("RABBIT_PASS", "guest")
	url := fmt.Sprintf("http://%s:15672", host)
	rmqc, err := rabbithole.NewClient(url, user, pass)
	FailOnError("Failed to connect to RabbitMQ Management", err)
	queue, err := rmqc.GetQueue("/", queueName)
	FailOnError(fmt.Sprintf("Failed to get queue %s", queueName), err)
	return queue.MessagesUnacknowledged
}
