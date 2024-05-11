package signature_issuer

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
	tmsv1.UnimplementedSignatureIssuerServiceServer
	log           *slog.Logger
	issuerService IssuerService
}

type IssuerService interface {
	SignData(
		ctx context.Context,
		keyLabel string,
		userId string,
		documentId string,
	) (models.Signature, error)
	VerifySignature(
		ctx context.Context,
		signature string,
		documentId string,
		keyLabel string,
		userId string,
	) (models.Signature, error)
}

func Register(
	gRPC *grpc.Server,
	log *slog.Logger,
	issuerService IssuerService,
) {
	tmsv1.RegisterSignatureIssuerServiceServer(gRPC, &serverAPI{
		log:           log,
		issuerService: issuerService,
	})
}

func (s *serverAPI) Sign(
	ctx context.Context,
	request *tmsv1.SignRequest,
) (*tmsv1.SignResponse, error) {
	signature, err := s.issuerService.SignData(
		ctx,
		request.GetKeyLabel(),
		request.GetUserId(),
		request.GetDocumentId(),
	)

	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &tmsv1.SignResponse{
		Valid:      signature.Valid,
		Signature:  signature.Signature,
		DocumentId: request.GetDocumentId(),
		UserId:     request.GetUserId(),
	}, nil
}

func (s *serverAPI) ValidateSignature(
	ctx context.Context,
	request *tmsv1.ValidateSignatureRequest,
) (*tmsv1.SignResponse, error) {
	signature, err := s.issuerService.VerifySignature(
		ctx,
		request.GetSignature(),
		request.GetDocumentId(),
		request.GetKeyLabel(),
		request.GetUserId(),
	)

	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &tmsv1.SignResponse{
		Valid:      signature.Valid,
		Signature:  signature.Signature,
		DocumentId: request.GetDocumentId(),
		UserId:     request.GetUserId(),
	}, nil
}
