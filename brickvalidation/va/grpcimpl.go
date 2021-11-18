package va

import (
	"brick/core"
	"brick/core/log"
	"brick/grpc"
	"brick/grpc/corepb"
	"brick/grpc/vapb"
	"context"
	"fmt"

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ValidationService anchors the implementaton of the ACME validation service
type ValidationService struct {
	channel chan core.VerificationRequest
	logger  logrus.FieldLogger
}

// NewValidationGrpcService returns an implementation of the BrickValidation interface
func NewValidationGrpcService(channel chan core.VerificationRequest, logger logrus.FieldLogger) *ValidationService {
	impl := ValidationService{channel: channel, logger: logger}

	return &impl
}

// DoValidate translates the wireformat into the local format and performs the validation
func (v *ValidationService) DoValidate(ctx context.Context, valMsg *vapb.ValidationMessage) (*corepb.Empty, error) {

	log.WithTraceID(v.logger, ctx).Debug("Start Validation")
	verificationRequest, err := grpc.ProtoToValidation(ctx, valMsg)
	if err != nil {
		v.logger.Errorf("Error reading Validation request: %v", err)
		return nil, status.Errorf(codes.Internal, fmt.Sprintf("Error reading Validation request: %v", err))
	}

	l := log.WithTraceID(v.logger.WithFields(logrus.Fields{
		"authorizationID": verificationRequest.Authorization.ID, "challengeID": verificationRequest.Challenge.ID,
		"accountKeyID": verificationRequest.AccountJWK.KeyID, "authorizationAccountID": verificationRequest.Authorization.AccountID,
	}), ctx)

	v.channel <- *verificationRequest

	l.Info("Placed verification in channel")

	return new(corepb.Empty), nil
}
