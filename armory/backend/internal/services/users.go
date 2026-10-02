package services

import (
	"context"
	"errors"
	"strings"

	"golang.org/x/crypto/bcrypt"

	"armory/internal/database"
	"armory/internal/models"
)

type UserService struct {
	store *database.Store
}

func NewUserService(store *database.Store) *UserService {
	return &UserService{store: store}
}

func (s *UserService) List(ctx context.Context) ([]models.User, error) {
	return s.store.ListUsers(ctx)
}

func (s *UserService) HasLoginAdmin(ctx context.Context) (bool, error) {
	n, err := s.store.CountPasswordAdmins(ctx)
	return n > 0, err
}

func (s *UserService) SetupAdmin(ctx context.Context, name, username, password string) (models.User, error) {
	name, username = strings.TrimSpace(name), strings.ToLower(strings.TrimSpace(username))
	if name == "" {
		return models.User{}, ErrNameRequired
	}
	if username == "" {
		return models.User{}, ErrUsernameRequired
	}
	if len(password) < 8 {
		return models.User{}, ErrPasswordTooShort
	}
	if ok, err := s.HasLoginAdmin(ctx); err != nil {
		return models.User{}, err
	} else if ok {
		return models.User{}, ErrSetupComplete
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return models.User{}, err
	}
	u, err := s.store.CreateAdmin(ctx, name, username, string(hash))
	if errors.Is(err, database.ErrDuplicate) {
		return models.User{}, ErrUserExists
	}
	return u, err
}

func (s *UserService) AuthenticateAdmin(ctx context.Context, username, password string) (models.User, error) {
	u, hash, err := s.store.AdminCredentials(ctx, strings.ToLower(strings.TrimSpace(username)))
	if errors.Is(err, database.ErrNotFound) || (err == nil && bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) != nil) {
		return models.User{}, ErrInvalidLogin
	}
	return u, err
}

func (s *UserService) Get(ctx context.Context, id int64) (models.User, error) {
	u, err := s.store.GetUser(ctx, id)
	if errors.Is(err, database.ErrNotFound) {
		return models.User{}, ErrUserNotFound
	}
	return u, err
}

func (s *UserService) Create(ctx context.Context, name, serviceNo, role string) (models.User, error) {
	u := models.User{
		Name:      strings.TrimSpace(name),
		ServiceNo: strings.ToUpper(strings.TrimSpace(serviceNo)),
		Role:      role,
	}
	if u.Name == "" {
		return models.User{}, ErrNameRequired
	}
	if u.ServiceNo == "" {
		return models.User{}, ErrServiceNoRequired
	}
	if u.Role == "" {
		u.Role = "requester"
	}
	if u.Role != "admin" && u.Role != "requester" {
		return models.User{}, ErrInvalidRole
	}

	created, err := s.store.CreateUser(ctx, u)
	if errors.Is(err, database.ErrDuplicate) {
		return models.User{}, ErrUserExists
	}
	return created, err
}
