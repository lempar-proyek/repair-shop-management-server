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

package domains

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"cloud.google.com/go/datastore"
	"github.com/google/uuid"
	"google.golang.org/api/iterator"
)

type User struct {
	Id       *datastore.Key `datastore:"__key__" json:"id"`
	Name     string         `json:"name"`
	Username string         `json:"username"`
	Email    string         `json:"email"`
	GoogleId string         `json:"-"`
	Picture  string         `json:"picture"`
}

var userKind = "User"

func (u *User) GetByGID(ctx *context.Context, gid string, client *datastore.Client) error {
	query := datastore.NewQuery("User").Filter(
		"GoogleId =",
		gid,
	).Limit(1)
	it := client.Run(*ctx, query)

	if _, err := it.Next(u); err == iterator.Done {
		return datastore.ErrNoSuchEntity
	} else if err != nil {
		return err
	}
	return nil
}

func (u *User) GetById(ctx *context.Context, client *datastore.Client, id string) error {
	key := datastore.NameKey(userKind, id, nil)
	err := client.Get(*ctx, key, u)
	return err
}

func (u *User) Create(ctx *context.Context, client *datastore.Client) error {
	tx, err := client.NewTransaction(*ctx, datastore.ReadOnly)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var users []User

	usernameQuery := datastore.NewQuery("User").Filter("Username =", u.Username).Transaction(tx)
	_, err = client.GetAll(*ctx, usernameQuery, &users)
	if err != nil {
		return err
	}
	if len(users) > 0 {
		return ErrDuplicateUsername
	}

	emailQuery := datastore.NewQuery("User").Filter("Email =", u.Email).Transaction(tx)
	_, err = client.GetAll(*ctx, emailQuery, &users)
	if err != nil {
		return err
	}
	if len(users) > 0 {
		return ErrDuplicateEmail
	}

	googleIdQuery := datastore.NewQuery("User").Filter("GoogleId =", u.GoogleId).Transaction(tx)
	_, err = client.GetAll(*ctx, googleIdQuery, &users)
	if err != nil {
		return err
	}
	if len(users) > 0 {
		return ErrDuplicateGoogleId
	}

	userKey := datastore.IncompleteKey("User", nil)
	key, err := client.Put(*ctx, userKey, u)
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

func (u *User) CreateFromGoogleClaims(ctx *context.Context, client *datastore.Client, claims map[string]interface{}) error {
	u.Email = fmt.Sprint(claims["email"])
	u.GoogleId = fmt.Sprint(claims["sub"])
	u.Name = fmt.Sprint(claims["name"])
	u.Username = strings.Replace(uuid.New().String(), "-", "", -1)
	if err := u.Create(ctx, client); err != nil {
		return err
	}
	return nil
}

type UserDto struct {
	Name     string `json:"name"`
	Username string `json:"username"`
	Picture  string `json:"picture"`
}

var ErrDuplicateUsername = errors.New("username has been picked by another account")
var ErrDuplicateEmail = errors.New("email has been picked by another account")
var ErrDuplicateGoogleId = errors.New("google account has been registered by another account")
