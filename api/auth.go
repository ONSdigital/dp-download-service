package api

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/ONSdigital/dp-api-clients-go/v2/identity"
	"github.com/ONSdigital/dp-download-service/downloads"
	dprequest "github.com/ONSdigital/dp-net/v3/request"
)

// getAccessTokenFromHeaders extracts the access token from Authorization header.
// It removes the "Bearer " prefix if present.
func getAccessTokenFromHeaders(headers http.Header) string {
	return strings.TrimPrefix(headers.Get(dprequest.AuthHeaderKey), dprequest.BearerPrefix)
}

// getTokenIdentifier validates the access token returns the identifier associated with it.
// It first checks if the token is a user token, if not it checks if it's a service token.
func getTokenIdentifier(ctx context.Context, accessToken string, identityClient downloads.IdentityClient) (string, error) {
	identityResp, err := identityClient.CheckTokenIdentity(ctx, accessToken, identity.TokenTypeUser)
	if err == nil {
		return identityResp.Identifier, nil
	}

	identityResp, err = identityClient.CheckTokenIdentity(ctx, accessToken, identity.TokenTypeService)
	if err == nil {
		return identityResp.Identifier, nil
	}

	return "", fmt.Errorf("failed to validate token with identity client: %w", err)
}
