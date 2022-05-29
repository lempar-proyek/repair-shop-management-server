// Copyright (C) 2022 Lempar Proyek
//
// This file is part of domains.
//
// domains is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// domains is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with domains.  If not, see <http://www.gnu.org/licenses/>.

package user_test

import (
	"context"
	"os"
	"strings"
	"testing"

	"cloud.google.com/go/datastore"
	"github.com/lempar-proyek/repair-shop-management-server/domains/user"
)

var projectID = strings.TrimSpace(os.Getenv("GOOGLE_CLOUD_PROJECT"))
var wants = []map[string]string{
	{
		"name":     "Name",
		"picture":  "http://picture.com/picture",
		"username": "username1",
		"googleid": "1982732831129",
		"email":    "email1@email.com",
	},
	{
		"name":     "Name 2",
		"picture":  "http://picture.com/picture 2",
		"username": "username2",
		"googleid": "12133211",
		"email":    "email2@email.com",
	},
}

func getClient(ctx *context.Context) (*datastore.Client, error) {
	client, err := datastore.NewClient(*ctx, projectID)
	return client, err
}

func matchData(data *user.User, want map[string]string, t *testing.T) {
	if data.Name != want["name"] {
		t.Errorf("User.Name == %q, want %q", data.Name, want["name"])
	}
	if data.Picture != want["picture"] {
		t.Errorf("User.Picture == %q, want %q", data.Picture, want["picture"])
	}
	if data.Username != want["username"] {
		t.Errorf("User.Username == %q, want %q", data.Username, want["username"])
	}
	if data.GoogleId != want["googleid"] {
		t.Errorf("User.GoogleId == %q, want %q", data.GoogleId, want["googleid"])
	}
	if data.Email != want["email"] {
		t.Errorf("User.Email == %q, want %q", data.Email, want["email"])
	}
}

func CreateUser(ctx *context.Context, client *datastore.Client, want int) (*user.User, error) {
	var userData user.User
	userData.Name = wants[want]["name"]
	userData.Picture = wants[want]["picture"]
	userData.Email = wants[want]["email"]
	userData.GoogleId = wants[want]["googleid"]
	userData.Username = wants[want]["username"]
	err := userData.Create(ctx, client)
	if err != nil {
		return nil, err
	}
	return &userData, nil
}

func UpdateUser(ctx *context.Context, client *datastore.Client, want int, userData *user.User) (*user.User, error) {
	userData.Name = wants[want]["name"]
	userData.Picture = wants[want]["picture"]
	userData.Email = wants[want]["email"]
	userData.GoogleId = wants[want]["googleid"]
	userData.Username = wants[want]["username"]
	err := userData.Update(ctx, client)
	if err != nil {
		return nil, err
	}
	return userData, nil
}

func TestCRUDUser(t *testing.T) {
	// Initialization
	ctx := context.Background()

	client, err := getClient(&ctx)
	if err != nil {
		t.Errorf("Failed to create datastore connection: %v", err.Error())
		return
	}
	defer client.Close()

	// Create User
	userData, err := CreateUser(&ctx, client, 0)
	if err != nil {
		t.Errorf("Failed to create user: %v", err.Error())
		return
	}

	// Get user
	err = userData.GetById(&ctx, client, userData.Id.ID)
	if err != nil {
		t.Errorf("Failed to get created user: %v", err.Error())
		return
	}
	matchData(userData, wants[0], t)
	if userData.CreatedAt == nil {
		t.Error("User.CreatedAt == (nil), want (not nil)")
	}
	if userData.UpdatedAt == nil {
		t.Error("User.UpdatedAt == (nil), want (not nil)")
	} else if *userData.UpdatedAt != *userData.CreatedAt && userData.CreatedAt != nil {
		t.Errorf("User.UpdatedAt == %q, want %q", userData.UpdatedAt, userData.CreatedAt)
	}

	if userData.DeletedAt != nil {
		t.Error("User.DeletedAt == (not nil), want (nil")
	}

	// Update user
	userData, err = UpdateUser(&ctx, client, 1, userData)
	if err != nil {
		t.Errorf("Failed to update user: %v", err.Error())
		return
	}
	matchData(userData, wants[1], t)
	if userData.UpdatedAt == nil {
		t.Error("User.UpdatedAt == (nil), want (not nil)")
	} else if *userData.UpdatedAt == *userData.CreatedAt && userData.CreatedAt != nil {
		t.Errorf("User.UpdatedAt == %q, want > %q", userData.UpdatedAt, userData.CreatedAt)
	}

	if userData.DeletedAt != nil {
		t.Error("User.DeletedAt == (not nil), want (nil")
	}

	// Soft Delete user
	err = userData.SoftDelete(&ctx, client)
	if err != nil {
		t.Errorf("Failed to soft delete item: %v", err.Error())
		return
	}
	if userData.DeletedAt == nil {
		t.Error("User.DeletedAt == (nil), want (not nil)")
	}

	// Get Deleted User
	err = userData.GetById(&ctx, client, userData.Id.ID)
	if err != datastore.ErrNoSuchEntity {
		t.Errorf("Get result == %v, want %v", err.Error(), datastore.ErrNoSuchEntity.Error())
	}

	err = userData.GetByIdWithTrash(&ctx, client, userData.Id.ID)
	if err != nil {
		t.Errorf("Failed to get item with trashed: %v", err.Error())
	}
	matchData(userData, wants[1], t)

	// Delete user
	err = userData.Delete(&ctx, client)
	if err != nil {
		t.Errorf("Failed to delete user: %v", err.Error())
		return
	}
}
