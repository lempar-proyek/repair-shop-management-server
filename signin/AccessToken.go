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
	"strconv"
	"time"

	"cloud.google.com/go/datastore"
	"github.com/golang-jwt/jwt"
)

var accessTokenKind = "AccessToken"

type AccessToken struct {
	Id        *datastore.Key `datastore:"__key__"`
	UserId    int64
	ExpiresAt time.Time
	CreatedAt time.Time
}

func (a *AccessToken) Create(ctx *context.Context, client *datastore.Client) error {
	key := datastore.IncompleteKey(accessTokenKind, nil)
	a.CreatedAt = time.Now().UTC()

	newKey, err := client.Put(*ctx, key, a)
	if err != nil {
		return err
	}
	a.Id = newKey
	return nil
}

func (a *AccessToken) GetById(ctx *context.Context, client *datastore.Client, id string) error {
	key := datastore.NameKey(accessTokenKind, id, nil)
	err := client.Get(*ctx, key, a)
	return err
}

func (a *AccessToken) CreateFromUser(
	ctx *context.Context,
	client *datastore.Client,
	user *User,
	expires int,
) error {
	now := time.Now().UTC()
	a.UserId = user.Id.ID
	a.ExpiresAt = now.Add(time.Second * time.Duration(expires))
	err := a.Create(ctx, client)
	return err
}

func (a *AccessToken) Claims() *AccessTokenClaims {
	return &AccessTokenClaims{
		"access_token",
		jwt.StandardClaims{
			Id:        strconv.FormatInt(a.Id.ID, 10),
			Issuer:    "repair-shop-management-authorizer",
			Audience:  "repair-shop-management-server",
			ExpiresAt: a.ExpiresAt.Unix(),
			IssuedAt:  a.CreatedAt.Unix(),
			Subject:   strconv.FormatInt(a.UserId, 10),
		},
	}
}

type AccessTokenClaims struct {
	Type string `json:"typ"`
	jwt.StandardClaims
}

func (ac *AccessTokenClaims) SignRsa256(key *rsa.PrivateKey) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, ac)
	signed, err := token.SignedString(key)
	return signed, err
}
