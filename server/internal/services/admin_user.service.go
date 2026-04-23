package services

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/tylerjvollick/nori/internal/models"
	"github.com/tylerjvollick/nori/internal/repositories"
	"golang.org/x/crypto/bcrypt"
)

type AdminUserService struct {
	userRepo        *repositories.UserRepository
	userAccountRepo *repositories.UserAccountRepository
	spaceRepo       *repositories.SpaceRepository
	spaceMemberRepo *repositories.SpaceMemberRepository
}

func NewAdminUserService(userRepo *repositories.UserRepository, userAccountRepo *repositories.UserAccountRepository, spaceRepo *repositories.SpaceRepository, spaceMemberRepo *repositories.SpaceMemberRepository) *AdminUserService {
	return &AdminUserService{
		userRepo:        userRepo,
		userAccountRepo: userAccountRepo,
		spaceRepo:       spaceRepo,
		spaceMemberRepo: spaceMemberRepo,
	}
}

// CreateUser creates a new user with the specified role and temporary password
func (s *AdminUserService) CreateUser(accountID uuid.UUID, email, firstName, lastName, tempPassword string, role models.Role) (*models.User, error) {
	// Check if user with email already exists
	existingUser, err := s.userRepo.GetUserByEmail(email)
	if err == nil && existingUser != nil {
		return nil, fmt.Errorf("user with email %s already exists", email)
	}

	// Hash the temporary password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(tempPassword), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	hashedPasswordStr := string(hashedPassword)

	// Create user
	user := &models.User{
		Email:              email,
		FirstName:          &firstName,
		LastName:           &lastName,
		Password:           &hashedPasswordStr,
		Role:               &role,
		MustChangePassword: true, // Force password change on first login
		DefaultAccountID:   &accountID,
		RecentSpaces:       models.RecentSpaces{},
	}

	err = s.userRepo.CreateUser(user)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	// Create UserAccount relationship
	_, err = s.userAccountRepo.Create(user.ID, accountID)
	if err != nil {
		// TODO: Consider rolling back user creation if UserAccount creation fails
		return nil, fmt.Errorf("failed to create user account relationship: %w", err)
	}

	// Auto-add user to all spaces in the account
	if s.spaceRepo != nil && s.spaceMemberRepo != nil {
		spaces, err := s.spaceRepo.FindByAccountID(accountID)
		if err == nil {
			for _, space := range spaces {
				member := &models.SpaceMember{
					UserID:  user.ID,
					SpaceID: space.ID,
				}
				if err := s.spaceMemberRepo.Create(member); err != nil {
					// Log but don't fail user creation
					fmt.Printf("Warning: failed to add user %s to space %s: %v\n", user.ID, space.ID, err)
				}
			}
		}
	}

	return user, nil
}

// ListUsers returns all users for the specified account
func (s *AdminUserService) ListUsers(accountID uuid.UUID) ([]models.User, error) {
	userAccounts, err := s.userAccountRepo.GetByAccountID(accountID)
	if err != nil {
		return nil, fmt.Errorf("failed to get users for account: %w", err)
	}

	users := make([]models.User, 0, len(userAccounts))
	for _, ua := range userAccounts {
		user, err := s.userRepo.GetUserByID(ua.UserID)
		if err != nil {
			// Skip users that can't be found, but log the error
			continue
		}
		users = append(users, *user)
	}

	return users, nil
}

// UpdateUser updates user details (name and/or role)
func (s *AdminUserService) UpdateUser(userID uuid.UUID, firstName, lastName *string, role *models.Role) (*models.User, error) {
	// Get existing user
	user, err := s.userRepo.GetUserByID(userID)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	// Update fields
	if firstName != nil {
		user.FirstName = firstName
	}
	if lastName != nil {
		user.LastName = lastName
	}
	if role != nil {
		user.Role = role
	}

	// Use the existing UpdateUser method to save changes
	input := &repositories.UpdateUserInput{
		Email:            &user.Email,
		DefaultAccountID: user.DefaultAccountID,
	}

	updatedUser, err := s.userRepo.UpdateUser(userID, input)
	if err != nil {
		return nil, fmt.Errorf("failed to update user: %w", err)
	}

	return updatedUser, nil
}

// DeleteUser deletes (or deactivates) a user
func (s *AdminUserService) DeleteUser(userID uuid.UUID) error {
	// For now, we'll just check that the user exists
	// In a production system, you might want to soft-delete instead
	_, err := s.userRepo.GetUserByID(userID)
	if err != nil {
		return fmt.Errorf("user not found: %w", err)
	}

	// TODO: Implement actual deletion or deactivation
	// For now, we'll just return nil to indicate success
	// In production, you'd want to:
	// 1. Soft delete by adding a DeletedAt timestamp
	// 2. Or remove all UserAccount relationships
	// 3. Or deactivate the user with an IsActive flag

	return fmt.Errorf("user deletion not yet implemented")
}
