package signature_issuer

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	blockchainv1 "github.com/alexprishmont/masters-protos/gen/go/blockchain-processor"
	"golang.org/x/exp/slog"
	"google.golang.org/grpc"
	"tms/internal/domain/models"
	"tms/internal/services/crypto"
)

type IssuerService struct {
	log                 *slog.Logger
	cryptoOperator      crypto.Operator
	documentProvider    Provider
	blockchainProcessor blockchainv1.BlockchainProcessorClient
}

type Provider interface {
	GetDocument(ctx context.Context, id string) (models.Document, error)
}

func New(
	log *slog.Logger,
	cryptoOperator crypto.Operator,
	documentProvider Provider,
) *IssuerService {
	conn, err := grpc.Dial("localhost:44046", grpc.WithInsecure())
	if err != nil {
		log.Error("could not connect to blockchain processor: %v", err)
	}

	client := blockchainv1.NewBlockchainProcessorClient(conn)

	return &IssuerService{
		log:                 log,
		cryptoOperator:      cryptoOperator,
		documentProvider:    documentProvider,
		blockchainProcessor: client,
	}
}

func (s *IssuerService) SignData(
	ctx context.Context,
	keyLabel string,
	userId string,
	documentId string,
) (models.Signature, error) {
	const op = "services.signature_issuer.SignData"
	// get document
	document, err := s.documentProvider.GetDocument(ctx, documentId)

	if err != nil {
		return models.Signature{}, fmt.Errorf("%s: %w", op, err)
	}

	// sign document
	documentContent := []byte(document.Content)
	signature, err := s.cryptoOperator.SignData(keyLabel, documentContent)

	if err != nil {
		return models.Signature{}, fmt.Errorf("%s: %w", op, err)
	}

	// send signature to blockchain processor to save
	req := &blockchainv1.SaveRequest{
		Id:        fmt.Sprintf("%s-%s-%s", documentId, userId, keyLabel),
		Signature: base64.StdEncoding.EncodeToString(signature),
	}

	res, err := s.blockchainProcessor.SaveSignature(ctx, req)

	if err != nil {
		return models.Signature{}, fmt.Errorf("%s: failed to send signature to blockchain. (%w)", op, err)
	}

	if !res.Success {
		return models.Signature{}, fmt.Errorf("%s: failed to save signature (%w)", op, err)
	}

	return models.Signature{
		Signature: base64.StdEncoding.EncodeToString(signature),
		Valid:     true,
	}, nil
}

func (s *IssuerService) VerifySignature(
	ctx context.Context,
	signature string,
	documentId string,
	keyLabel string,
	userId string,
) (models.Signature, error) {
	const op = "services.signature_issuer.VerifySignature"

	document, err := s.documentProvider.GetDocument(ctx, documentId)

	if err != nil {
		return models.Signature{}, fmt.Errorf("%s: %w", op, err)
	}

	// send data & signature to blockchain processor for verification
	req := &blockchainv1.GetRequest{
		Id: fmt.Sprintf("%s-%s-%s", documentId, userId, keyLabel),
	}

	res, err := s.blockchainProcessor.GetSignature(ctx, req)

	if err != nil {
		return models.Signature{}, fmt.Errorf("%s: failed to get signature from blockchain", op)
	}

	providedSign, err := base64.StdEncoding.DecodeString(signature)

	if err != nil {
		return models.Signature{}, fmt.Errorf("%s: %w", op, err)
	}

	sign, err := base64.StdEncoding.DecodeString(res.Signature)

	if err != nil {
		return models.Signature{}, fmt.Errorf("%s: %w", op, err)
	}

	if !bytes.Equal(providedSign, sign) {
		return models.Signature{
			Signature: signature,
			Valid:     false,
		}, nil
	}

	success, err := s.cryptoOperator.VerifySignature(
		keyLabel,
		[]byte(document.Content),
		sign,
	)

	if err != nil {
		return models.Signature{}, fmt.Errorf("%s: %w", op, err)
	}

	return models.Signature{
		Signature: res.Signature,
		Valid:     success,
	}, nil
}
