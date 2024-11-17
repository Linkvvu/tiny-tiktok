package rabbitmq

import (
	"log"

	amqp "github.com/rabbitmq/amqp091-go"
)

func NewRmqConnection(uri string) *amqp.Connection {
	conn, err := amqp.Dial(uri)
	if err != nil {
		log.Panicf("failed to connect target MQ, detail: %s", err)
	}
	return conn
}
