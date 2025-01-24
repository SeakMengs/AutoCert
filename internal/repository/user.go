package repository

import (
	"context"

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

func (ur UserRepository) CreateOrUpdateByEmail(ctx context.Context, tx *gorm.DB, newUser model.User) (*model.User, error) {
	ur.logger.Debugf("Create or update by email with data: %v \n", newUser)

	db := ur.getDB(tx)
	ctx, cancel := context.WithTimeout(ctx, constant.QUERY_TIMEOUT_DURATION)
	defer cancel()

	var user *model.User
	// Assign mean it will create or update regardless of whether record is found or not
	// It check based on where condition
	if err := db.WithContext(ctx).Model(&model.User{}).Where(&model.User{Email: newUser.Email}).Assign(model.User{
		Email:      newUser.Email,
		FirstName:  newUser.FirstName,
		LastName:   newUser.LastName,
		ProfileURL: newUser.ProfileURL,
	}).FirstOrCreate(&user).Error; err != nil {
		return user, err
	}

	return user, nil
}
