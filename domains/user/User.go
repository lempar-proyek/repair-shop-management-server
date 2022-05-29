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

package user

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/datastore"
	"github.com/google/uuid"
	"google.golang.org/api/iterator"
)

type User struct {
	Id        *datastore.Key `datastore:"__key__" json:"id"`
	Name      string         `json:"name"`
	Username  string         `json:"username"`
	Email     string         `json:"email"`
	GoogleId  string         `json:"-"`
	Picture   string         `json:"picture"`
	Blocked   bool           `json:"-" default:"false"`
	CreatedAt *time.Time     `json:"created_at"`
	UpdatedAt *time.Time     `json:"updated_at"`
	DeletedAt *time.Time     `json:"-"`
}

var kind = "User"

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

func (u *User) GetByIdWithTrash(ctx *context.Context, client *datastore.Client, id int64) error {
	key := datastore.IDKey(kind, id, nil)
	err := client.Get(*ctx, key, u)
	return err
}

func (u *User) GetById(ctx *context.Context, client *datastore.Client, id int64) error {
	key := datastore.IDKey(kind, id, nil)
	err := client.Get(*ctx, key, u)
	if err != nil {
		return err
	}
	if u.DeletedAt != nil {
		return datastore.ErrNoSuchEntity
	}
	return nil
}

func (u *User) Create(ctx *context.Context, client *datastore.Client) error {
	var ckUser User

	usernameQuery := datastore.NewQuery(kind).Filter("Username =", u.Username)
	iter := client.Run(*ctx, usernameQuery)
	if _, err := iter.Next(&ckUser); err != iterator.Done {
		return ErrDuplicateUsername
	}

	emailQuery := datastore.NewQuery(kind).Filter("Email =", u.Email)
	iter = client.Run(*ctx, emailQuery)
	if _, err := iter.Next(&ckUser); err != iterator.Done {
		return ErrDuplicateEmail
	}

	googleIdQuery := datastore.NewQuery(kind).Filter("GoogleId =", u.GoogleId)
	iter = client.Run(*ctx, googleIdQuery)
	if _, err := iter.Next(&ckUser); err != iterator.Done {
		return ErrDuplicateGoogleId
	}

	now := time.Now().UTC()
	u.CreatedAt = &now
	u.UpdatedAt = &now

	userKey := datastore.IncompleteKey(kind, nil)
	key, err := client.Put(*ctx, userKey, u)
	if err != nil {
		return err
	}
	u.Id = key

	return nil
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

func (u *User) Update(ctx *context.Context, client *datastore.Client) error {
	now := time.Now().UTC()
	u.UpdatedAt = &now
	_, err := client.Put(*ctx, u.Id, u)
	return err
}

func (u *User) SoftDelete(ctx *context.Context, client *datastore.Client) error {
	now := time.Now().UTC()
	u.DeletedAt = &now
	_, err := client.Put(*ctx, u.Id, u)
	return err
}

func (u *User) Delete(ctx *context.Context, client *datastore.Client) error {
	err := client.Delete(*ctx, u.Id)
	return err
}

func (u *User) Dto() *UserDto {
	return &UserDto{
		Name:     u.Name,
		Username: u.Username,
		Picture:  u.Picture,
	}
}

type UserDto struct {
	Name     string `json:"name"`
	Username string `json:"username"`
	Picture  string `json:"picture"`
}

var ErrDuplicateUsername = errors.New("username has been picked by another account")
var ErrDuplicateEmail = errors.New("email has been picked by another account")
var ErrDuplicateGoogleId = errors.New("google account has been registered by another account")
