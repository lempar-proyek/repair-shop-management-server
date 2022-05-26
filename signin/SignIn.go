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
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"cloud.google.com/go/datastore"
	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	"github.com/golang-jwt/jwt"
	"github.com/google/uuid"
	ua "github.com/mileusna/useragent"
	"google.golang.org/api/idtoken"
	"google.golang.org/api/iterator"
	secretmanagerpb "google.golang.org/genproto/googleapis/cloud/secretmanager/v1"
)

var projectID string
var datastoreClient *datastore.Client

type User struct {
	Id       *datastore.Key `datastore:"__key__" json:"id"`
	Name     string         `json:"name"`
	Username string         `json:"username"`
	Email    string         `json:"email"`
	GoogleId string         `json:"-"`
	Picture  string         `json:"picture"`
}

type UserDto struct {
	Name     string `json:"name"`
	Username string `json:"username"`
	Picture  string `json:"picture"`
}

type TokenResponse struct {
	AccessToken  string   `json:"access_token"`
	Type         string   `json:"type"`
	ExpiresIn    uint     `json:"expires_in"`
	RefreshToken string   `json:"refresh_token"`
	UserData     *UserDto `json:"user"`
}

var errNotFound = errors.New("data not found in database")
var errDuplicateUsername = errors.New("username has been picked by another account")
var errDuplicateEmail = errors.New("email has been picked by another account")
var errDuplicateGoogleId = errors.New("google account has been registered by another account")

func (u *User) GetByGID(ctx *context.Context, gid string) error {
	query := datastore.NewQuery("User").Filter(
		"GoogleId =",
		gid,
	).Limit(1)
	it := datastoreClient.Run(*ctx, query)

	if _, err := it.Next(u); err == iterator.Done {
		return errNotFound
	} else if err != nil {
		return err
	}
	return nil
}

func (u *User) Create(ctx *context.Context) error {
	tx, err := datastoreClient.NewTransaction(*ctx, datastore.ReadOnly)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var users []User

	usernameQuery := datastore.NewQuery("User").Filter("Username =", u.Username).Transaction(tx)
	_, err = datastoreClient.GetAll(*ctx, usernameQuery, &users)
	if err != nil {
		return err
	}
	if len(users) > 0 {
		return errDuplicateUsername
	}

	emailQuery := datastore.NewQuery("User").Filter("Email =", u.Email).Transaction(tx)
	_, err = datastoreClient.GetAll(*ctx, emailQuery, &users)
	if err != nil {
		return err
	}
	if len(users) > 0 {
		return errDuplicateEmail
	}

	googleIdQuery := datastore.NewQuery("User").Filter("GoogleId =", u.GoogleId).Transaction(tx)
	_, err = datastoreClient.GetAll(*ctx, googleIdQuery, &users)
	if err != nil {
		return err
	}
	if len(users) > 0 {
		return errDuplicateGoogleId
	}

	userKey := datastore.IncompleteKey("User", nil)
	key, err := datastoreClient.Put(*ctx, userKey, u)
	if err != nil {
		return err
	}
	u.Id = key

	return nil
}

func (u *User) Dto() *UserDto {
	return &UserDto{
		Name:     u.Name,
		Username: u.Username,
		Picture:  u.Picture,
	}
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

func createNewUserFromGoogleClaims(ctx *context.Context, u *User, claims map[string]interface{}) error {
	u.Email = fmt.Sprint(claims["email"])
	u.GoogleId = fmt.Sprint(claims["sub"])
	u.Name = fmt.Sprint(claims["name"])
	u.Username = strings.Replace(uuid.New().String(), "-", "", -1)
	if err := u.Create(ctx); err != nil {
		return err
	}
	return nil
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

func generateToken(ctx *context.Context, user *User, req *http.Request) (TokenResponse, error) {
	refreshExpires := 15768000 // 6 months
	now := time.Now().UTC()

	var tokenResponse TokenResponse
	privateKeyPem, err := getPrivateJwtSecret(ctx)
	if err != nil {
		return tokenResponse, err
	}
	privateKey, err := jwt.ParseRSAPrivateKeyFromPEM(privateKeyPem)
	if err != nil {
		return tokenResponse, err
	}

	userAgent := ua.Parse(req.Header.Get("User-Agent"))

	var refreshToken RefreshToken
	refreshToken.ClientName = userAgent.Name
	refreshToken.Device = userAgent.Device
	refreshToken.Os = userAgent.OS

	refreshToken.ExpiresAt = time.Now().UTC().Add(time.Second * time.Duration(refreshExpires))
	refreshToken.UserId = user.Id.Name
	err = refreshToken.Create(ctx, datastoreClient)
	if err != nil {
		return tokenResponse, err
	}

	type RefreshClaims struct {
		Type string `json:"typ,omitempty"`
		jwt.StandardClaims
	}
	refreshClaims := &RefreshClaims{
		"refresh_token",
		jwt.StandardClaims{
			Id:       refreshToken.Id.Name,
			Subject:  refreshToken.UserId,
			Issuer:   "repair-shop-management-authorizer",
			Audience: "repair-shop-management-server",
			IssuedAt: now.Unix(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS512, refreshClaims)
	signedRefreshToken, err := token.SignedString(privateKey)
	if err != nil {
		return tokenResponse, err
	}

	tokenResponse.AccessToken = ""
	tokenResponse.ExpiresIn = 0
	tokenResponse.Type = "Bearer"
	tokenResponse.UserData = user.Dto()
	tokenResponse.RefreshToken = signedRefreshToken
	return tokenResponse, nil
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
		if err := user.GetByGID(&ctx, fmt.Sprint(claims["sub"])); err == errNotFound {
			err = createNewUserFromGoogleClaims(&ctx, &user, claims)
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
