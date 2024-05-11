package crypto

import (
	"context"
	"fmt"
	"github.com/miekg/pkcs11"
	"tms/internal/storage/mongodb"
)

type Operator struct {
	Pkcs11Lib   string
	TokenLabel  string
	Pin         string
	MongoClient *mongodb.Storage
	pkcs11Ctx   *pkcs11.Ctx
	session     pkcs11.SessionHandle
	slot        uint
}

func (op *Operator) Init() error {
	// Initialize PKCS#11
	op.pkcs11Ctx = pkcs11.New(op.Pkcs11Lib)
	if err := op.pkcs11Ctx.Initialize(); err != nil {
		return fmt.Errorf("PKCS#11 initialization error: %w", err)
	}

	// Find slot by token label
	slots, err := op.pkcs11Ctx.GetSlotList(true)
	if err != nil {
		return fmt.Errorf("GetSlotList failed: %w", err)
	}
	for _, slot := range slots {
		tokenInfo, err := op.pkcs11Ctx.GetTokenInfo(slot)
		if err == nil && tokenInfo.Label == op.TokenLabel {
			op.slot = slot
			break
		}
	}

	// Open a session
	op.session, err = op.pkcs11Ctx.OpenSession(op.slot, pkcs11.CKF_SERIAL_SESSION|pkcs11.CKF_RW_SESSION)
	if err != nil {
		return fmt.Errorf("OpenSession failed: %w", err)
	}

	// Login
	if err := op.pkcs11Ctx.Login(op.session, pkcs11.CKU_USER, op.Pin); err != nil {
		return fmt.Errorf("Login failed: %w", err)
	}

	return nil
}

func (op *Operator) GenerateKeyPair(ctx context.Context, userId string, label string) error {
	publicKeyTemplate := []*pkcs11.Attribute{
		pkcs11.NewAttribute(pkcs11.CKA_CLASS, pkcs11.CKO_PUBLIC_KEY),
		pkcs11.NewAttribute(pkcs11.CKA_KEY_TYPE, pkcs11.CKK_RSA),
		pkcs11.NewAttribute(pkcs11.CKA_TOKEN, true),
		pkcs11.NewAttribute(pkcs11.CKA_ENCRYPT, true),
		pkcs11.NewAttribute(pkcs11.CKA_PUBLIC_EXPONENT, []byte{1, 0, 1}),
		pkcs11.NewAttribute(pkcs11.CKA_MODULUS_BITS, 2048),
		pkcs11.NewAttribute(pkcs11.CKA_LABEL, label),
	}
	privateKeyTemplate := []*pkcs11.Attribute{
		pkcs11.NewAttribute(pkcs11.CKA_CLASS, pkcs11.CKO_PRIVATE_KEY),
		pkcs11.NewAttribute(pkcs11.CKA_KEY_TYPE, pkcs11.CKK_RSA),
		pkcs11.NewAttribute(pkcs11.CKA_TOKEN, true),
		pkcs11.NewAttribute(pkcs11.CKA_PRIVATE, true),
		pkcs11.NewAttribute(pkcs11.CKA_LABEL, label),
	}

	_, _, err := op.pkcs11Ctx.GenerateKeyPair(op.session, []*pkcs11.Mechanism{pkcs11.NewMechanism(pkcs11.CKM_RSA_PKCS_KEY_PAIR_GEN, nil)}, publicKeyTemplate, privateKeyTemplate)
	if err != nil {
		return fmt.Errorf("GenerateKeyPair failed: %w", err)
	}

	// Store key information in MongoDB
	err = op.MongoClient.SaveKeyPair(
		ctx,
		label,
		userId,
	)

	if err != nil {
		return fmt.Errorf("Failed to store key info in MongoDB: %w", err)
	}

	return nil
}

func (op *Operator) SignData(label string, data []byte) ([]byte, error) {
	privateKeyHandle, err := op.FindKeyByLabel(label, "private")
	if err != nil {
		return nil, err
	}

	// Initialize signing operation
	err = op.pkcs11Ctx.SignInit(op.session, []*pkcs11.Mechanism{pkcs11.NewMechanism(pkcs11.CKM_SHA256_RSA_PKCS, nil)}, privateKeyHandle)
	if err != nil {
		return nil, fmt.Errorf("SignInit failed: %w", err)
	}

	// Perform the signing operation
	signature, err := op.pkcs11Ctx.Sign(op.session, data)
	if err != nil {
		return nil, fmt.Errorf("Sign failed: %w", err)
	}
	return signature, nil
}

func (op *Operator) VerifySignature(label string, data []byte, signature []byte) (bool, error) {
	publicKeyHandle, err := op.FindKeyByLabel(label, "public")
	if err != nil {
		return false, err
	}

	// Initialize verification operation
	err = op.pkcs11Ctx.VerifyInit(op.session, []*pkcs11.Mechanism{pkcs11.NewMechanism(pkcs11.CKM_SHA256_RSA_PKCS, nil)}, publicKeyHandle)
	if err != nil {
		return false, fmt.Errorf("VerifyInit failed: %w", err)
	}

	// Perform the verification
	err = op.pkcs11Ctx.Verify(op.session, data, signature)
	if err != nil {
		return false, fmt.Errorf("verify failed: %w", err)
	}
	return true, nil
}

func (op *Operator) DeleteKeyPair(ctx context.Context, userId string, label string) error {
	// Find and delete the public key
	pubKeyHandle, err := op.FindKeyByLabel(label, "public")
	if err != nil {
		return err
	}
	err = op.pkcs11Ctx.DestroyObject(op.session, pubKeyHandle)
	if err != nil {
		return fmt.Errorf("Failed to delete public key: %w", err)
	}

	// Find and delete the private key
	privKeyHandle, err := op.FindKeyByLabel(label, "private")
	if err != nil {
		return err
	}
	err = op.pkcs11Ctx.DestroyObject(op.session, privKeyHandle)
	if err != nil {
		return fmt.Errorf("Failed to delete private key: %w", err)
	}

	// Remove key info from MongoDB
	success, err := op.MongoClient.DeleteKeyPair(ctx, userId, label)

	if err != nil || success == false {
		return fmt.Errorf("Failed to remove key info from MongoDB: %w", err)
	}
	return nil
}

func (op *Operator) FindKeyByLabel(label string, keyType string) (pkcs11.ObjectHandle, error) {
	keyTypeClass := pkcs11.CKO_PUBLIC_KEY

	if keyType == "private" {
		keyTypeClass = pkcs11.CKO_PRIVATE_KEY
	}

	template := []*pkcs11.Attribute{
		pkcs11.NewAttribute(pkcs11.CKA_LABEL, label),
		pkcs11.NewAttribute(pkcs11.CKA_CLASS, keyTypeClass),
	}
	if err := op.pkcs11Ctx.FindObjectsInit(op.session, template); err != nil {
		return 0, fmt.Errorf("FindObjectsInit failed: %w", err)
	}
	defer op.pkcs11Ctx.FindObjectsFinal(op.session)

	objects, _, err := op.pkcs11Ctx.FindObjects(op.session, 1)
	if err != nil {
		return 0, fmt.Errorf("FindObjects failed: %w", err)
	}
	if len(objects) == 0 {
		return 0, fmt.Errorf("No objects found")
	}
	return objects[0], nil
}

func (op *Operator) Close() {
	if op.pkcs11Ctx != nil {
		op.pkcs11Ctx.Logout(op.session)
		op.pkcs11Ctx.CloseSession(op.session)
		op.pkcs11Ctx.Finalize()
	}
}
