// Copyright 2023 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package interfaces

import (
	"context"
	"fmt"

	"cloud.google.com/go/bigquery"
	"google.golang.org/api/option"
)

func QueryForResults(variant string, milestone string) *bigquery.RowIterator {
	ctx := context.Background()
	c, err := bigquery.NewClient(ctx, "chromeos-bot",
		option.WithCredentialsFile("/creds/service_accounts/service-account-chromeos.json"))
	if err != nil {
		fmt.Printf("Query issues: %s\n", err)
	}
	fmt.Println("Good Query.")
	cmd := fmt.Sprintf("SELECT * FROM chromeos-test-platform-data.analytics.FlakeCache WHERE total_runs > 0 AND board = \"%s\"  AND (REGEXP_CONTAINS(build, \"%s\"))", variant, milestone)
	fmt.Println((cmd))
	bqQ := c.Query(cmd)
	// Execute the query.
	it, err := bqQ.Read(ctx)
	if err != nil {
		fmt.Printf("INFORMATIONAL: query error: %s", err)
	}

	return it
}
