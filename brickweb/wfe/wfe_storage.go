package wfe

import (
	"brick/brickweb/acme"
	"brick/core"
	"context"
	"crypto/x509"
	"math/big"
)

//storage is the interface needed to perform all tasks
type storage interface {
	GetAccountByID(context.Context, string) (*core.Account, error)
	GetOrderByID(context.Context, string) (*core.Order, error)
	GetAuthorizationByID(context.Context, string) (*core.Authorization, error)
	AddAccount(context.Context, *core.Account) error
	UpdateAccount(context.Context, *core.Account) error
	GetAuthFromIdent(context.Context, acme.Identifier, *core.Account) (*core.Authorization, error)
	AddOrder(context.Context, core.AddOrderRequest) (string, error)
	AddAuthorization(context.Context, core.AddAuthz) (string, error)
	UpdateOrder(context.Context, *core.Order) error
	GetCertificateAndChain(context.Context, string) (*core.Certificate, []*x509.Certificate, error)
	GetChallengeByID(context.Context, string) (*core.Challenge, string, string, error)
	UpdateChallengeStatus(context.Context, string, string) error
	UpdateAuthorization(context.Context, *core.Challenge, string, string) error
	GetCertificateBySerial(context.Context, *big.Int, []byte) (*core.Certificate, error)
	RevokeCertificate(context.Context, string, int) error
}
