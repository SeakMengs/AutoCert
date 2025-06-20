package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/SeakMengs/AutoCert/internal/auth"
	"github.com/SeakMengs/AutoCert/internal/config"
	"github.com/SeakMengs/AutoCert/internal/constant"
	"github.com/SeakMengs/AutoCert/internal/repository"
	"github.com/minio/minio-go/v7"
	"github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"
)

type ConsumerContext struct {
	// Config holds application settings provided from .env file.
	Config *config.Config

	Logger *zap.SugaredLogger

	// Repository provides access to data storage operations.
	Repository *repository.Repository

	// JWTService manages JWT operations for authentication such as generate, verify, refresh token.
	JWTService auth.JWTInterface

	S3 *minio.Client
}

type CertificateGeneratePayload struct {
	ProjectID string `json:"project_id"`
	UserID    string `json:"user_id"`
	CreatedAt string `json:"created_at"`
	// Try start from 0
	Try int `json:"try" default:"0"`
}

func NewCertificateGeneratePayload(projectID, userID string) CertificateGeneratePayload {
	return CertificateGeneratePayload{
		ProjectID: projectID,
		UserID:    userID,
		Try:       0,
		CreatedAt: time.Now().Format(time.RFC3339),
	}
}

type CertificateGenerateJobHandler func(ctx context.Context, jobPayload CertificateGeneratePayload, app *ConsumerContext) (bool, error)

func (r *RabbitMQ) ConsumeCertificateGenerateJob(ctx context.Context, handler CertificateGenerateJobHandler, maxWorker int, app *ConsumerContext) error {
	msgs, err := r.Consume(QueueCertificateGenerate)
	if err != nil {
		return fmt.Errorf("failed to start consuming: %w", err)
	}

	for i := range maxWorker {
		go func(workerNumber int) {
			runCertificateWorker(ctx, r, workerNumber, msgs, handler, app)
		}(i + 1)
	}

	return nil
}

// Run a single worker that processes certificate generation jobs
func runCertificateWorker(ctx context.Context, rabbitMQ *RabbitMQ, workerNumber int, msgs <-chan amqp091.Delivery, handler CertificateGenerateJobHandler, app *ConsumerContext) {
	for {
		select {
		case <-ctx.Done():
			log.Printf("[Worker %d] Shutting down", workerNumber)
			return
		case msg, ok := <-msgs:
			if !ok {
				log.Printf("[Worker %d] Message channel closed", workerNumber)
				return
			}
			processCertificateJob(ctx, rabbitMQ, workerNumber, msg, handler, app)
		}
	}
}

// Handle a single certificate generation job message
func processCertificateJob(ctx context.Context, rabbitMQ *RabbitMQ, workerNumber int, msg amqp091.Delivery, handler CertificateGenerateJobHandler, app *ConsumerContext) {
	if msg.Body == nil {
		log.Printf("[Worker %d] Received empty message body", workerNumber)
		rabbitMQ.Nack(msg, false)
		return
	}

	var jobPayload CertificateGeneratePayload
	if err := json.Unmarshal(msg.Body, &jobPayload); err != nil {
		log.Printf("[Worker %d] Invalid payload: %v", workerNumber, err)
		rabbitMQ.Nack(msg, false)
		return
	}

	workerPrefix := fmt.Sprintf("[Worker %d: Retry %d]", workerNumber, jobPayload.Try)

	shouldRequeue, err := handler(ctx, jobPayload, app)
	if err != nil {
		log.Printf("%s Handler error processing job for ProjectID: %s, UserID: %s: %v", workerPrefix, jobPayload.ProjectID, jobPayload.UserID, err)

		// Check if we've reached max retries or shouldn't requeue
		if !shouldRequeue || jobPayload.Try >= MAX_QUEUE_RETRY {
			log.Printf("%s Not requeuing job for ProjectID: %s, UserID: %s after error (retry: %d, shouldRequeue: %v)",
				workerPrefix, jobPayload.ProjectID, jobPayload.UserID, jobPayload.Try, shouldRequeue)
			handleCertificateJobFailure(ctx, rabbitMQ, workerPrefix, msg, jobPayload, app)
			return
		}

		requeueCertificateJob(rabbitMQ, workerPrefix, msg, jobPayload)
		return
	}

	log.Printf("%s Successfully processed job for ProjectID: %s, UserID: %s", workerPrefix, jobPayload.ProjectID, jobPayload.UserID)
	rabbitMQ.Ack(msg)
}

// Handle cleanup when a job fails then acknowledge the message and remove it from the queue without requeuing
func handleCertificateJobFailure(ctx context.Context, rabbitMQ *RabbitMQ, workerPrefix string, msg amqp091.Delivery, jobPayload CertificateGeneratePayload, app *ConsumerContext) {
	if err := dropQueueCleanUp(ctx, app, jobPayload.ProjectID); err != nil {
		log.Printf("%s Failed to clean up project %s: %v", workerPrefix, jobPayload.ProjectID, err)
	}
	rabbitMQ.Nack(msg, false)
}

func dropQueueCleanUp(ctx context.Context, app *ConsumerContext, projectID string) error {
	if err := app.Repository.Project.UpdateStatus(ctx, nil, projectID, constant.ProjectStatusDraft); err != nil {
		return fmt.Errorf("failed to update project status: %w", err)
	}
	return nil
}

// Requeue a failed job with updated retry count
func requeueCertificateJob(rabbitMQ *RabbitMQ, workerPrefix string, msg amqp091.Delivery, jobPayload CertificateGeneratePayload) {
	jobPayload.Try++
	payloadBytes, err := json.Marshal(jobPayload)
	if err != nil {
		log.Printf("%s Failed to marshal payload for requeue: %v", workerPrefix, err)
		rabbitMQ.Nack(msg, false)
		return
	}

	if err := rabbitMQ.Publish(QueueCertificateGenerate, payloadBytes); err != nil {
		log.Printf("%s Failed to requeue job for ProjectID: %s, UserID: %s: %v", workerPrefix, jobPayload.ProjectID, jobPayload.UserID, err)
		rabbitMQ.Nack(msg, false)
		return
	}

	log.Printf("%s Requeued job for ProjectID: %s, UserID: %s", workerPrefix, jobPayload.ProjectID, jobPayload.UserID)
	rabbitMQ.Ack(msg)
}
