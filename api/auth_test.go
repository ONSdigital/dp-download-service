package api

import (
	"context"
	"errors"
	"net/http"
	"testing"

	authMock "github.com/ONSdigital/dp-authorisation/v2/authorisation/mock"
	dprequest "github.com/ONSdigital/dp-net/v3/request"
	permissionsAPISDK "github.com/ONSdigital/dp-permissions-api/sdk"
	"github.com/ONSdigital/log.go/v2/log"
	. "github.com/smartystreets/goconvey/convey"
	"github.com/stretchr/testify/assert"
)

var (
	testAuthToken = "test-token"
)

func TestGetAccessTokenFromRequest(t *testing.T) {
	testCases := []struct {
		name                string
		authorizationHeader string
		accessTokenCookie   *http.Cookie
		expectedToken       string
	}{
		{
			name:                "Token only in Authorization header",
			authorizationHeader: dprequest.BearerPrefix + testAuthToken,
			expectedToken:       testAuthToken,
		},
		{
			name:              "Token only in cookie",
			accessTokenCookie: &http.Cookie{Name: dprequest.FlorenceCookieKey, Value: testAuthToken},
			expectedToken:     testAuthToken,
		},
		{
			name:                "Token in both header and cookie, header value is used",
			authorizationHeader: dprequest.BearerPrefix + testAuthToken,
			accessTokenCookie:   &http.Cookie{Name: dprequest.FlorenceCookieKey, Value: "other-token"},
			expectedToken:       testAuthToken,
		},
		{
			name:          "No token in header or cookie",
			expectedToken: "",
		},
	}

	Convey("Testing getAccessTokenFromRequest", t, func() {
		for _, tc := range testCases {
			Convey(tc.name, func() {
				req, err := http.NewRequest("GET", "http://example.com", http.NoBody)
				So(err, ShouldBeNil)

				if tc.authorizationHeader != "" {
					req.Header.Set(dprequest.AuthHeaderKey, tc.authorizationHeader)
				}
				if tc.accessTokenCookie != nil {
					req.AddCookie(tc.accessTokenCookie)
				}

				token := getAccessTokenFromRequest(req)
				So(token, ShouldEqual, tc.expectedToken)
			})
		}
	})
}

func TestGetAuthEntityData(t *testing.T) {

	Convey("Testing getAuthEntityData invalid JWT token error", t, func() {
		authorisationMock := &authMock.MiddlewareMock{
			ParseFunc: func(token string) (*permissionsAPISDK.EntityData, error) {
				return nil, errors.New("parse error")
			},
		}

		authEntityData, err := getAuthEntityData(context.Background(), authorisationMock, "invalid.token", nil)
		assert.NotNil(t, err)
		assert.Nil(t, authEntityData)
	})

	Convey("Testing getAuthEntityData returns success", t, func() {
		authorisationMock := &authMock.MiddlewareMock{
			ParseFunc: func(token string) (*permissionsAPISDK.EntityData, error) {
				return &permissionsAPISDK.EntityData{UserID: "user-1"}, nil
			},
		}

		authEntityData, err := getAuthEntityData(context.Background(), authorisationMock, "valid.test-token", nil)
		assert.NotNil(t, authEntityData)
		assert.Nil(t, err)
		assert.Equal(t, authEntityData.UserID, "user-1")
	})
}

func TestCheckUserPermissions(t *testing.T) {

	permissionAttrs := map[string]string{
		"dataset_edition": "test-dataset/edition1",
	}

	Convey("Testing user does not have permissions for a specific dataset/edition", t, func() {

		permissionsChecker := &authMock.PermissionsCheckerMock{
			HasPermissionFunc: func(ctx context.Context, entityData permissionsAPISDK.EntityData, permission string, attributes map[string]string) (bool, error) {
				return false, nil
			},
		}

		entityData := permissionsAPISDK.EntityData{UserID: "user-1"}

		authorised := checkUserPermission(context.Background(), log.Data{}, "static-files:read", permissionAttrs, permissionsChecker, &entityData)
		assert.False(t, authorised)
	})

	Convey("Testing user has permissions for a specific dataset/edition", t, func() {

		permissionsChecker := &authMock.PermissionsCheckerMock{
			HasPermissionFunc: func(ctx context.Context, entityData permissionsAPISDK.EntityData, permission string, attributes map[string]string) (bool, error) {
				return true, nil
			},
		}

		entityData := permissionsAPISDK.EntityData{UserID: "user-1"}

		authorised := checkUserPermission(context.Background(), log.Data{}, "static-files:read", permissionAttrs, permissionsChecker, &entityData)
		assert.True(t, authorised)
	})
}
