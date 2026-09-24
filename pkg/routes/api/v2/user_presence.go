// Vikunja is a to-do list application to facilitate your life.
// Copyright 2018-present Vikunja and contributors. All rights reserved.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package apiv2

import (
	"context"
	"net/http"

	"code.vikunja.io/api/pkg/config"
	"code.vikunja.io/api/pkg/db"
	"code.vikunja.io/api/pkg/user"
	"code.vikunja.io/api/pkg/websocket"

	"github.com/danielgtaylor/huma/v2"
)

type userPresenceBody struct {
	UserIDs []int64 `json:"user_ids" readOnly:"true" doc:"Ids of the online users who share at least one project or one team with you, ascending. Never includes your own id. A user stays online for 30 seconds after their last websocket connection closed."`
}

// RegisterUserPresenceRoutes mounts the presence snapshot only when
// service.ms.presence is on; otherwise the route does not exist (404), like
// every other feature switched off in the config.
func RegisterUserPresenceRoutes(api huma.API) {
	if !config.ServiceMSPresence.GetBool() {
		return
	}

	Register(api, huma.Operation{
		OperationID: "user-presence",
		Summary:     "List the online users you work with",
		Description: "Returns who is online right now among the users who share at least one project (as owner, through a direct share or through a team share, inherited from a parent project included) or one team with you. Meant for the initial load: after it, the websocket event user.presence carries every change. Not available to link shares (403) or API tokens.",
		Method:      http.MethodGet,
		Path:        "/user/presence",
		Tags:        []string{"user"},
	}, userPresence)
}

func init() { AddRouteRegistrar(RegisterUserPresenceRoutes) }

func userPresence(ctx context.Context, _ *struct{}) (*singleBody[userPresenceBody], error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	doer, err := user.GetFromAuth(a)
	if err != nil {
		return nil, translateDomainError(err)
	}

	s := db.NewSession()
	defer s.Close()

	ids, err := websocket.GetHub().OnlineUserIDsVisibleTo(s, doer.ID)
	if err != nil {
		return nil, translateDomainError(err)
	}

	return &singleBody[userPresenceBody]{Body: &userPresenceBody{UserIDs: ids}}, nil
}
