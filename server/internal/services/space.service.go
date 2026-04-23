package services

import (
	"errors"
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"
	"unicode"

	"github.com/google/uuid"
	"github.com/tylerjvollick/nori/internal/dtos"
	"github.com/tylerjvollick/nori/internal/models"
	"github.com/tylerjvollick/nori/internal/repositories"
	"gorm.io/gorm"
)

// slugAlphaRegex matches non-letter characters for slug generation
var slugAlphaRegex = regexp.MustCompile(`[^A-Z]`)

// validSlugRegex matches valid slug format: 2-5 uppercase alphanumeric characters
var validSlugRegex = regexp.MustCompile(`^[A-Z][A-Z0-9]{1,4}$`)

type SpaceService struct {
	spaceRepository       *repositories.SpaceRepository
	userRepository        *repositories.UserRepository
	spaceMemberRepository *repositories.SpaceMemberRepository
}

func NewSpaceService(
	spaceRepository *repositories.SpaceRepository,
	userRepository *repositories.UserRepository,
	spaceMemberRepository *repositories.SpaceMemberRepository,
) *SpaceService {
	return &SpaceService{
		spaceRepository:       spaceRepository,
		userRepository:        userRepository,
		spaceMemberRepository: spaceMemberRepository,
	}
}

// GenerateSlug generates a slug from a space name.
// Takes first letter of each word (like Jira project keys).
// e.g. "My Workshop" -> "MW", "Default" -> "DEF"
func GenerateSlug(name string) string {
	upper := strings.ToUpper(name)
	words := strings.Fields(upper)

	if len(words) == 0 {
		return "SPC"
	}

	if len(words) >= 2 {
		// Take first letter of each word, up to 5 chars
		var slug strings.Builder
		for _, w := range words {
			// Get first alpha char
			for _, r := range w {
				if unicode.IsLetter(r) {
					slug.WriteRune(r)
					break
				}
			}
			if slug.Len() >= 5 {
				break
			}
		}
		result := slugAlphaRegex.ReplaceAllString(slug.String(), "")
		if len(result) >= 2 {
			return result
		}
	}

	// Single word: take first 3 uppercase alpha characters
	cleaned := slugAlphaRegex.ReplaceAllString(strings.ToUpper(name), "")
	if len(cleaned) < 2 {
		return "SPC"
	}
	if len(cleaned) > 3 {
		cleaned = cleaned[:3]
	}
	return cleaned
}

// ValidateSlug validates that a slug meets the format requirements.
func ValidateSlug(slug string) error {
	if len(slug) < 2 || len(slug) > 5 {
		return fmt.Errorf("slug must be between 2 and 5 characters")
	}
	if !validSlugRegex.MatchString(slug) {
		return fmt.Errorf("slug must be uppercase letters and digits, starting with a letter")
	}
	return nil
}

// ensureUniqueSlug checks if the slug is unique within the account and appends
// a numeric suffix if needed.
func (s *SpaceService) ensureUniqueSlug(slug string, accountID uuid.UUID) (string, error) {
	exists, err := s.spaceRepository.SlugExistsInAccount(slug, accountID)
	if err != nil {
		return "", err
	}
	if !exists {
		return slug, nil
	}

	// Try appending digits 2-99
	base := slug
	if len(base) > 4 {
		base = base[:4] // Leave room for suffix
	}
	for i := 2; i < 100; i++ {
		candidate := fmt.Sprintf("%s%d", base, i)
		if len(candidate) > 5 {
			// Shorten base more
			base = base[:len(base)-1]
			candidate = fmt.Sprintf("%s%d", base, i)
		}
		exists, err := s.spaceRepository.SlugExistsInAccount(candidate, accountID)
		if err != nil {
			return "", err
		}
		if !exists {
			return candidate, nil
		}
	}

	return "", fmt.Errorf("unable to generate unique slug for %q", slug)
}

// CreateSpace creates a new space for an account and adds the creating user as a member.
func (s *SpaceService) CreateSpace(accountID uuid.UUID, dto *dtos.CreateSpaceDTO, creatorUserID uuid.UUID) (*models.Space, error) {
	// Determine slug
	var slug string
	if dto.Slug != nil && *dto.Slug != "" {
		slug = strings.ToUpper(*dto.Slug)
		if err := ValidateSlug(slug); err != nil {
			return nil, err
		}
		// Check uniqueness
		exists, err := s.spaceRepository.SlugExistsInAccount(slug, accountID)
		if err != nil {
			return nil, err
		}
		if exists {
			return nil, fmt.Errorf("slug %q is already in use", slug)
		}
	} else {
		// Auto-generate from name
		generated := GenerateSlug(dto.Name)
		unique, err := s.ensureUniqueSlug(generated, accountID)
		if err != nil {
			return nil, err
		}
		slug = unique
	}

	space := &models.Space{
		ID:        uuid.New(),
		Name:      dto.Name,
		Slug:      slug,
		AccountID: accountID,
		IsDefault: false,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := s.spaceRepository.Create(space); err != nil {
		return nil, err
	}

	// Auto-add the creating user as a space member
	if err := s.addMember(creatorUserID, space.ID); err != nil {
		log.Printf("Warning: failed to add creator as space member for space %s: %v", space.ID, err)
	}

	return space, nil
}

// CreateDefaultSpace creates a default space for an account and adds the given user as a member.
func (s *SpaceService) CreateDefaultSpace(accountID uuid.UUID, creatorUserID uuid.UUID) (*models.Space, error) {
	slug, err := s.ensureUniqueSlug("DEF", accountID)
	if err != nil {
		return nil, err
	}

	space := &models.Space{
		ID:        uuid.New(),
		Name:      "Default",
		Slug:      slug,
		AccountID: accountID,
		IsDefault: true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := s.spaceRepository.Create(space); err != nil {
		return nil, err
	}

	// Auto-add the creating user as a space member
	if err := s.addMember(creatorUserID, space.ID); err != nil {
		log.Printf("Warning: failed to add creator as space member for default space %s: %v", space.ID, err)
	}

	return space, nil
}

// AddMember adds a user as a member of a space (idempotent — skips if already a member).
func (s *SpaceService) AddMember(userID uuid.UUID, spaceID uuid.UUID) error {
	return s.addMember(userID, spaceID)
}

// addMember is the internal helper that creates a SpaceMember record.
// It is idempotent: if the user is already a member, it returns nil.
func (s *SpaceService) addMember(userID uuid.UUID, spaceID uuid.UUID) error {
	if s.spaceMemberRepository == nil {
		return nil
	}

	// Check if already a member
	existing, _ := s.spaceMemberRepository.GetByUserAndSpace(userID, spaceID)
	if existing != nil {
		return nil // Already a member
	}

	member := &models.SpaceMember{
		UserID:  userID,
		SpaceID: spaceID,
	}
	return s.spaceMemberRepository.Create(member)
}

// GetSpaceByID retrieves a space by ID and verifies it belongs to the user's account
func (s *SpaceService) GetSpaceByID(spaceID uuid.UUID, accountID uuid.UUID) (*models.Space, error) {
	space, err := s.spaceRepository.FindByID(spaceID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("space not found")
		}
		return nil, err
	}

	// Verify the space belongs to the user's account
	if space.AccountID != accountID {
		return nil, fmt.Errorf("unauthorized access to space")
	}

	return space, nil
}

// GetSpaceBySlug retrieves a space by slug within an account
func (s *SpaceService) GetSpaceBySlug(slug string, accountID uuid.UUID) (*models.Space, error) {
	space, err := s.spaceRepository.FindBySlugAndAccountID(strings.ToUpper(slug), accountID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("space not found")
		}
		return nil, err
	}

	return space, nil
}

// GetSpacesByAccountID retrieves all spaces for an account
func (s *SpaceService) GetSpacesByAccountID(accountID uuid.UUID) ([]models.Space, error) {
	return s.spaceRepository.FindByAccountID(accountID)
}

// GetRecentSpaces retrieves the user's recently visited spaces
func (s *SpaceService) GetRecentSpaces(userID uuid.UUID, accountID uuid.UUID) ([]models.Space, error) {
	// Get user's recent space IDs
	user, err := s.userRepository.GetUserByID(userID)
	if err != nil {
		return nil, err
	}

	// If no recent spaces, return default space
	if len(user.RecentSpaces) == 0 {
		defaultSpace, err := s.spaceRepository.FindDefaultByAccountID(accountID)
		if err != nil {
			return []models.Space{}, nil
		}
		return []models.Space{*defaultSpace}, nil
	}

	// Fetch spaces by IDs and filter by account
	spaces := []models.Space{}
	for _, spaceID := range user.RecentSpaces {
		space, err := s.spaceRepository.FindByID(spaceID)
		if err != nil {
			continue // Skip if space not found
		}
		// Only include spaces from the current account
		if space.AccountID == accountID {
			spaces = append(spaces, *space)
		}
	}

	return spaces, nil
}

// UpdateSpace updates a space
func (s *SpaceService) UpdateSpace(spaceID uuid.UUID, accountID uuid.UUID, dto *dtos.UpdateSpaceDTO) (*models.Space, error) {
	space, err := s.GetSpaceByID(spaceID, accountID)
	if err != nil {
		return nil, err
	}

	// Update name if provided
	if dto.Name != nil {
		space.Name = *dto.Name
	}

	// Update slug if provided
	if dto.Slug != nil && *dto.Slug != "" {
		newSlug := strings.ToUpper(*dto.Slug)
		if err := ValidateSlug(newSlug); err != nil {
			return nil, err
		}
		// Check uniqueness (excluding current space)
		existing, err := s.spaceRepository.FindBySlugAndAccountID(newSlug, accountID)
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
		if existing != nil && existing.ID != spaceID {
			return nil, fmt.Errorf("slug %q is already in use", newSlug)
		}
		space.Slug = newSlug
	}

	space.UpdatedAt = time.Now()

	if err := s.spaceRepository.Update(space); err != nil {
		return nil, err
	}

	return space, nil
}

// DeleteSpace deletes a space
func (s *SpaceService) DeleteSpace(spaceID uuid.UUID, accountID uuid.UUID) error {
	space, err := s.GetSpaceByID(spaceID, accountID)
	if err != nil {
		return err
	}

	// Prevent deletion of default space
	if space.IsDefault {
		return fmt.Errorf("cannot delete default space")
	}

	// Check if it's the last space
	count, err := s.spaceRepository.CountByAccountID(accountID)
	if err != nil {
		return err
	}
	if count <= 1 {
		return fmt.Errorf("cannot delete the last space")
	}

	return s.spaceRepository.Delete(spaceID)
}

// RecordSpaceVisit records a visit to a space in the user's recent spaces
func (s *SpaceService) RecordSpaceVisit(userID uuid.UUID, spaceID uuid.UUID) error {
	user, err := s.userRepository.GetUserByID(userID)
	if err != nil {
		return err
	}

	// Remove space if it already exists in the list
	newRecentSpaces := []uuid.UUID{}
	for _, id := range user.RecentSpaces {
		if id != spaceID {
			newRecentSpaces = append(newRecentSpaces, id)
		}
	}

	// Add space to the beginning
	newRecentSpaces = append([]uuid.UUID{spaceID}, newRecentSpaces...)

	// Keep only the last 5 spaces
	if len(newRecentSpaces) > 5 {
		newRecentSpaces = newRecentSpaces[:5]
	}

	return s.userRepository.UpdateRecentSpaces(userID, newRecentSpaces)
}
