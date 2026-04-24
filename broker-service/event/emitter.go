package event

import (
	"log"

	amqp "github.com/rabbitmq/amqp091-go"
)

type Emitter struct {
	conn *amqp.Connection
}

func (e *Emitter) Push(event string, severity string) error {
	channel, err := e.conn.Channel()
	if err != nil {
		return err
	}
	defer channel.Close()

	log.Println("Pushing to channel")

	// publish the event to the channel
	err = channel.Publish(
		"logs_topic",
		severity, // info, warning, error
		false,
		false,
		amqp.Publishing{
			ContentType: "text/plain",
			Body:        []byte(event),
		},
	)

	if err != nil {
		return err
	}

	return nil
}

// NewEventEmitter creates a new event emitter
func NewEventEmitter(conn *amqp.Connection) (Emitter, error) {
	emitter := Emitter{
		conn: conn,
	}

	err := emitter.setup()
	if err != nil {
		return Emitter{}, err
	}

	return emitter, nil
}

// Helper function to setup the channel and declare the exchange
func (e *Emitter) setup() error {
	channel, err := e.conn.Channel()
	if err != nil {
		return err
	}
	defer channel.Close()

	return declareExchange(channel)
}
