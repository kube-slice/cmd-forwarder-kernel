// Copyright (c) 2026 Avesha, Inc. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at:
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package recvfd

import (
	"context"
	"testing"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/pkg/errors"
)

type failingServer struct {
	networkservice.NetworkServiceServer
}

func (failingServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	return nil, errors.New("all forwarders have failed")
}

func (failingServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	return &empty.Empty{}, nil
}

// Close is the only thing that removes a connection's file map, and a request
// that failed will never be closed. Every failed attempt therefore used to
// leave an entry behind for good, so the forwarder's footprint tracked how
// often requests failed rather than how many clients it served. On a node whose
// attaches were failing -- four thousand in three minutes was measured -- it
// grew until the kernel killed it, taking out the one component every pod needs
// in order to get an interface at all.
func TestFailedRequestLeavesNoFileMapBehind(t *testing.T) {
	server := &recvFDServer{}

	const id = "pg-dcdr-dc-a-0-0-0-RDHlAa70"
	request := &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			Id:        id,
			Mechanism: &networkservice.Mechanism{Parameters: map[string]string{}},
		},
	}

	ctx := next.NewNetworkServiceServer(server, failingServer{})
	if _, err := ctx.Request(context.Background(), request); err == nil {
		t.Fatal("precondition: the request must fail")
	}

	if _, loaded := server.fileMaps.Load(id); loaded {
		t.Error("a failed request left its file map behind; this is the leak that OOMKills the forwarder")
	}
}
