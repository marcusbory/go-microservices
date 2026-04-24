package event

import (
	amqp "github.com/rabbitmq/amqp091-go"
)

// ? Ensure RabbitMQ has a topic exchange named 'logs_topic' with such properties.
// Create a topic exchange in RabbitMQ (if not existing)
// * "Setting up a shared mailroom - publisher drops off mail, consumer delivers to right places"
func declareExchange(ch *amqp.Channel) error {
	return ch.ExchangeDeclare(
		"logs_topic", // name
		"topic",      // type
		true,         // durable
		false,        // auto-deleted
		false,        // internal
		false,        // no-wait
		nil,          // arguments
	)
}

// ! Not needed for a Publisher (Broker), only Consumer (Listener)
// func declareRandomQueue(ch *amqp.Channel) (amqp.Queue, error) {
// 	return ch.QueueDeclare(
// 		"",    // name? pick your own or auto generate
// 		false, // durable, get rid when done with it
// 		false, // delete when unused
// 		true,  // exclusive, don't share queue
// 		false, // no-wait
// 		nil,   // arguments
// 	)
// }
