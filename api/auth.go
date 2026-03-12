package api

import (
	"context"
	"net/http"
	"strings"

	permissionsAPISDK "github.com/ONSdigital/dp-permissions-api/sdk"

	auth "github.com/ONSdigital/dp-authorisation/v2/authorisation"
	"github.com/ONSdigital/dp-download-service/files"
	dprequest "github.com/ONSdigital/dp-net/v3/request"
	"github.com/ONSdigital/log.go/v2/log"
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

// getAuthEntityData returns the EntityData associated with the provided access token
func getAuthEntityData(ctx context.Context, authMiddleware auth.Middleware, accessToken string, logData log.Data) (*permissionsAPISDK.EntityData, error) {
	var entityData *permissionsAPISDK.EntityData
	var err error
	if strings.Contains(accessToken, ".") {
		// check JWT token
		entityData, err = authMiddleware.Parse(accessToken)
		if err != nil {
			log.Error(ctx, "authorisation failed: unable to parse jwt", err, log.Classification(log.ProtectiveMonitoring), log.Auth(log.USER, ""), logData)
			return nil, err
		}
	} else {
		// serice tokens not allowed for access to these endpoints
		log.Error(ctx, "authorisation failed: service token issue", files.ErrNotAuthorised, log.Classification(log.ProtectiveMonitoring), log.Auth(log.SERVICE, ""), logData)
		return nil, files.ErrNotAuthorised
	}

	return entityData, nil
}

// checks the user permission within a function to determine access to pre-publish data
func checkUserPermission(ctx context.Context, logData log.Data, permission string, attributes map[string]string, permissionsChecker auth.PermissionsChecker, entityData *permissionsAPISDK.EntityData) bool {
	var authorised bool

	hasPermission, err := permissionsChecker.HasPermission(ctx, *entityData, permission, attributes)
	if err != nil {
		log.Error(ctx, "permissions check errored", err, logData)
		return false
	}

	if hasPermission {
		authorised = true
	}

	logData["authenticated"] = authorised

	return authorised
}
