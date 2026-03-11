package dbseed

import (
	"log"

	"github.com/danknooob/fluxmesh-dex/api/internal/models"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type seedUser struct {
	Email    string
	Password string
	Role     models.UserRole
}

var defaultUsers = []seedUser{
	{Email: "admin@example.com", Password: "admin123", Role: models.UserRoleAdmin},
	{Email: "trader@example.com", Password: "trader123", Role: models.UserRoleTrader},
}

// SeedDefaultUsers inserts default dev users if they don't already exist.
func SeedDefaultUsers(db *gorm.DB) {
	for _, su := range defaultUsers {
		var count int64
		db.Model(&models.User{}).Where("email = ?", su.Email).Count(&count)
		if count > 0 {
			continue
		}

		hash, err := bcrypt.GenerateFromPassword([]byte(su.Password), bcrypt.DefaultCost)
		if err != nil {
			log.Printf("seed users: hash error for %s: %v", su.Email, err)
			continue
		}

		user := models.User{
			Email:        su.Email,
			PasswordHash: string(hash),
			Role:         su.Role,
		}
		if err := db.Create(&user).Error; err != nil {
			log.Printf("seed users: create error for %s: %v", su.Email, err)
			continue
		}
		log.Printf("seed users: created %s (%s)", su.Email, su.Role)
	}
}
