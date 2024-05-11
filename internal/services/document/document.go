package document

import (
	"context"
	"fmt"
	"golang.org/x/exp/slog"
	"tms/internal/domain/models"
)

type DocumentService struct {
	log              *slog.Logger
	documentProvider Provider
}

type Provider interface {
	SaveDocument(ctx context.Context,
		title string,
		ownerId string,
		content string,
	) (string, error)
	GetDocument(ctx context.Context, id string) (models.Document, error)
	UpdateDocument(ctx context.Context, id string, title string, content string, ownerId string) (models.Document, error)
	DeleteDocument(ctx context.Context, id string) (bool, error)
}

func New(
	log *slog.Logger,
	documentProvider Provider,
) *DocumentService {
	return &DocumentService{
		log:              log,
		documentProvider: documentProvider,
	}
}

func (d *DocumentService) CreateDocument(
	ctx context.Context,
	title string,
	ownerId string,
	content string,
) (string, error) {
	const op = "services.document.CreateDocument"

	id, err := d.documentProvider.SaveDocument(
		ctx,
		title,
		ownerId,
		content,
	)

	if err != nil {
		return "", fmt.Errorf("%s: %w", op, err)
	}

	return id, nil
}

func (d *DocumentService) Document(
	ctx context.Context,
	id string,
) (models.Document, error) {
	const op = "services.document.Document"

	document, err := d.documentProvider.GetDocument(ctx, id)

	if err != nil {
		return models.Document{}, fmt.Errorf("%s: %w", op, err)
	}

	return document, nil
}
func (d *DocumentService) UpdateDocument(
	ctx context.Context,
	id string,
	title string,
	content string,
	ownerId string,
) (models.Document, error) {
	const op = "services.document.UpdateDocument"

	document, err := d.documentProvider.UpdateDocument(ctx, id, title, content, ownerId)

	if err != nil {
		return models.Document{}, fmt.Errorf("%s: %w", op, err)
	}

	return document, nil
}
func (d *DocumentService) DeleteDocument(ctx context.Context, id string) (bool, error) {
	const op = "services.document.DeleteDocument"

	success, err := d.documentProvider.DeleteDocument(ctx, id)

	if err != nil {
		return false, fmt.Errorf("%s: %w", op, err)
	}

	return success, nil
}
