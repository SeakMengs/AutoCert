package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/SeakMengs/AutoCert/internal/config"
	"github.com/SeakMengs/AutoCert/internal/mailer"
	"github.com/SeakMengs/AutoCert/internal/repository"
	"github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"
)

type MailConsumerContext struct {
	Config     *config.Config
	Logger     *zap.SugaredLogger
	Repository *repository.Repository
	Mailer     mailer.Client
}

type MailJobPayload struct {
	ToEmail      string                  `json:"to_email"`
	TemplateFile mailer.MailTemplateFile `json:"template_file"`
	Data         json.RawMessage         `json:"data"`
	CreatedAt    string                  `json:"created_at"`
	Try          int                     `json:"try" default:"0"`
}

func NewMailJobPayload[T any](toEmail string, templateFile mailer.MailTemplateFile, data T) (MailJobPayload, error) {
	dataBytes, err := json.Marshal(data)
	if err != nil {
		return MailJobPayload{}, fmt.Errorf("failed to marshal data: %w", err)
	}

	return MailJobPayload{
		ToEmail:      toEmail,
		TemplateFile: templateFile,
		Data:         dataBytes,
		Try:          0,
		CreatedAt:    time.Now().Format(time.RFC3339),
	}, nil
}

func NewSignatureRequestInvitationMailJob(toEmail string, data mailer.SignatureRequestInvitationData) (MailJobPayload, error) {
	return NewMailJobPayload(toEmail, mailer.TemplateSignatureRequestInvitation, data)
}

type MailJobHandler func(ctx context.Context, jobPayload MailJobPayload, app *MailConsumerContext) (bool, error)

func (r *RabbitMQ) ConsumeMailJob(ctx context.Context, handler MailJobHandler, maxWorker int, app *MailConsumerContext) error {
	msgs, err := r.Consume(QueueMail)
	if err != nil {
		return fmt.Errorf("failed to start consuming mail jobs: %w", err)
	}

	for i := range maxWorker {
		go func(workerNumber int) {
			runMailWorker(ctx, r, workerNumber, msgs, handler, app)
		}(i + 1)
	}

	return nil
}

func runMailWorker(ctx context.Context, rabbitMQ *RabbitMQ, workerNumber int, msgs <-chan amqp091.Delivery, handler MailJobHandler, app *MailConsumerContext) {
	for {
		select {
		case <-ctx.Done():
			log.Printf("[Mail Worker %d] Shutting down", workerNumber)
			return
		case msg, ok := <-msgs:
			if !ok {
				log.Printf("[Mail Worker %d] Message channel closed", workerNumber)
				return
			}
			processMailJob(ctx, rabbitMQ, workerNumber, msg, handler, app)
		}
	}
}

func processMailJob(ctx context.Context, rabbitMQ *RabbitMQ, workerNumber int, msg amqp091.Delivery, handler MailJobHandler, app *MailConsumerContext) {
	if msg.Body == nil {
		log.Printf("[Mail Worker %d] Received empty message body", workerNumber)
		rabbitMQ.Nack(msg, false)
		return
	}

	var jobPayload MailJobPayload
	if err := json.Unmarshal(msg.Body, &jobPayload); err != nil {
		log.Printf("[Mail Worker %d] Invalid payload: %v", workerNumber, err)
		rabbitMQ.Nack(msg, false)
		return
	}

	workerPrefix := fmt.Sprintf("[Mail Worker %d: Retry %d]", workerNumber, jobPayload.Try)

	shouldRequeue, err := handler(ctx, jobPayload, app)
	if err != nil {
		log.Printf("%s Handler error processing mail job for recipient: %s, template: %s: %v",
			workerPrefix, jobPayload.ToEmail, jobPayload.TemplateFile, err)

		if !shouldRequeue || jobPayload.Try >= MAX_QUEUE_RETRY {
			log.Printf("%s Not requeuing mail job for recipient: %s, template: %s after error (retry: %d, shouldRequeue: %v)",
				workerPrefix, jobPayload.ToEmail, jobPayload.TemplateFile, jobPayload.Try, shouldRequeue)
			handleMailJobFailure(ctx, rabbitMQ, workerPrefix, msg, jobPayload, app)
			return
		}

		requeueMailJob(rabbitMQ, workerPrefix, msg, jobPayload)
		return
	}

	log.Printf("%s Successfully processed mail job for recipient: %s, template: %s",
		workerPrefix, jobPayload.ToEmail, jobPayload.TemplateFile)
	rabbitMQ.Ack(msg)
}

func handleMailJobFailure(ctx context.Context, rabbitMQ *RabbitMQ, workerPrefix string, msg amqp091.Delivery, jobPayload MailJobPayload, app *MailConsumerContext) {
	log.Printf("%s Handling mail job failure for recipient: %s, template: %s",
		workerPrefix, jobPayload.ToEmail, jobPayload.TemplateFile)

	rabbitMQ.Nack(msg, false)
}

func requeueMailJob(rabbitMQ *RabbitMQ, workerPrefix string, msg amqp091.Delivery, jobPayload MailJobPayload) {
	jobPayload.Try++
	payloadBytes, err := json.Marshal(jobPayload)
	if err != nil {
		log.Printf("%s Failed to marshal mail payload for requeue: %v", workerPrefix, err)
		rabbitMQ.Nack(msg, false)
		return
	}

	if err := rabbitMQ.Publish(QueueMail, payloadBytes); err != nil {
		log.Printf("%s Failed to requeue mail job for recipient: %s, template: %s: %v",
			workerPrefix, jobPayload.ToEmail, jobPayload.TemplateFile, err)
		rabbitMQ.Nack(msg, false)
		return
	}

	log.Printf("%s Requeued mail job for recipient: %s, template: %s",
		workerPrefix, jobPayload.ToEmail, jobPayload.TemplateFile)
	rabbitMQ.Ack(msg)
}
