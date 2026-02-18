package api

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/ONSdigital/dp-api-clients-go/v2/identity"
	"github.com/ONSdigital/dp-download-service/downloads/mocks"
	dprequest "github.com/ONSdigital/dp-net/v3/request"
	"github.com/golang/mock/gomock"
	. "github.com/smartystreets/goconvey/convey"
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

func TestGetTokenIdentifier(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	Convey("Given a valid user token", t, func() {
		mockIdentityClient := mocks.NewMockIdentityClient(ctrl)
		accessToken := "valid-user-token"
		expectedIdentifier := "user-123"

		mockIdentityClient.EXPECT().
			CheckTokenIdentity(ctx, accessToken, identity.TokenTypeUser).
			Return(&dprequest.IdentityResponse{Identifier: expectedIdentifier}, nil)

		Convey("When getTokenIdentifier is called", func() {
			identifier, err := getTokenIdentifier(ctx, accessToken, mockIdentityClient)

			Convey("Then the expected identifier is returned without error", func() {
				So(err, ShouldBeNil)
				So(identifier, ShouldEqual, expectedIdentifier)
			})
		})
	})

	Convey("Given a valid service token", t, func() {
		mockIdentityClient := mocks.NewMockIdentityClient(ctrl)
		accessToken := "valid-service-token"

		mockIdentityClient.EXPECT().
			CheckTokenIdentity(ctx, accessToken, identity.TokenTypeUser).
			Return(nil, fmt.Errorf("user token not valid"))

		Convey("When getTokenIdentifier is called", func() {
			identifier, err := getTokenIdentifier(ctx, accessToken, mockIdentityClient)
			Convey("Then the expected identifier is returned without error", func() {
				So(err, ShouldNotBeNil)
				So(identifier, ShouldEqual, "")
			})
		})
	})

	Convey("Given an invalid token", t, func() {
		mockIdentityClient := mocks.NewMockIdentityClient(ctrl)
		accessToken := "invalid-token"

		mockIdentityClient.EXPECT().
			CheckTokenIdentity(ctx, accessToken, identity.TokenTypeUser).
			Return(nil, fmt.Errorf("user token not valid"))

		Convey("When getTokenIdentifier is called", func() {
			identifier, err := getTokenIdentifier(ctx, accessToken, mockIdentityClient)

			Convey("Then an error is returned indicating token validation failure", func() {
				So(err.Error(), ShouldContainSubstring, "failed to validate user token with identity client")
				So(identifier, ShouldEqual, "")
			})
		})
	})
}
