package main

import (
	"fmt"
	"tiktok/middleware/rabbitmq"

	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	amqp_uri = "amqp://localhost:5672"
)

func CoverService(delivery amqp.Delivery) {
	title := string(delivery.Body)
	for {
		if ok := screenCaptureAndUpload(title); ok {
			break
		}
	}
}

func main() {
	conn := rabbitmq.NewRmqConnection(amqp_uri)
	defer conn.Close()
	coverQueue := rabbitmq.NewCoverQueue(conn)
	defer coverQueue.Close()

	go coverQueue.Consume(CoverService)
	fmt.Printf(" [*] Waiting for messages. To exit press CTRL+C")
	select {}
}
