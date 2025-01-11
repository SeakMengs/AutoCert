package repository

import (
	"github.com/SeakMengs/AutoCert/internal/auth"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type baseRepository struct {
	db         *gorm.DB
	logger     *zap.SugaredLogger
	jwtService auth.JWTInterface
}

type Repository struct {
	User          *UserRepository
	JWT           *JWTRepository
	OAuthProvider *OAuthProviderRepository
}

func newBaseRepository(db *gorm.DB, logger *zap.SugaredLogger, jwtService auth.JWTInterface) *baseRepository {
	return &baseRepository{db: db, logger: logger, jwtService: jwtService}
}

func NewRepository(db *gorm.DB, logger *zap.SugaredLogger, jwtService auth.JWTInterface) *Repository {
	br := newBaseRepository(db, logger, jwtService)
	_userRepo := &UserRepository{baseRepository: br}

	return &Repository{
		User:          _userRepo,
		JWT:           &JWTRepository{baseRepository: br, user: _userRepo},
		OAuthProvider: &OAuthProviderRepository{baseRepository: br},
	}
}

// Example usage can be found in user repository: GetUserAndCreate
// Note: GORM perform write (create/update/delete) operations run inside a transaction to ensure data consistency | So this function is helpful only if we disable auto transaction
// Docs: https://gorm.io/docs/transactions.html#Disable-Default-Transaction
func (b baseRepository) withTx(db *gorm.DB, fn func(*gorm.DB) error) error {
	tx := db.Begin()
	if err := tx.Error; err != nil {
		return err
	}

	defer func() {
		// https://gorm.io/docs/transactions.html#A-Specific-Example
		// If panic is throw rollback
		if r := recover(); r != nil {
			b.logger.Info("withTx() Transaction panic, perform rollback")
			tx.Rollback()
		}
	}()

	if err := fn(tx); err != nil {
		b.logger.Info("withTx() Error during transaction, perform rollback")
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}

func (b baseRepository) getDB(tx *gorm.DB) *gorm.DB {
	if tx != nil {
		return tx
	}

	return b.db
}
