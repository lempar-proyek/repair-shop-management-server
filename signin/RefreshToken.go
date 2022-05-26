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
	"crypto/rsa"
	"net/http"
	"strconv"
	"time"

	"cloud.google.com/go/datastore"
	"github.com/golang-jwt/jwt"
	ua "github.com/mileusna/useragent"
)

type RefreshToken struct {
	Id         *datastore.Key `datastore:"__key__"`
	UserId     int64
	ExpiresAt  time.Time
	Device     string
	Os         string
	ClientName string
	CreatedAt  time.Time
}

var kind = "RefreshToken"

func (r *RefreshToken) Create(ctx *context.Context, client *datastore.Client) error {
	key := datastore.IncompleteKey(kind, nil)
	r.CreatedAt = time.Now().UTC()

	key, err := client.Put(*ctx, key, r)
	if err != nil {
		return err
	}
	r.Id = key
	return nil
}

func (r *RefreshToken) GetById(ctx *context.Context, client *datastore.Client, id string) error {
	key := datastore.NameKey(kind, id, nil)
	err := client.Get(*ctx, key, r)
	return err
}

func (r *RefreshToken) Claims() *RefreshTokenClaims {
	return &RefreshTokenClaims{
		"refresh_token",
		jwt.StandardClaims{
			Id:        strconv.FormatInt(r.Id.ID, 10),
			Subject:   strconv.FormatInt(r.UserId, 10),
			Issuer:    "repair-shop-management-authorizer",
			Audience:  "repair-shop-management-server",
			IssuedAt:  r.CreatedAt.Unix(),
			ExpiresAt: r.ExpiresAt.Unix(),
		},
	}
}

func (r *RefreshToken) CreateFromRequest(ctx *context.Context, user *User, expires int, req *http.Request) (*RefreshTokenClaims, error) {
	userAgent := ua.Parse(req.Header.Get("User-Agent"))
	r.ClientName = userAgent.Name
	r.Device = userAgent.Device
	r.Os = userAgent.OS

	r.ExpiresAt = time.Now().UTC().Add(time.Second * time.Duration(expires))
	r.UserId = user.Id.ID
	err := r.Create(ctx, datastoreClient)
	if err != nil {
		return nil, err
	}

	return r.Claims(), nil
}

type RefreshTokenClaims struct {
	Type string `json:"typ,omitempty"`
	jwt.StandardClaims
}

func (rc *RefreshTokenClaims) SignRsa512(key *rsa.PrivateKey) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodRS512, rc)
	signedRc, err := token.SignedString(key)
	if err != nil {
		return "", err
	}
	return signedRc, nil
}
