package main_test

import (
	"brick/brickweb/acme"
	"brick/core"
	"context"
	"crypto/x509"
	"math/big"
)

type MockWFEStorage struct {
}

func (m MockWFEStorage) RevokeCertificate(context.Context, string, int) error {
	panic("NYI")
}
func (m MockWFEStorage) GetCertificateBySerial(context.Context, *big.Int, []byte) (*core.Certificate, error) {
	panic("NYI")
}
func (m MockWFEStorage) GetAccountByID(context.Context, string) (*core.Account, error) {
	panic("NYI")
}
func (m MockWFEStorage) GetOrderByID(context.Context, string) (*core.Order, error) {
	panic("NYI")
}
func (m MockWFEStorage) GetAuthorizationByID(context.Context, string) (*core.Authorization, error) {
	panic("NYI")
}
func (m MockWFEStorage) AddAccount(context.Context, *core.Account) error {
	panic("NYI")
}
func (m MockWFEStorage) UpdateAccount(context.Context, *core.Account) error {
	panic("NYI")
}
func (m MockWFEStorage) GetAuthFromIdent(context.Context, acme.Identifier, *core.Account) (*core.Authorization, error) {
	panic("NYI")
}
func (m MockWFEStorage) AddOrder(context.Context, core.AddOrderRequest) (string, error) {
	panic("NYI")
}
func (m MockWFEStorage) AddAuthorization(context.Context, core.AddAuthz) (string, error) {
	panic("NYI")
}
func (m MockWFEStorage) UpdateOrder(context.Context, *core.Order) error {
	panic("NYI")
}
func (m MockWFEStorage) GetCertificateAndChain(context.Context, string) (*core.Certificate, []*x509.Certificate, error) {
	panic("NYI")
}
func (m MockWFEStorage) GetChallengeByID(context.Context, string) (*core.Challenge, string, string, error) {
	panic("NYI")
}
func (m MockWFEStorage) UpdateChallengeStatus(context.Context, string, string) error {
	panic("NYI")
}
func (m MockWFEStorage) UpdateAuthorization(context.Context, *core.Challenge, string, string) error {
	panic("NYI")
}

type MockWFECa struct {
}

func (c MockWFECa) CompleteOrder(context.Context, *core.Order, *x509.CertificateRequest) error {
	panic("NYI")
}

type MockWfeVa struct {
}

func (m MockWfeVa) DoValidation(context.Context, *core.VerificationRequest) error {
	panic("NYI")
}
