package queue

import (
	"context"
	"encoding/json"
	"log"

	"github.com/SeakMengs/AutoCert/internal/auth"
	"github.com/SeakMengs/AutoCert/internal/config"
	"github.com/SeakMengs/AutoCert/internal/constant"
	"github.com/SeakMengs/AutoCert/internal/repository"
	"github.com/minio/minio-go/v7"
	"go.uber.org/zap"
)

type ConsumerContext struct {
	// Config holds application settings provided from .env file.
	Config *config.Config

	// Logger lol....
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
	Retry     int    `json:"retry" default:"0"`
}

type CertificateGenerateJobHandler func(jobPayload CertificateGeneratePayload, app *ConsumerContext) (bool, error)

func dropQeueCleanUp(app *ConsumerContext, projectId string) error {
	ctx := context.Background()

	if err := app.Repository.Project.UpdateStatus(ctx, nil, projectId, constant.ProjectStatusDraft); err != nil {
		return err
	}

	return nil
}

func (r *RabbitMQ) ConsumeCertificateGenerateJob(handler CertificateGenerateJobHandler, maxWorker int, app *ConsumerContext) error {
	msgs, err := r.Consume(QueueCertificateGenerate)
	if err != nil {
		return err
	}

	for i := range maxWorker {
		go func(workerID int) {
			for msg := range msgs {
				if msg.Body == nil {
					log.Printf("[Worker %d] Received empty message body", workerID)
					// Acknowledge the message and remove it from the queue
					_ = r.Nack(msg, false)
					continue
				}

				var jobPayload CertificateGeneratePayload
				if err := json.Unmarshal(msg.Body, &jobPayload); err != nil {
					log.Printf("[Worker %d] Invalid payload: %v", workerID, err)
					// Acknowledge the message and remove it from the queue
					_ = r.Nack(msg, false)
					continue
				}

				jobPayload.Retry++
				if jobPayload.Retry > MAX_QUEUE_RETRY {
					log.Printf("[Worker %d] Max retries reached", workerID)
					// Acknowledge the message and remove it from the queue
					_ = r.Nack(msg, false)
					continue
				}
				lastRetry := jobPayload.Retry == MAX_QUEUE_RETRY

				shouldRequeue, err := handler(jobPayload, app)
				if err != nil {
					log.Printf("[Worker %d] Handler error: %v", workerID, err)

					if !shouldRequeue || lastRetry {
						if err := dropQeueCleanUp(app, jobPayload.ProjectID); err != nil {
							log.Printf("[Worker %d] Failed to clean up project %s: %v", workerID, jobPayload.ProjectID, err)
							r.Nack(msg, false)
							continue
						}

						log.Printf("[Worker %d] Dropped project %s due to max retries", workerID, jobPayload.ProjectID)
						r.Nack(msg, false)
						continue
					}

					payloadBytes, err := json.Marshal(jobPayload)
					if err != nil {
						log.Printf("[Worker %d] Failed to marshal job payload: %v", workerID, err)
						_ = r.Nack(msg, false)
						continue
					}

					// requeue with updated retry count
					if err := r.Publish(QueueCertificateGenerate, payloadBytes); err != nil {
						log.Printf("[Worker %d] Failed to requeue job: %v", workerID, err)
						// Acknowledge the message and remove it from the queue
						_ = r.Nack(msg, false)
						continue
					}

					log.Printf("[Worker %d] Requeued job for ProjectID: %s, UserID: %s, Retry: %d", workerID, jobPayload.ProjectID, jobPayload.UserID, jobPayload.Retry)
					_ = r.Ack(msg)
					continue
				}

				log.Printf("[Worker %d] Successfully processed job for ProjectID: %s, UserID: %s", workerID, jobPayload.ProjectID, jobPayload.UserID)
				_ = r.Ack(msg)
			}
		}(i + 1)
	}

	return nil
}
