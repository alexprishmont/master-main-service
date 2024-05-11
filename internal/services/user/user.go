package user

import (
	"context"
	"fmt"
	"golang.org/x/exp/slog"
	"tms/internal/domain/models"
)

type Service struct {
	log          *slog.Logger
	userProvider Provider
}

type Provider interface {
	SaveUser(
		ctx context.Context,
		id string,
		name string,
		email string,
	) (models.User, error)

	RemoveUser(
		ctx context.Context,
		id string,
	) (bool, error)

	GetUser(
		ctx context.Context,
		id string,
	) (models.User, error)
}

func New(
	log *slog.Logger,
	userProvider Provider,
) *Service {
	return &Service{
		log:          log,
		userProvider: userProvider,
	}
}

func (s *Service) UpdateUser(
	ctx context.Context,
	id string,
	name string,
	email string,
) (models.User, error) {
	const op = "services.user.UpdateUser"

	user, err := s.userProvider.SaveUser(ctx, id, name, email)

	if err != nil {
		return models.User{}, fmt.Errorf("%s: %w", op, err)
	}

	return user, nil
}

func (s *Service) DeleteUser(
	ctx context.Context,
	id string,
) (bool, error) {
	const op = "services.user.DeleteUser"

	success, err := s.userProvider.RemoveUser(ctx, id)

	if err != nil {
		return false, fmt.Errorf("%s: %w", op, err)
	}

	return success, nil
}

func (s *Service) User(
	ctx context.Context,
	id string,
) (models.User, error) {
	const op = "services.user.User"

	user, err := s.userProvider.GetUser(ctx, id)

	if err != nil {
		return models.User{}, fmt.Errorf("%s: %w", op, err)
	}

	return user, nil
}
