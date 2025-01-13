package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"

	constant "github.com/SeakMengs/AutoCert/internal/constant"
	"github.com/SeakMengs/AutoCert/internal/model"
	"gorm.io/gorm"
)

type UserRepository struct {
	*baseRepository
}

func (ur UserRepository) GetById(ctx context.Context, tx *gorm.DB, userId string) (*model.User, error) {
	ur.logger.Debugf("Get user by id: %s \n", userId)

	db := ur.getDB(tx)
	var user *model.User

	ctx, cancel := context.WithTimeout(ctx, constant.QUERY_TIMEOUT_DURATION)
	defer cancel()

	if err := db.WithContext(ctx).Model(&model.User{}).Where(&model.User{ID: userId}).First(&user).Error; err != nil {
		return user, err
	}

	return user, nil
}

func (ur UserRepository) GetByEmail(ctx context.Context, tx *gorm.DB, email string) (*model.User, error) {
	ur.logger.Debugf("Get user by email: %s \n", email)

	db := ur.getDB(tx)
	var user *model.User

	ctx, cancel := context.WithTimeout(ctx, constant.QUERY_TIMEOUT_DURATION)
	defer cancel()

	if err := db.WithContext(ctx).Model(&model.User{}).Where(&model.User{Email: email}).First(&user).Error; err != nil {
		return user, err
	}

	return user, nil
}

func (ur *UserRepository) Create(ctx context.Context, tx *gorm.DB, newUser model.User) error {
	ur.logger.Debugf("Create user with data: %v \n", newUser)

	db := ur.getDB(tx)
	ctx, cancel := context.WithTimeout(ctx, constant.QUERY_TIMEOUT_DURATION)
	defer cancel()

	// Use omit if u want to prevent insert id
	// if err := db.WithContext(ctx).Model(&model.User{}).Omit(user.ID).Create(&user).Error; err != nil {

	if err := db.WithContext(ctx).Model(&model.User{}).Create(&model.User{
		Email:     newUser.Email,
		FirstName: newUser.FirstName,
		LastName:  newUser.LastName,
	}).Error; err != nil {
		return err
	}

	return nil
}

// Example transaction
func (ur *UserRepository) CheckDupAndCreate(ctx context.Context, tx *gorm.DB, newUser model.User) error {
	ur.logger.Debugf("Get user and create user with data (Transaction): %v \n", newUser)

	db := ur.getDB(tx)
	txErr := ur.withTx(db, func(tx *gorm.DB) error {
		existingUser, err := ur.GetByEmail(ctx, tx, newUser.Email)
		if err != nil {
			// Since not found is not an error, we can ignore it
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}
		}

		if strings.EqualFold(existingUser.Email, newUser.Email) {
			return fmt.Errorf("user with %s already exist", existingUser.Email)
		}

		// Create a new user with the transaction
		if err := ur.Create(ctx, tx, newUser); err != nil {
			return err
		}

		return nil
	})

	return txErr
}
