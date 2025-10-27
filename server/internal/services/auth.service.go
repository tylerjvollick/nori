package services

import (
	"errors"

	"github.com/google/uuid"
	"github.com/tylerjvollick/nori/internal/models"
	"github.com/tylerjvollick/nori/internal/repositories"
	"golang.org/x/crypto/bcrypt"
)

type AuthService struct {
	userRepository *repositories.UserRepository
}

func NewAuthService(userRepository *repositories.UserRepository) *AuthService {
	return &AuthService{userRepository: userRepository}
}

func (s *AuthService) CreateUser(firstName, lastName, email, password string) (*models.User, error) {
	// check to see if use already exists
	existing, err := s.userRepository.GetUserByEmail(email)
	if err == nil && existing != nil {
		return nil, errors.New("email already in use")
	}

	// hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	// build the user object
	user := &models.User{
		ID:        uuid.New(),
		FirstName: &firstName,
		LastName:  &lastName,
		Email:     email,
		Password:  ptrString(string(hashedPassword)),
	}

	// save user

	if err := s.userRepository.CreateUser(user); err != nil {
		return nil, err
	}

	// do not return password
	user.Password = nil
	return user, nil
}

func ptrString(s string) *string {
	return &s
}
