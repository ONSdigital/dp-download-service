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

func TestGetAccessTokenFromHeaders(t *testing.T) {
	testCases := []struct {
		name          string
		headers       http.Header
		expectedToken string
	}{
		{
			name: "Authorization header with Bearer prefix",
			headers: http.Header{
				"Authorization": []string{"Bearer access-token"},
			},
			expectedToken: "access-token",
		},
		{
			name: "Authorization header without Bearer prefix",
			headers: http.Header{
				"Authorization": []string{"access-token"},
			},
			expectedToken: "access-token",
		},
		{
			name:          "No Authorization header",
			headers:       http.Header{},
			expectedToken: "",
		},
		{
			name:          "Nil headers",
			headers:       nil,
			expectedToken: "",
		},
	}

	for _, tc := range testCases {
		Convey("Given: "+tc.name, t, func() {
			Convey("When getAccessTokenFromHeaders is called", func() {
				token := getAccessTokenFromHeaders(tc.headers)

				Convey("Then the expected token is returned", func() {
					So(token, ShouldEqual, tc.expectedToken)
				})
			})
		})
	}
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
		expectedIdentifier := "service-123"

		mockIdentityClient.EXPECT().
			CheckTokenIdentity(ctx, accessToken, identity.TokenTypeUser).
			Return(nil, fmt.Errorf("user token not valid"))

		mockIdentityClient.EXPECT().
			CheckTokenIdentity(ctx, accessToken, identity.TokenTypeService).
			Return(&dprequest.IdentityResponse{Identifier: expectedIdentifier}, nil)

		Convey("When getTokenIdentifier is called", func() {
			identifier, err := getTokenIdentifier(ctx, accessToken, mockIdentityClient)
			Convey("Then the expected identifier is returned without error", func() {
				So(err, ShouldBeNil)
				So(identifier, ShouldEqual, expectedIdentifier)
			})
		})
	})

	Convey("Given an invalid token", t, func() {
		mockIdentityClient := mocks.NewMockIdentityClient(ctrl)
		accessToken := "invalid-token"

		mockIdentityClient.EXPECT().
			CheckTokenIdentity(ctx, accessToken, identity.TokenTypeUser).
			Return(nil, fmt.Errorf("user token not valid"))

		mockIdentityClient.EXPECT().
			CheckTokenIdentity(ctx, accessToken, identity.TokenTypeService).
			Return(nil, fmt.Errorf("service token not valid"))

		Convey("When getTokenIdentifier is called", func() {
			identifier, err := getTokenIdentifier(ctx, accessToken, mockIdentityClient)

			Convey("Then an error is returned indicating token validation failure", func() {
				So(err.Error(), ShouldContainSubstring, "failed to validate token with identity client")
				So(identifier, ShouldEqual, "")
			})
		})
	})
}
