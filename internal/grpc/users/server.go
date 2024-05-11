package users

import (
	"context"
	tmsv1 "github.com/alexprishmont/masters-protos/gen/go/trustmanagement"
	"golang.org/x/exp/slog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"tms/internal/domain/models"
)

type serverAPI struct {
	tmsv1.UnimplementedUsersServiceServer
	log          *slog.Logger
	usersService UserService
}

type UserService interface {
	UpdateUser(ctx context.Context, id string, name string, email string) (models.User, error)
	DeleteUser(ctx context.Context, id string) (bool, error)
	User(ctx context.Context, id string) (models.User, error)
}

func Register(
	gRPC *grpc.Server,
	log *slog.Logger,
	usersService UserService,
) {
	tmsv1.RegisterUsersServiceServer(gRPC, &serverAPI{
		log:          log,
		usersService: usersService,
	})
}

func (s *serverAPI) UpdateUser(
	ctx context.Context,
	request *tmsv1.UpdateUserRequest,
) (*tmsv1.User, error) {
	user, err := s.usersService.UpdateUser(
		ctx,
		request.GetId(),
		request.GetName(),
		request.GetEmail(),
	)

	if err != nil {
		return nil, status.Error(codes.Internal, "User update process failed.")
	}

	return &tmsv1.User{
		Id:    user.UniqueId,
		Name:  user.Name,
		Email: user.Email,
	}, nil
}

func (s *serverAPI) DeleteUser(
	ctx context.Context,
	request *tmsv1.GetUserRequest,
) (*tmsv1.User, error) {
	success, err := s.usersService.DeleteUser(
		ctx,
		request.GetId(),
	)

	if err != nil {
		return nil, status.Error(codes.Internal, "User delete process failed.")
	}

	if success == false {
		return nil, status.Error(codes.Internal, "User delete process has failed with status - FALSE")
	}

	return &tmsv1.User{
		Id:    request.GetId(),
		Name:  "Removed.",
		Email: "Removed.",
	}, nil
}

func (s *serverAPI) GetUser(
	ctx context.Context,
	request *tmsv1.GetUserRequest,
) (*tmsv1.User, error) {
	user, err := s.usersService.User(
		ctx,
		request.GetId(),
	)

	if err != nil {
		return nil, status.Error(codes.Internal, "User get process failed.")
	}

	return &tmsv1.User{
		Id:    user.UniqueId,
		Name:  user.Name,
		Email: user.Email,
	}, nil
}
