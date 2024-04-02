package compatibility

import (
	"errors"

	testpb "go.chromium.org/chromiumos/config/go/test/api"
	"go.chromium.org/chromiumos/infra/proto/go/testplans"
)

// chooseCompanions chooses a program for each companion according to the
// primary program chosen, indicated by the index number. When a companion
// has the same number of programs to choose from as the primary, choose the
// same index. Otherwise, choose `index % len(companion_programs)` -- this is
// just to randomly choose a program before we official support this use case
// (initially the starlark config will enforce only equal size of programs).
func chooseCompanions(primaryDutProgramIndex int, rule *testpb.CoverageRule, programAttr *testpb.DutAttribute) ([]*testplans.TestCompanion, error) {
	if len(rule.GetDutTargets()) <= 1 {
		return nil, nil
	}
	rv := make([]*testplans.TestCompanion, 0)
	for _, dutTarget := range rule.GetDutTargets()[1:] {
		programs, err := getAttrFromCriteria(dutTarget.GetCriteria(), programAttr)
		if err != nil {
			return nil, err
		}
		if len(programs) == 0 {
			return nil, errors.New("DutCriteria must contain at least one \"attr-program\" attribute")
		}
		chosenProgram := programs[primaryDutProgramIndex%len(programs)]
		rv = append(rv, &testplans.TestCompanion{
			Board:  chosenProgram,
			Config: dutTarget.GetProvisionConfig().GetCompanion(),
		})
	}
	return rv, nil
}
