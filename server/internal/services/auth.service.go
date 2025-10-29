package services

import (
	"errors"
	"log"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/tylerjvollick/nori/internal/models"
	"github.com/tylerjvollick/nori/internal/repositories"
	"golang.org/x/crypto/bcrypt"
)

type AuthService struct {
	userRepository        *repositories.UserRepository
	accountRepository     *repositories.AccountRepository
	userAccountRepository *repositories.UserAccountRepository
}

func NewAuthService(userRepository *repositories.UserRepository, accountRepository *repositories.AccountRepository, userAccountRepository *repositories.UserAccountRepository) *AuthService {
	return &AuthService{userRepository: userRepository, accountRepository: accountRepository,
		userAccountRepository: userAccountRepository,
	}
}

type LoginResponse struct {
	AccessToken string    `json:"accessToken"`
	UserID      uuid.UUID `json:"userId"`
	UserEmail   string    `json:"userEmail"`
	FirstName   string    `json:"firstName"`
	LastName    string    `json:"lastName"`
}

func (s *AuthService) ValidatePassword(user models.User, password string) bool {
	if user.Password == nil {
		return false
	}

	if err := bcrypt.CompareHashAndPassword([]byte(*user.Password), []byte(password)); err != nil {
		return false
	}
	return true
}

func (s *AuthService) Login(email, password string) (*LoginResponse, error) {
	// find user by email with password set
	user, err := s.userRepository.GetUserByEmail(email)
	if err != nil {
		// TODO: handle error in same line as failed password.
		log.Println("Failed to get user by email")
		return nil, errors.New("inNvalid email or password")
	}

	// if user is found validate password
	if user != nil {
		isAuthenticated := s.ValidatePassword(*user, password)
		if !isAuthenticated {
			log.Println("failed to validate password")
			return nil, errors.New("invalid Email or password")
		}
	}

	return s.CreateLoginResponse(*user)
}

func (s *AuthService) CreateLoginResponse(user models.User) (*LoginResponse, error) {
	// TODO: store secret in env vars
	var jwtSecret = []byte("your-secret-key")

	claims := jwt.MapClaims{
		"sub":   user.ID,
		"email": user.Email,
		"exp":   time.Now().Add(time.Hour * 24).Unix(),
		"iat":   time.Now().Unix(), // issued at
	}

	// create the token with claims
	// TODO learn about the different signing methods
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// sign token and return as string
	accessToken, err := token.SignedString(jwtSecret)
	if err != nil {
		log.Printf("failed to sign access token: %v", err)
		return nil, err
	}
	return &LoginResponse{
		AccessToken: accessToken,
		UserID:      user.ID,
		UserEmail:   user.Email,

		FirstName: *user.FirstName,
		LastName:  *user.LastName,
	}, nil
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

		// add link able
		userAccountRepository, err := s.userAccountRepository.Create(user.ID, defaultAccount.ID)
		if err != nil {
			return nil, err
		}
		if userAccountRepository == nil {
			return nil, nil
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
