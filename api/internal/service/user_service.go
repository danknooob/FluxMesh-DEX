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

func (s *userService) Register(email, password string, role models.UserRole) (*models.User, error) {
	existing, err := s.repo.FindByEmail(email)
	if err == nil && existing != nil {
		return nil, ErrEmailTaken
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	user := &models.User{
		Email:        email,
		PasswordHash: string(hash),
		Role:         role,
	}
	if err := s.repo.Create(user); err != nil {
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

func (s *userService) UpdateProfile(ctx context.Context, userID uuid.UUID, req UpdateProfileRequest) (*models.User, error) {
	user, err := s.repo.FindByID(userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	changes := map[string]interface{}{}

	if req.Name != nil {
		user.Name = strings.TrimSpace(*req.Name)
		changes["name"] = user.Name
	}
	if req.Email != nil {
		newEmail := strings.TrimSpace(strings.ToLower(*req.Email))
		if newEmail != user.Email {
			existing, findErr := s.repo.FindByEmail(newEmail)
			if findErr == nil && existing != nil && existing.ID != user.ID {
				return nil, ErrEmailTaken
			}
			changes["old_email"] = user.Email
			changes["new_email"] = newEmail
			user.Email = newEmail
		}
	}
	if req.AvatarURL != nil {
		user.AvatarURL = *req.AvatarURL
		changes["avatar_url"] = user.AvatarURL
	}

	if err := s.repo.Update(user); err != nil {
		return nil, err
	}

	if len(changes) > 0 {
		changes["user_id"] = user.ID.String()
		changes["action"] = "profile_updated"
		changes["timestamp"] = time.Now().UTC().Format(time.RFC3339)
		if err := s.producer.PublishUserUpdated(ctx, changes); err != nil {
			log.Printf("kafka: publish users.updated failed: %v", err)
		}
	}

	return user, nil
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
