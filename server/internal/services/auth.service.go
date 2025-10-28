package services

import (
	"errors"
	"log"

	"github.com/google/uuid"
	"github.com/tylerjvollick/nori/internal/models"
	"github.com/tylerjvollick/nori/internal/repositories"
	"golang.org/x/crypto/bcrypt"
)

type AuthService struct {
	userRepository    *repositories.UserRepository
	accountRepository *repositories.AccountRepository
}

func NewAuthService(userRepository *repositories.UserRepository, accountRepository *repositories.AccountRepository) *AuthService {
	return &AuthService{userRepository: userRepository, accountRepository: accountRepository}
}

func (s *AuthService) CreateUser(firstName, lastName, email, password string, createDefaultAccount bool) (*models.User, error) {
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

	if createDefaultAccount {
		defaultAccount, err := s.CreateAccount(email, user.ID, models.Trial)
		if err != nil {
			return nil, err
		}

		// update user with default account id.
		// user.DefaultAccountID = defaultAccount.
		user, err = s.userRepository.UpdateUser(user.ID, &repositories.UpdateUserInput{DefaultAccountID: &defaultAccount.ID})
		log.Println("Updating user default account")
		if err != nil {
			return nil, err
		}
	}

	// do not return password
	user.Password = nil
	return user, nil
}

func (s *AuthService) CreateAccount(billingEmail string, createdByUserId uuid.UUID, plan models.Plan) (*models.Account, error) {
	// save account
	account, err := s.accountRepository.Create(billingEmail, createdByUserId, plan)
	if err != nil {
		return nil, err
	}
	return account, nil
}

func ptrString(s string) *string {
	return &s
}
