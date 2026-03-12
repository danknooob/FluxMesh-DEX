package repository

import (
	"github.com/danknooob/fluxmesh-dex/api/internal/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type UserRepository interface {
	Create(user *models.User) error
	FindByEmail(email string) (*models.User, error)
	FindByID(id uuid.UUID) (*models.User, error)
	Update(user *models.User) error
	SoftDelete(id uuid.UUID) error
}

type userRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) UserRepository {
	return &userRepository{db: db}
}

func (r *userRepository) Create(user *models.User) error {
	return r.db.
		Raw(
			"SELECT * FROM fn_create_user($1,$2,$3,$4,$5)",
			user.Email, user.Name, user.AvatarURL,
			user.PasswordHash, string(user.Role),
		).Scan(user).Error
}

func (r *userRepository) FindByEmail(email string) (*models.User, error) {
	var user models.User
	result := r.db.
		Raw("SELECT * FROM fn_find_user_by_email($1)", email).
		Scan(&user)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	return &user, nil
}

func (r *userRepository) FindByID(id uuid.UUID) (*models.User, error) {
	var user models.User
	result := r.db.
		Raw("SELECT * FROM fn_find_user_by_id($1)", id.String()).
		Scan(&user)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	return &user, nil
}

func (r *userRepository) Update(user *models.User) error {
	return r.db.
		Exec(
			"SELECT fn_update_user($1,$2,$3,$4,$5,$6)",
			user.ID.String(), user.Email, user.Name,
			user.AvatarURL, user.PasswordHash, string(user.Role),
		).Error
}

func (r *userRepository) SoftDelete(id uuid.UUID) error {
	return r.db.
		Exec("SELECT fn_soft_delete_user($1)", id.String()).Error
}
