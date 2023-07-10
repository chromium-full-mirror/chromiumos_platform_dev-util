// Copyright 2023 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.
package cli_test

import (
	"chromiumos/test/plan/internal/cli"
	"flag"
	"testing"

	"github.com/google/go-cmp/cmp"
	"go.chromium.org/chromiumos/config/go/test/plan"
	"google.golang.org/protobuf/testing/protocmp"
)

func TestTemplateParametersFlag(t *testing.T) {
	tests := []struct {
		name  string
		input []string
		want  map[string][]*plan.SourceTestPlan_TestPlanStarlarkFile_TemplateParameters
	}{
		{
			name:  "single parameter",
			input: []string{"-templateparameter", `test.star:{"suiteName": "mysuiteA"}`},
			want: map[string][]*plan.SourceTestPlan_TestPlanStarlarkFile_TemplateParameters{
				"test.star": {{SuiteName: "mysuiteA"}},
			},
		},
		{
			name: "multiple parameters",
			input: []string{
				"-templateparameter", `test.star:{"suiteName": "mysuiteA"}`,
				"-templateparameter", `test.star:{"suiteName": "mysuiteB"}`,
			},
			want: map[string][]*plan.SourceTestPlan_TestPlanStarlarkFile_TemplateParameters{
				"test.star": {{SuiteName: "mysuiteA"}, {SuiteName: "mysuiteB"}},
			},
		},
		{
			name:  "single quotes",
			input: []string{"-templateparameter", `test.star:'{"suiteName": "mysuiteA"}'`},
			want: map[string][]*plan.SourceTestPlan_TestPlanStarlarkFile_TemplateParameters{
				"test.star": {{SuiteName: "mysuiteA"}},
			},
		},
		{
			name:  "double quotes",
			input: []string{"-templateparameter", `test.star:"{"suiteName": "mysuiteA"}"`},
			want: map[string][]*plan.SourceTestPlan_TestPlanStarlarkFile_TemplateParameters{
				"test.star": {{SuiteName: "mysuiteA"}},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			templateParametersFlag := cli.TemplateParametersFlag{}
			fs := &flag.FlagSet{}
			templateParametersFlag.Register(fs)
			if err := fs.Parse(tc.input); err != nil {
				t.Fatalf("fs.Parse() failed: %q", err)
			}

			got, err := templateParametersFlag.Parse()
			if err != nil {
				t.Fatalf("templateParametersFlag.Parse() failed: %q", err)
			}

			if diff := cmp.Diff(tc.want, got, protocmp.Transform()); diff != "" {
				t.Errorf("templateParametersFlag.Parse()) returned unexpected diff (-want +got):\n%s", diff)
			}
		})
	}
}
