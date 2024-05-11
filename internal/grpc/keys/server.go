package keys

import (
	"context"
	tmsv1 "github.com/alexprishmont/masters-protos/gen/go/trustmanagement"
	"github.com/miekg/pkcs11"
	"golang.org/x/exp/slog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"tms/internal/services/crypto"
)

type serverAPI struct {
	tmsv1.UnimplementedKeysServiceServer
	log      *slog.Logger
	session  pkcs11.SessionHandle
	operator crypto.Operator
}

func Register(
	gRPC *grpc.Server,
	log *slog.Logger,
	operator crypto.Operator,
) {
	tmsv1.RegisterKeysServiceServer(gRPC, &serverAPI{
		log:      log,
		operator: operator,
	})
}

func (s *serverAPI) CreateKeyPair(
	ctx context.Context,
	request *tmsv1.CreateKeyPairRequest,
) (*tmsv1.KeyPair, error) {
	err := s.operator.GenerateKeyPair(
		ctx,
		request.GetUserId(),
		request.GetKeyLabel(),
	)

	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &tmsv1.KeyPair{
		UserId:   request.GetUserId(),
		KeyLabel: request.GetKeyLabel(),
	}, nil
}

func (s *serverAPI) DeleteKeyPair(
	ctx context.Context,
	request *tmsv1.GetKeyPairRequest,
) (*tmsv1.KeyPair, error) {
	err := s.operator.DeleteKeyPair(
		ctx,
		request.GetUserId(),
		request.GetKeyLabel(),
	)

	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &tmsv1.KeyPair{
		UserId:   request.GetUserId(),
		KeyLabel: "Deleted.",
	}, nil
}
