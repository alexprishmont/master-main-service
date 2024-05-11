package documents

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
	tmsv1.UnimplementedDocumentServiceServer
	log      *slog.Logger
	document Document
}

type Document interface {
	CreateDocument(ctx context.Context,
		title string,
		ownerId string,
		content string,
	) (string, error)
	Document(ctx context.Context, id string) (models.Document, error)
	UpdateDocument(ctx context.Context, id string, title string, content string, ownerId string) (models.Document, error)
	DeleteDocument(ctx context.Context, id string) (bool, error)
}

func Register(
	gRPC *grpc.Server,
	log *slog.Logger,
	document Document,
) {
	tmsv1.RegisterDocumentServiceServer(gRPC, &serverAPI{
		log:      log,
		document: document,
	})
}

func (s *serverAPI) CreateDocument(
	ctx context.Context,
	request *tmsv1.CreateRequest,
) (*tmsv1.Document, error) {
	id, err := s.document.CreateDocument(
		ctx,
		request.GetTitle(),
		request.GetOwnerId(),
		request.GetContent(),
	)

	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	return &tmsv1.Document{
		Id:      id,
		Title:   request.GetTitle(),
		Content: request.GetContent(),
		Owner: &tmsv1.Owner{
			Id: request.GetOwnerId(),
		},
	}, nil
}

func (s *serverAPI) GetDocument(
	ctx context.Context,
	request *tmsv1.GetRequest,
) (*tmsv1.Document, error) {
	document, err := s.document.Document(ctx, request.GetId())

	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "Invalid request")
	}

	return &tmsv1.Document{
		Id:      request.GetId(),
		Title:   document.Title,
		Content: document.Content,
		Owner: &tmsv1.Owner{
			Id:    document.Owner.Id,
			Name:  document.Owner.Name,
			Email: document.Owner.Email,
		},
	}, nil
}

func (s *serverAPI) UpdateDocument(
	ctx context.Context,
	request *tmsv1.UpdateRequest,
) (*tmsv1.Document, error) {
	document, err := s.document.UpdateDocument(
		ctx,
		request.GetId(),
		request.GetTitle(),
		request.GetContent(),
		request.GetOwnerId(),
	)

	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	return &tmsv1.Document{
		Id:      request.GetId(),
		Title:   document.Title,
		Content: document.Content,
		Owner: &tmsv1.Owner{
			Id:    document.Owner.Id,
			Name:  document.Owner.Name,
			Email: document.Owner.Email,
		},
	}, nil
}

func (s *serverAPI) DeleteDocument(
	ctx context.Context,
	request *tmsv1.GetRequest,
) (*tmsv1.Document, error) {
	success, err := s.document.DeleteDocument(ctx, request.GetId())

	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "Invalid request")
	}

	if success == false {
		return nil, status.Error(codes.InvalidArgument, "Deletion is not successful.")
	}

	return &tmsv1.Document{
		Id:      request.GetId(),
		Title:   "Removed.",
		Content: "Removed.",
		Owner: &tmsv1.Owner{
			Id:    "",
			Name:  "",
			Email: "",
		},
	}, nil
}
