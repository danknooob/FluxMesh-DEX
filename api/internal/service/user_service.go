package service

import (
	"context"
	"errors"
	"log"
	"strings"
	"time"

	"github.com/danknooob/fluxmesh-dex/api/internal/kafka"
	"github.com/danknooob/fluxmesh-dex/api/internal/models"
	"github.com/danknooob/fluxmesh-dex/api/internal/repository"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

var (
	ErrEmailTaken         = errors.New("email already registered")
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrUserNotFound       = errors.New("user not found")
)

type UpdateProfileRequest struct {
	Name      *string `json:"name"`
	Email     *string `json:"email"`
	AvatarURL *string `json:"avatar_url"`
}

type UserService interface {
	Register(email, password string, role models.UserRole) (*models.User, error)
	Authenticate(email, password string) (*models.User, error)
	GetProfile(ctx context.Context, userID uuid.UUID) (*models.User, error)
	UpdateProfile(ctx context.Context, userID uuid.UUID, req UpdateProfileRequest) (*models.User, error)
	DeleteAccount(ctx context.Context, userID uuid.UUID) error
}

type userService struct {
	repo     repository.UserRepository
	producer *kafka.Producer
}

func NewUserService(repo repository.UserRepository, producer *kafka.Producer) UserService {
	return &userService{repo: repo, producer: producer}
}

// Register atomically checks email uniqueness and inserts the user
// inside a single stored function call, eliminating the TOCTOU window.
func (s *userService) Register(email, password string, role models.UserRole) (*models.User, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	user := &models.User{
		Email:        email,
		PasswordHash: string(hash),
		Role:         role,
	}
	if err := s.repo.RegisterAtomic(user); err != nil {
		if isPgException(err, "EMAIL_TAKEN") {
			return nil, ErrEmailTaken
		}
		return nil, err
	}
	return user, nil
}

func (s *userService) Authenticate(email, password string) (*models.User, error) {
	user, err := s.repo.FindByEmail(email)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	return user, nil
}

func (s *userService) GetProfile(ctx context.Context, userID uuid.UUID) (*models.User, error) {
	user, err := s.repo.FindByID(userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return user, nil
}

// UpdateProfile atomically validates email uniqueness (row-locked)
// and applies changes via fn_update_profile_atomic.
func (s *userService) UpdateProfile(ctx context.Context, userID uuid.UUID, req UpdateProfileRequest) (*models.User, error) {
	oldUser, err := s.repo.FindByID(userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	var email, name, avatarURL *string
	if req.Email != nil {
		e := strings.TrimSpace(strings.ToLower(*req.Email))
		email = &e
	}
	if req.Name != nil {
		n := strings.TrimSpace(*req.Name)
		name = &n
	}
	if req.AvatarURL != nil {
		avatarURL = req.AvatarURL
	}

	updated, err := s.repo.UpdateProfileAtomic(ctx, userID, email, name, avatarURL)
	if err != nil {
		if isPgException(err, "EMAIL_TAKEN") {
			return nil, ErrEmailTaken
		}
		if isPgException(err, "USER_NOT_FOUND") {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	changes := map[string]interface{}{}
	if email != nil && *email != oldUser.Email {
		changes["old_email"] = oldUser.Email
		changes["new_email"] = *email
	}
	if name != nil && *name != oldUser.Name {
		changes["name"] = *name
	}
	if avatarURL != nil && *avatarURL != oldUser.AvatarURL {
		changes["avatar_url"] = *avatarURL
	}

	if len(changes) > 0 {
		changes["user_id"] = userID.String()
		changes["action"] = "profile_updated"
		changes["timestamp"] = time.Now().UTC().Format(time.RFC3339)
		if err := s.producer.PublishUserUpdated(ctx, changes); err != nil {
			log.Printf("kafka: publish users.updated failed: %v", err)
		}
	}

	return updated, nil
}

func (s *userService) DeleteAccount(ctx context.Context, userID uuid.UUID) error {
	user, err := s.repo.FindByID(userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrUserNotFound
		}
		return err
	}

	if err := s.repo.SoftDelete(userID); err != nil {
		return err
	}

	event := map[string]interface{}{
		"user_id":   user.ID.String(),
		"email":     user.Email,
		"action":    "account_deleted",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}
	if err := s.producer.PublishUserDeleted(ctx, event); err != nil {
		log.Printf("kafka: publish users.deleted failed: %v", err)
	}

	return nil
}

// isPgException checks if the error message from a PL/pgSQL RAISE
// contains the given sentinel string.
func isPgException(err error, sentinel string) bool {
	return err != nil && strings.Contains(err.Error(), sentinel)
}
