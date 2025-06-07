package queue

import (
	amqp "github.com/rabbitmq/amqp091-go"
)

type RabbitMQ struct {
	conn    *amqp.Connection
	channel *amqp.Channel
}

type QueueName string

const (
	QueueCertificateGenerate QueueName = "certificate_generate_queue"
)

const (
	MAX_QUEUE_RETRY = 3
)

func NewRabbitMQ(url string) (*RabbitMQ, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, err
	}

	channel, err := conn.Channel()
	if err != nil {
		return nil, err
	}

	// Declare a queue to ensure it exists before publishing messages
	_, err = channel.QueueDeclare(
		string(QueueCertificateGenerate), // name of the queue
		true,                             // durable
		false,                            // delete when unused
		false,                            // exclusive
		false,                            // no-wait
		nil,                              // arguments
	)
	if err != nil {
		return nil, err
	}

	return &RabbitMQ{
		conn:    conn,
		channel: channel,
	}, nil
}

func (r *RabbitMQ) Close() error {
	if err := r.channel.Close(); err != nil {
		return err
	}
	if err := r.conn.Close(); err != nil {
		return err
	}
	return nil
}

func (r *RabbitMQ) Publish(routingKey QueueName, body []byte) error {
	err := r.channel.Publish(
		"", // default exchange
		string(routingKey),
		false, // mandatory
		false, // immediate
		amqp.Publishing{
			// make message persistent even if RabbitMQ restarts or crashes
			DeliveryMode: amqp.Persistent,
			ContentType:  "application/json",
			Body:         body,
		},
	)
	if err != nil {
		return err
	}
	return nil
}

// Tell RabbitMQ to deliver messages one at a time to consumers
// until it has processed and acknowledged the previous one.
// Docs: https://www.rabbitmq.com/tutorials/tutorial-two-go#fair-dispatch
func (r *RabbitMQ) fairDispatch() error {
	return r.channel.Qos(
		1,     // prefetch count
		0,     // prefetch size
		false, // global
	)
}

func (r *RabbitMQ) Consume(queueName QueueName) (<-chan amqp.Delivery, error) {
	err := r.fairDispatch()
	if err != nil {
		return nil, err
	}

	deliveries, err := r.channel.Consume(
		string(queueName), // name of the queue
		"",                // consumer tag
		false,             // auto-ack
		false,             // exclusive
		false,             // no-local
		false,             // no-wait
		nil,               // arguments
	)
	if err != nil {
		return nil, err
	}
	return deliveries, nil
}

func (r *RabbitMQ) Ack(delivery amqp.Delivery) error {
	if err := delivery.Ack(false); err != nil {
		return err
	}
	return nil
}

func (r *RabbitMQ) Nack(delivery amqp.Delivery, requeue bool) error {
	if err := delivery.Nack(false, requeue); err != nil {
		return err
	}
	return nil
}
