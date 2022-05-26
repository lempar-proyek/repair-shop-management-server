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
	"time"

	"cloud.google.com/go/datastore"
)

type RefreshToken struct {
	Id         *datastore.Key `datastore:"__key__"`
	UserId     string
	ExpiresAt  time.Time
	Device     string
	Os         string
	ClientName string
}

var kind = "RefreshToken"

func (r *RefreshToken) Create(ctx *context.Context, client *datastore.Client) error {
	key := datastore.IncompleteKey(kind, nil)
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
