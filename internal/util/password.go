package util

import (
	"errors"

	"golang.org/x/crypto/bcrypt"
)

func HashPassword(password string) (string, error) {
	// default cost is 10 rounds of hashing
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

func CheckPassword(hashPassword string, plainPassword []byte) (bool, error) {
	if hashPassword == "" {
		return false, errors.New("Wrong password")
	}

	hashPW := []byte(hashPassword)
	if err := bcrypt.CompareHashAndPassword(hashPW, plainPassword); err != nil {
		return false, err
	}
	return true, nil
}
