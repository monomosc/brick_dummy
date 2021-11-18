package wfe

import (
	"brick/brickweb/acme"
	"context"
	"fmt"
	"time"

	opentracing "github.com/opentracing/opentracing-go"
)

func (wfe *WebFrontEndImpl) getAuthzJSON(ctx context.Context, ID string, requestingAccountID string) (*acme.Authorization, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "getAuthzJson")
	defer span.Finish()
	authz, err := wfe.db.GetAuthorizationByID(ctx, ID)
	if err != nil {
		span.SetTag("error", true)
		span.LogKV("event", "error", "error.object", err, "message", "Could not GetAuthz")
		return nil, err
	}
	if requestingAccountID != authz.AccountID {
		unauthorized := func() (*acme.Authorization, error) {
			span.SetTag("error", true)
			span.LogKV("event", "error", "error.message", "AccountID and authorizationAccountID do not match",
				"accountID", requestingAccountID, "authz.AccountID", authz.AccountID, "wfe.ProhibitGet", wfe.ProhibitGet)
			return nil, acme.UnauthorizedProblem("You do not own this Authorization")
		}
		if requestingAccountID == "" {
			if wfe.ProhibitGet {
				return unauthorized()
			}
		} else {
			return unauthorized()
		}
	}
	challenges := make([]*acme.Challenge, 0)
	switch authz.Status {
	case acme.StatusPending:
		for _, c := range authz.Challenges {
			validated := ""
			if !c.ValidatedAt.IsZero() {
				validated = c.ValidatedAt.Format(time.RFC3339)
			}
			challenges = append(challenges, &acme.Challenge{
				Type:      c.Type,
				URL:       wfe.RelativePath(fmt.Sprintf("%s%s", challengePath, c.ID)),
				Token:     c.Token,
				Status:    c.Status,
				Validated: validated,
				Error:     c.Error,
			})
		}
	case acme.StatusValid:
		for _, c := range authz.Challenges {
			validated := ""
			if !c.ValidatedAt.IsZero() {
				validated = c.ValidatedAt.Format(time.RFC3339)
			}
			if c.Status == acme.StatusValid {
				challenges = append(challenges, &acme.Challenge{
					Type:      c.Type,
					URL:       wfe.RelativePath(fmt.Sprintf("%s%s", challengePath, c.ID)),
					Token:     c.Token,
					Status:    c.Status,
					Validated: validated,
					Error:     c.Error,
				})
			}
		}
	case acme.StatusInvalid:
		for _, c := range authz.Challenges {
			validated := ""
			if !c.ValidatedAt.IsZero() {
				validated = c.ValidatedAt.Format(time.RFC3339)
			}
			if c.Status == acme.StatusInvalid {
				challenges = append(challenges, &acme.Challenge{
					Type:      c.Type,
					URL:       wfe.RelativePath(fmt.Sprintf("%s%s", challengePath, c.ID)),
					Token:     c.Token,
					Status:    c.Status,
					Validated: validated,
					Error:     c.Error,
				})
			}
		}
	}
	return &acme.Authorization{
		Status:     authz.Status,
		Identifier: authz.Identifier,
		Expires:    authz.ExpiresDate.Format(time.RFC3339),
		Wildcard:   authz.Wildcard,
		Challenges: challenges,
	}, nil
}
