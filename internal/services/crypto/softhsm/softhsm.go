package softhsm

import (
	"context"
	"fmt"
	"github.com/miekg/pkcs11"
)

type CryptoOperator struct {
	Pkcs11LibPath string
	TokenLabel    string
	Pin           string
	Pkcs11Ctx     *pkcs11.Ctx
	initialized   bool
	keyProvider   KeyProvider
}

type KeyProvider interface {
	SaveKeyPair(
		ctx context.Context,
		session pkcs11.SessionHandle,
		publicKeyHandle,
		privateKeyHandle pkcs11.ObjectHandle,
		publicKeyLabel string,
		privateKeyLabel string,
		userId string,
	) error
	DeleteKeys(
		ctx context.Context,
		userId string,
		publicKeyLabel string,
		privateKeyLabel string,
	) (bool, error)
}

func New(
	libPath string,
	tokenLabel string,
	pin string,
	keyProvider KeyProvider,
) *CryptoOperator {
	ctx := pkcs11.New(libPath)

	op := &CryptoOperator{
		Pkcs11LibPath: libPath,
		TokenLabel:    tokenLabel,
		Pin:           pin,
		Pkcs11Ctx:     ctx,
		keyProvider:   keyProvider,
	}

	if err := op.Pkcs11Ctx.Initialize(); err != nil {
		return &CryptoOperator{}
	}
	op.initialized = true

	return op
}

func (op *CryptoOperator) Close() error {
	if op.initialized {
		if err := op.Pkcs11Ctx.Finalize(); err != nil {
			return fmt.Errorf("error finalizing PKCS#11: %w", err)
		}
		op.initialized = false
	}
	return nil
}

func (op *CryptoOperator) OpenSession() (pkcs11.SessionHandle, error) {
	slot, err := op.findSlot()
	if err != nil {
		return 0, err
	}

	session, err := op.Pkcs11Ctx.OpenSession(slot, pkcs11.CKF_SERIAL_SESSION|pkcs11.CKF_RW_SESSION)
	if err != nil {
		return 0, fmt.Errorf("error opening session: %w", err)
	}
	if err := op.Pkcs11Ctx.Login(session, pkcs11.CKU_USER, op.Pin); err != nil {
		op.Pkcs11Ctx.CloseSession(session)
		return 0, fmt.Errorf("error logging in: %w", err)
	}
	return session, nil
}

func (op *CryptoOperator) CloseSession(session pkcs11.SessionHandle) {
	op.Pkcs11Ctx.Logout(session)
	op.Pkcs11Ctx.CloseSession(session)
}

func (op *CryptoOperator) GenerateKeyPair(
	ctx context.Context,
	session pkcs11.SessionHandle,
	userId string,
	publicKeyLabel string,
	privateKeyLabel string,
) (pkcs11.ObjectHandle, pkcs11.ObjectHandle, error) {
	pubKey, privKey, err := op.Pkcs11Ctx.GenerateKeyPair(session,
		[]*pkcs11.Mechanism{pkcs11.NewMechanism(pkcs11.CKM_RSA_PKCS_KEY_PAIR_GEN, nil)},
		[]*pkcs11.Attribute{
			pkcs11.NewAttribute(pkcs11.CKA_CLASS, pkcs11.CKO_PUBLIC_KEY),
			pkcs11.NewAttribute(pkcs11.CKA_KEY_TYPE, pkcs11.CKK_RSA),
			pkcs11.NewAttribute(pkcs11.CKA_TOKEN, true),
			pkcs11.NewAttribute(pkcs11.CKA_ENCRYPT, true),
			pkcs11.NewAttribute(pkcs11.CKA_VERIFY, true),
			pkcs11.NewAttribute(pkcs11.CKA_PUBLIC_EXPONENT, []byte{1, 0, 1}),
			pkcs11.NewAttribute(pkcs11.CKA_MODULUS_BITS, 2048),
			pkcs11.NewAttribute(pkcs11.CKA_LABEL, publicKeyLabel),
		}, []*pkcs11.Attribute{
			pkcs11.NewAttribute(pkcs11.CKA_CLASS, pkcs11.CKO_PRIVATE_KEY),
			pkcs11.NewAttribute(pkcs11.CKA_KEY_TYPE, pkcs11.CKK_RSA),
			pkcs11.NewAttribute(pkcs11.CKA_TOKEN, true),
			pkcs11.NewAttribute(pkcs11.CKA_PRIVATE, true),
			pkcs11.NewAttribute(pkcs11.CKA_SIGN, true),
			pkcs11.NewAttribute(pkcs11.CKA_DECRYPT, true),
			pkcs11.NewAttribute(pkcs11.CKA_LABEL, privateKeyLabel),
		})
	if err != nil {
		return 0, 0, fmt.Errorf("GenerateKeyPair failed: %w", err)
	}

	err = op.keyProvider.SaveKeyPair(
		ctx,
		session,
		pubKey,
		privKey,
		publicKeyLabel,
		privateKeyLabel,
		userId,
	)

	if err != nil {
		return 0, 0, fmt.Errorf("%s: %w", op, err)
	}

	return pubKey, privKey, nil
}

func (op *CryptoOperator) GenerateSignature(session pkcs11.SessionHandle, data []byte, privateKeyHandle pkcs11.ObjectHandle) ([]byte, error) {
	err := op.Pkcs11Ctx.SignInit(session, []*pkcs11.Mechanism{pkcs11.NewMechanism(pkcs11.CKM_SHA256_RSA_PKCS, nil)}, privateKeyHandle)
	if err != nil {
		return nil, fmt.Errorf("error initializing signing: %w", err)
	}
	signature, err := op.Pkcs11Ctx.Sign(session, data)
	if err != nil {
		return nil, fmt.Errorf("error signing data: %w", err)
	}
	return signature, nil
}

func (op *CryptoOperator) VerifySignature(session pkcs11.SessionHandle, data []byte, signature []byte, publicKeyHandle pkcs11.ObjectHandle) (bool, error) {
	err := op.Pkcs11Ctx.VerifyInit(session, []*pkcs11.Mechanism{pkcs11.NewMechanism(pkcs11.CKM_SHA256_RSA_PKCS, nil)}, publicKeyHandle)
	if err != nil {
		return false, fmt.Errorf("error initializing verification: %w", err)
	}
	err = op.Pkcs11Ctx.Verify(session, data, signature)
	if err != nil {
		return false, fmt.Errorf("verification failed: %w", err)
	}
	return true, nil
}

func (op *CryptoOperator) findSlot() (uint, error) {
	slots, err := op.Pkcs11Ctx.GetSlotList(true)
	if err != nil {
		return 0, fmt.Errorf("error getting slot list: %w", err)
	}
	for _, slot := range slots {
		tokenInfo, err := op.Pkcs11Ctx.GetTokenInfo(slot)
		if err == nil && tokenInfo.Label == op.TokenLabel {
			return slot, nil
		}
	}
	return 0, fmt.Errorf("token not found")
}

func (op *CryptoOperator) DeleteKeyPair(
	ctx context.Context,
	session pkcs11.SessionHandle,
	userId string,
	publicKeyLabel string,
	privateKeyLabel string,
) error {
	pubKey, err := op.findKeyByLabel(session, publicKeyLabel, pkcs11.CKO_PUBLIC_KEY)
	if err != nil {
		return fmt.Errorf("failed to find public key: %w", err)
	}

	privKey, err := op.findKeyByLabel(session, privateKeyLabel, pkcs11.CKO_PRIVATE_KEY)
	if err != nil {
		return fmt.Errorf("failed to find private key: %w", err)
	}

	err = op.Pkcs11Ctx.DestroyObject(session, pubKey)
	if err != nil {
		return fmt.Errorf("failed to delete public key: %w", err)
	}

	err = op.Pkcs11Ctx.DestroyObject(session, privKey)
	if err != nil {
		return fmt.Errorf("failed to delete private key: %w", err)
	}

	success, err := op.keyProvider.DeleteKeys(
		ctx,
		userId,
		publicKeyLabel,
		privateKeyLabel,
	)

	if success == false {
		return fmt.Errorf("failed to delete private key: %w", err)
	}

	if err != nil {
		return fmt.Errorf("failed to delete private key: %w", err)
	}

	return nil
}

func (op *CryptoOperator) findKeyByLabel(session pkcs11.SessionHandle, label string, keyClass uint) (pkcs11.ObjectHandle, error) {
	query := []*pkcs11.Attribute{
		pkcs11.NewAttribute(pkcs11.CKA_CLASS, keyClass),
		pkcs11.NewAttribute(pkcs11.CKA_LABEL, label),
	}

	if err := op.Pkcs11Ctx.FindObjectsInit(session, query); err != nil {
		return 0, fmt.Errorf("FindObjectsInit failed: %w", err)
	}
	defer op.Pkcs11Ctx.FindObjectsFinal(session)

	objects, _, err := op.Pkcs11Ctx.FindObjects(session, 1)
	if err != nil {
		return 0, fmt.Errorf("FindObjects failed: %w", err)
	}

	if len(objects) == 0 {
		return 0, fmt.Errorf("no objects found with label: %s", label)
	}

	return objects[0], nil
}
