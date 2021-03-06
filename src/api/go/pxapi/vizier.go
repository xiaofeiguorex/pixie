/*
 * Copyright 2018- The Pixie Authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package pxapi

import (
	"context"

	cloudapipb "go.withpixie.dev/pixie/src/api/public/cloudapipb"
	vizierapipb "go.withpixie.dev/pixie/src/api/public/vizierapipb"
)

// VizierStatus stores the enumeration of all vizier statuses.
type VizierStatus string

// Vizier Statuses.
const (
	VizierStatusUnknown      VizierStatus = "Unknown"
	VizierStatusHealthy                   = "Healthy"
	VizierStatusUnhealthy                 = "Unhealthy"
	VizierStatusDisconnected              = "Disconnected"
)

// VizierInfo has information of a single Vizier.
type VizierInfo struct {
	// Name of the vizier.
	Name string
	// ID of the Vizier (uuid as a string).
	ID string
	// Status of the Vizier.
	Status VizierStatus
	// Version of the installed vizier.
	Version string
	// DirectAccess says the cluster has direct access mode enabled. This means the data transfer will bypass the cloud.
	DirectAccess bool
}

func clusterStatusToVizierStatus(status cloudapipb.ClusterStatus) VizierStatus {
	switch status {
	case cloudapipb.CS_HEALTHY:
		return VizierStatusHealthy
	case cloudapipb.CS_UNHEALTHY:
		return VizierStatusUnhealthy
	case cloudapipb.CS_DISCONNECTED:
		return VizierStatusDisconnected
	default:
		return VizierStatusUnknown
	}
}

// ListViziers gets a list of Viziers registered with Pixie.
func (c *Client) ListViziers(ctx context.Context) ([]VizierInfo, error) {
	req := &cloudapipb.GetClusterRequest{}
	res, err := c.cmClient.GetCluster(c.cloudCtxWithMD(ctx), req)
	if err != nil {
		return nil, err
	}

	viziers := make([]VizierInfo, 0)
	for _, v := range res.Clusters {
		viziers = append(viziers, VizierInfo{
			Name:         v.ClusterName,
			ID:           string(v.ID.Data),
			Version:      v.VizierVersion,
			Status:       clusterStatusToVizierStatus(v.Status),
			DirectAccess: !v.Config.PassthroughEnabled,
		})
	}

	return viziers, nil
}

// VizierClient is the client for a single vizier.
type VizierClient struct {
	cloud        *Client
	directAccess bool
	accessToken  string
	vizierID     string

	vzClient vizierapipb.VizierServiceClient
}

// ExecuteScript runs the script on vizier.
func (v *VizierClient) ExecuteScript(ctx context.Context, pxl string, mux TableMuxer) (*ScriptResults, error) {
	req := &vizierapipb.ExecuteScriptRequest{
		ClusterID: v.vizierID,
		QueryStr:  pxl,
	}
	// TODO(zasgar): Fix the token to use the right version dependent or directaccess or cloud.
	ctx, cancel := context.WithCancel(ctx)
	res, err := v.vzClient.ExecuteScript(v.cloud.cloudCtxWithMD(ctx), req)
	if err != nil {
		cancel()
		return nil, err
	}

	sr := newScriptResults()
	sr.c = res
	sr.cancel = cancel
	sr.tm = mux

	return sr, nil
}
