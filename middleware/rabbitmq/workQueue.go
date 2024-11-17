package rabbitmq

import (
	"log"

	amqp "github.com/rabbitmq/amqp091-go"
)

type WorkQueue struct {
	Channel *amqp.Channel
	Que     amqp.Queue
}

func NewWorkQueue(conn *amqp.Connection, queue_name string, is_durable bool) *WorkQueue {
	channel, err := conn.Channel()
	if err != nil {
		log.Panicf("failed to open a channel from RabbitMQ connection, detail: %s", err)
	}

	q, err := channel.QueueDeclare(
		queue_name,
		is_durable,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		log.Panicf("failed to declare queue[%s], detail: %s", queue_name, err)
	}

	return &WorkQueue{
		Channel: channel,
		Que:     q,
	}
}

func (q WorkQueue) Publish(data []byte) {
	err := q.Channel.Publish(
		"", // use default exchange
		q.Que.Name,
		false,
		false,
		amqp.Publishing{
			DeliveryMode: amqp.Persistent,
			ContentType:  "text/plain",
			Body:         data,
		},
	)
	if err != nil {
		log.Panicf("failed to publish to queue[%s], detail: %s", q.Que.Name, err)
	}
}

func (q WorkQueue) Consume(hanlder func(amqp.Delivery)) {
	msgs, err := q.Channel.Consume(
		q.Que.Name,
		"",
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		log.Panicf("failed to consume from queue[%s], detail: %s", q.Que.Name, err)
	}

	for d := range msgs {
		hanlder(d)
		d.Ack(false)
	}
}

func (q WorkQueue) Close() error {
	return q.Channel.Close()
}
