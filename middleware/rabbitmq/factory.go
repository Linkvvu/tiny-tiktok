package rabbitmq

import amqp "github.com/rabbitmq/amqp091-go"

const (
	cover_queue_name = "queue.cover"
	video_queue_name = "queue.video"
)

func NewCoverQueue(conn *amqp.Connection) *WorkQueue {
	return NewWorkQueue(conn, cover_queue_name, true)
}
