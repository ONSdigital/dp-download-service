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

// getAccessTokenFromRequest retrieves the access token from the request headers or cookies.
// If no token is found, it returns an empty string.
func getAccessTokenFromRequest(r *http.Request) string {
	accessToken := r.Header.Get(dprequest.AuthHeaderKey)

	// If no access token in header, check if it is present in cookies
	if accessToken == "" {
		// The only possible error is ErrNoCookie, which is not considered an error here
		c, err := r.Cookie(dprequest.FlorenceCookieKey)
		if err != nil {
			return ""
		}
		accessToken = c.Value
	}

	return strings.TrimPrefix(accessToken, dprequest.BearerPrefix)
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
