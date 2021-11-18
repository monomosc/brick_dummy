package wfe_test

import (
	"brick/brickweb/acme"
	"brick/core"
	"context"
	"crypto/x509"
	"math/big"
)

type mockCA struct {
}
type mockStorage struct {
}
type mockVa struct {
}

func (c *mockVa) DoValidation(ctx context.Context, v *core.VerificationRequest) error {
	return nil
}

func (c *mockCA) CompleteOrder(ctx context.Context, o *core.Order, r *x509.CertificateRequest) error {
	return nil
}

func (c *mockStorage) RevokeCertificate(context.Context, string, int) error {
	return nil
}
func (c *mockStorage) GetCertificateBySerial(context.Context, *big.Int, []byte) (*core.Certificate, error) {
	return nil, nil
}
func (c *mockStorage) GetAccountByID(context.Context, string) (*core.Account, error) {
	return nil, nil
}
func (c *mockStorage) GetOrderByID(context.Context, string) (*core.Order, error) {
	return nil, nil
}
func (c *mockStorage) GetAuthorizationByID(context.Context, string) (*core.Authorization, error) {
	return nil, nil
}
func (c *mockStorage) AddAccount(context.Context, *core.Account) error {
	return nil
}
func (c *mockStorage) UpdateAccount(context.Context, *core.Account) error {
	return nil
}
func (c *mockStorage) GetAuthFromIdent(context.Context, acme.Identifier, *core.Account) (*core.Authorization, error) {
	return nil, nil
}
func (c *mockStorage) AddOrder(context.Context, core.AddOrderRequest) (string, error) {
	return "", nil
}
func (c *mockStorage) AddAuthorization(context.Context, core.AddAuthz) (string, error) {
	return "", nil
}
func (c *mockStorage) UpdateOrder(context.Context, *core.Order) error {
	return nil
}
func (c *mockStorage) GetCertificateAndChain(context.Context, string) (*core.Certificate, []*x509.Certificate, error) {
	return nil, nil, nil
}
func (c *mockStorage) GetChallengeByID(context.Context, string) (*core.Challenge, string, string, error) {
	return nil, "", "", nil
}
func (c *mockStorage) UpdateChallengeStatus(context.Context, string, string) error {
	return nil
}
func (c *mockStorage) UpdateAuthorization(context.Context, *core.Challenge, string, string) error {
	return nil
}
