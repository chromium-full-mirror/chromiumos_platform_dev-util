// Copyright 2023 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.
package cli

import (
	"flag"
	"fmt"
	"strings"

	"github.com/golang/glog"
	"go.chromium.org/chromiumos/config/go/test/plan"
	"go.chromium.org/luci/common/data/strpair"
	"go.chromium.org/luci/common/data/text"
	luciflag "go.chromium.org/luci/common/flag"
	"google.golang.org/protobuf/encoding/protojson"
)

// TemplateParametersFlag encapsulates logic to parse lists of
// TemplateParameters from the command line.
type TemplateParametersFlag struct {
	pathToRawTemplateParametersList strpair.Map
}

// Register creates a templateparameter flag in fs.
func (t *TemplateParametersFlag) Register(fs *flag.FlagSet) {
	t.pathToRawTemplateParametersList = strpair.Map{}
	fs.Var(
		luciflag.StringPairs(t.pathToRawTemplateParametersList),
		"templateparameter",
		text.Doc(`
			Colon-separated map from plan to TemplateParameter jsonproto. Can be
			specified multiple times, either for the same plan or different
			plans.

			The keys must be exactly the same as an argument to -plan.

			Example:
				-plan custom.star \
				-templateparameter 'custom.star:{"tag_criteria": {"tags": ["group:mycustom1"]}}, "suite_name": "mycustom1"' \
				-templateparameter 'custom.star:{"tag_criteria": {"tags": ["group:mycustom2"]}}, "suite_name": "mycustom2"'
		`),
	)
}

// Parse parses the raw strings given in the command line into
// TemplateParameters.
func (t *TemplateParametersFlag) Parse() (map[string][]*plan.SourceTestPlan_TestPlanStarlarkFile_TemplateParameters, error) {
	pathToTemplateParametersList := make(map[string][]*plan.SourceTestPlan_TestPlanStarlarkFile_TemplateParameters)
	unmarshalOpts := protojson.UnmarshalOptions{DiscardUnknown: true}

	for path, rawTemplateParametersList := range t.pathToRawTemplateParametersList {
		pathToTemplateParametersList[path] = make(
			[]*plan.SourceTestPlan_TestPlanStarlarkFile_TemplateParameters, 0, len(rawTemplateParametersList),
		)
		for _, rawTemplateParameters := range rawTemplateParametersList {
			templateParameters := &plan.SourceTestPlan_TestPlanStarlarkFile_TemplateParameters{}
			if err := unmarshalOpts.Unmarshal([]byte(rawTemplateParameters), templateParameters); err != nil {
				glog.Warningf(
					"failed to parse TemplateParameters json, trimming quotes and trying again. Error: %q. TemplateParameters: %q",
					err, rawTemplateParameters,
				)
				rawTemplateParametersTrimmed := strings.Trim(rawTemplateParameters, `"'`)

				if err := unmarshalOpts.Unmarshal([]byte(rawTemplateParametersTrimmed), templateParameters); err != nil {
					return nil, fmt.Errorf("failed to parse TemplateParameters %q: %w", rawTemplateParametersTrimmed, err)
				}
			}

			pathToTemplateParametersList[path] = append(pathToTemplateParametersList[path], templateParameters)
		}
	}

	return pathToTemplateParametersList, nil
}
