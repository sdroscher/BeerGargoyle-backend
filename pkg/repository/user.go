package repository

import (
	"context"

	"github.com/google/uuid"

	"droscher.com/BeerGargoyle/pkg/model"
)

func (r *Repository) GetUserByUUID(ctx context.Context, uuid uuid.UUID) (*model.User, error) {
	var user model.User

	result := r.DB.WithContext(ctx).Where("uuid = ?", uuid).First(&user)
	if result.Error != nil {
		return nil, result.Error
	}

	return &user, nil
}

func (r *Repository) GetUserByName(ctx context.Context, username string) (*model.User, error) {
	var user *model.User

	result := r.DB.WithContext(ctx).Where("username = ?", username).First(&user)
	if result.Error != nil {
		return nil, result.Error
	}

	return user, nil
}

func (r *Repository) GetUserFromEmail(ctx context.Context, email string) (*model.User, error) {
	var user *model.User

	result := r.DB.WithContext(ctx).Where("email = ?", email).First(&user)
	if result.Error != nil {
		return nil, result.Error
	}

	return user, nil
}

func (r *Repository) AddUser(ctx context.Context, name string, email string, untappdUserName *string) (*model.User, error) {
	user := model.User{
		UUID:            uuid.New(),
		Username:        name,
		Email:           email,
		UntappdUserName: untappdUserName,
	}

	if result := r.DB.WithContext(ctx).Create(&user); result.Error != nil {
		return nil, result.Error
	}

	return &user, nil
}
