package repository

import (
	"github.com/SeakMengs/AutoCert/internal/auth"
	"github.com/minio/minio-go/v7"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type baseRepository struct {
	db         *gorm.DB
	logger     *zap.SugaredLogger
	jwtService auth.JWTInterface
	s3         *minio.Client
}

type Repository struct {
	// DB can be used for transaction. Example usage:
	// tx := r.DB.Begin()
	// defer tx.Commit()
	// Then pass tx to the repository function. and use tx.Rollback() if error occurred
	DB                *gorm.DB
	User              *UserRepository
	JWT               *JWTRepository
	OAuthProvider     *OAuthProviderRepository
	Project           *ProjectRepository
	File              *FileRepository
	ColumnAnnotate    *ColumnAnnotateRepository
	SignatureAnnotate *SignatureAnnotateRepository
	Signature         *SignatureRepository
	Certificate       *CertificateRepository
	ProjectLog        *ProjectLogRepository
}

func newBaseRepository(db *gorm.DB, logger *zap.SugaredLogger, jwtService auth.JWTInterface, s3 *minio.Client) *baseRepository {
	return &baseRepository{db: db, logger: logger, jwtService: jwtService, s3: s3}
}

func NewRepository(db *gorm.DB, logger *zap.SugaredLogger, jwtService auth.JWTInterface, s3 *minio.Client) *Repository {
	br := newBaseRepository(db, logger, jwtService, s3)
	_userRepo := &UserRepository{baseRepository: br}

	return &Repository{
		DB:                db,
		User:              _userRepo,
		JWT:               &JWTRepository{baseRepository: br, user: _userRepo},
		OAuthProvider:     &OAuthProviderRepository{baseRepository: br},
		Project:           &ProjectRepository{baseRepository: br},
		File:              &FileRepository{baseRepository: br},
		ColumnAnnotate:    &ColumnAnnotateRepository{baseRepository: br},
		SignatureAnnotate: &SignatureAnnotateRepository{baseRepository: br},
		Signature:         &SignatureRepository{baseRepository: br},
		Certificate:       &CertificateRepository{baseRepository: br},
		ProjectLog:        &ProjectLogRepository{baseRepository: br},
	}
}

// Example usage can be found in user repository: GetUserAndCreate
// Note: GORM perform write (create/update/delete) operations run inside a transaction to ensure data consistency | So this function is helpful only if we disable auto transaction
// Docs: https://gorm.io/docs/transactions.html#Disable-Default-Transaction
func (b baseRepository) withTx(db *gorm.DB, fn func(*gorm.DB) error) error {
	// tx := db.Begin()
	// if err := tx.Error; err != nil {
	// 	return err
	// }

	// defer func() {
	// 	// https://gorm.io/docs/transactions.html#A-Specific-Example
	// 	// If panic is throw rollback
	// 	if r := recover(); r != nil {
	// 		b.logger.Error("withTx() Transaction panic, perform rollback")
	// 		tx.Rollback()
	// 	}
	// }()

	// if err := fn(tx); err != nil {
	// 	b.logger.Debugf("withTx() Error during transaction, perform rollback. Error: %v", err)
	// 	tx.Rollback()
	// 	return err
	// }

	// return tx.Commit().Error
	err := db.Transaction(func(tx *gorm.DB) error {
		return fn(tx)
	})

	if err != nil {
		b.logger.Error("withTx Transaction error: %v", err)
	}

	return err
}

func (b baseRepository) getDB(tx *gorm.DB) *gorm.DB {
	if tx != nil {
		return tx
	}

	return b.db
}
