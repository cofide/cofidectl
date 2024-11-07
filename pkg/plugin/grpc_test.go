// Copyright 2024 Cofide Limited.
// SPDX-License-Identifier: Apache-2.0

package plugin

import (
	"testing"
)

func TestDataSourcePluginClientGRPC_ImplementsDataSource(t *testing.T) {
	client := DataSourcePluginClientGRPC{}
	var _ DataSource = &client
}
