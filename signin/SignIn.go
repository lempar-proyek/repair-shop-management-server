// Copyright (C) 2022 Lempar Proyek
//
// This file is part of signin.
//
// signin is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// signin is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with signin.  If not, see <http://www.gnu.org/licenses/>.

package signin

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	"cloud.google.com/go/datastore"
	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	"github.com/golang-jwt/jwt"
	"google.golang.org/api/idtoken"
	secretmanagerpb "google.golang.org/genproto/googleapis/cloud/secretmanager/v1"
)

var projectID string
var datastoreClient *datastore.Client

type TokenResponse struct {
	AccessToken  string   `json:"access_token"`
	Type         string   `json:"type"`
	ExpiresIn    uint     `json:"expires_in"`
	RefreshToken string   `json:"refresh_token"`
	UserData     *UserDto `json:"user"`
}

type CredentialsDto struct {
	Token    string `json:"token"`
	Provider string `json:"provider"`
}

func sendErrorMsg(msg string, e string, w *http.ResponseWriter) {
	sendMsg(
		fmt.Sprintf("{\"code\": 500, \"message\": \"%v\", \"errors\": \"%v\"}", msg, e),
		http.StatusInternalServerError,
		w,
	)
}

func sendUnauthorizedMsg(msg string, e string, w *http.ResponseWriter) {
	sendMsg(
		fmt.Sprintf("{\"code\": 401, \"message\": \"%v\", \"errors\": \"%v\"}", msg, e),
		http.StatusUnauthorized,
		w,
	)
}

func sendUnprocessableEntityMsg(e string, w *http.ResponseWriter) {
	sendMsg(
		fmt.Sprintf("{\"code\": 422, \"message\": \"%v\", \"errors\": \"%v\"}", "Failed to process request body", e),
		http.StatusUnprocessableEntity,
		w,
	)
}

func sendMethodNotAllowedMsg(e string, w *http.ResponseWriter) {
	sendMsg(
		fmt.Sprintf("{\"code\": 405, \"message\": \"%v\", \"errors\": \"%v\"}", "HTTP Method not allowed", e),
		http.StatusMethodNotAllowed,
		w,
	)
}

func sendMsg(payload string, status int, w *http.ResponseWriter) {
	(*w).Header().Set("Content-Type", "application/json; charset=utf-8")
	(*w).WriteHeader(status)
	(*w).Write([]byte(payload))
}

func ensurePostMethod(r *http.Request) error {
	if r.Method != http.MethodPost {
		return fmt.Errorf("method %s is not allowed", r.Method)
	}
	return nil
}

func parseRequestBody(r *http.Request) (*CredentialsDto, error) {
	var credentialDto CredentialsDto
	if err := json.NewDecoder(r.Body).Decode(&credentialDto); err != nil {
		return nil, err
	}
	return &credentialDto, nil
}

func validateGoogleToken(ctx *context.Context, token string) (*idtoken.Payload, error) {
	serverID := strings.TrimSpace(os.Getenv("SERVER_ID"))
	googleClientID := strings.TrimSpace(os.Getenv("GOOGLE_CLIENT_ID"))

	payload, err := idtoken.Validate(*ctx, token, serverID)
	if err != nil {
		return nil, err
	}

	claims := payload.Claims
	azp := fmt.Sprint(claims["azp"])
	if azp != googleClientID {
		return nil, fmt.Errorf("client applications is not authorized by our server")
	}

	return payload, nil
}

func getPrivateJwtSecret(ctx *context.Context) ([]byte, error) {
	secretClient, err := secretmanager.NewClient(*ctx)
	if err != nil {
		return nil, err
	}
	defer secretClient.Close()

	privKeyReq := &secretmanagerpb.AccessSecretVersionRequest{
		Name: fmt.Sprintf("projects/%s/secrets/%s/versions/latest", projectID, "jwt-priv-key"),
	}
	privKeyResp, err := secretClient.AccessSecretVersion(*ctx, privKeyReq)
	if err != nil {
		return nil, err
	}

	return privKeyResp.Payload.Data, nil
}

func generateToken(ctx *context.Context, user *User, r *http.Request) (*TokenResponse, error) {
	refreshExpires := 15768000 // 6 months
	// now := time.Now().UTC()

	privateKeyPem, err := getPrivateJwtSecret(ctx)
	if err != nil {
		return nil, err
	}
	privateKey, err := jwt.ParseRSAPrivateKeyFromPEM(privateKeyPem)
	if err != nil {
		return nil, err
	}

	var refreshToken RefreshToken
	refreshTokenClaims, err := refreshToken.CreateFromRequest(ctx, user, refreshExpires, r)
	if err != nil {
		return nil, err
	}
	signedRefreshToken, err := refreshTokenClaims.SignRsa512(privateKey)
	if err != nil {
		return nil, err
	}

	var tokenResponse TokenResponse
	tokenResponse.AccessToken = ""
	tokenResponse.ExpiresIn = 0
	tokenResponse.Type = "Bearer"
	tokenResponse.UserData = user.Dto()
	tokenResponse.RefreshToken = signedRefreshToken
	return &tokenResponse, nil
}

func SignIn(w http.ResponseWriter, r *http.Request) {
	projectID = strings.TrimSpace(os.Getenv("GOOGLE_CLOUD_PROJECT"))

	err := ensurePostMethod(r)
	if err != nil {
		sendMethodNotAllowedMsg(err.Error(), &w)
		return
	}

	credentialDto, err := parseRequestBody(r)
	if err != nil {
		sendUnprocessableEntityMsg(err.Error(), &w)
		return
	}

	provider := strings.ToLower(credentialDto.Provider)
	switch provider {
	case "google":
		ctx := context.Background()
		tokenPayload, err := validateGoogleToken(&ctx, credentialDto.Token)
		if err != nil {
			sendUnauthorizedMsg("Token rejected.", err.Error(), &w)
			return
		}

		claims := tokenPayload.Claims

		// Connect to datastore
		datastoreClient, err = datastore.NewClient(ctx, projectID)
		if err != nil {
			sendErrorMsg("Failed to connect to database.", err.Error(), &w)
		}
		defer datastoreClient.Close()

		var user User
		if err := user.GetByGID(&ctx, fmt.Sprint(claims["sub"])); err == datastore.ErrNoSuchEntity {
			err = user.CreateFromGoogleClaims(&ctx, claims)
			if err != nil {
				sendErrorMsg("Error creating user", err.Error(), &w)
				return
			}
		} else if err != nil {
			sendErrorMsg("Failed to fetch user info.", err.Error(), &w)
			return
		}

		tokenResponse, err := generateToken(&ctx, &user, r)
		if err != nil {
			sendErrorMsg("Failed to generate token.", err.Error(), &w)
			return
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		json.NewEncoder(w).Encode(tokenResponse)

	default:
		sendErrorMsg("The provider is not registered in our system.", "", &w)
		return
	}

}
