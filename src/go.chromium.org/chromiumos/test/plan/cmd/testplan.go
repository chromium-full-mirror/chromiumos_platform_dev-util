// Copyright 2021 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// The testplan tool evaluates Starlark files to generate ChromeOS
// chromiumos.test.api.CoverageRule protos.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strings"

	"github.com/golang/glog"
	"github.com/maruel/subcommands"
	buildpb "go.chromium.org/chromiumos/config/go/build/api"
	"go.chromium.org/chromiumos/config/go/payload"
	testpb "go.chromium.org/chromiumos/config/go/test/api"
	"go.chromium.org/chromiumos/infra/proto/go/chromiumos"
	"go.chromium.org/chromiumos/infra/proto/go/testplans"
	bbpb "go.chromium.org/luci/buildbucket/proto"
	luciflag "go.chromium.org/luci/common/flag"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	testplan "go.chromium.org/chromiumos/test/plan/internal"
	"go.chromium.org/chromiumos/test/plan/internal/cli"
	"go.chromium.org/chromiumos/test/plan/internal/compatibility"
	"go.chromium.org/chromiumos/test/plan/internal/protoio"
)

// Version is set to the CROS_GO_VERSION eclass variable at build time. See
// cros-go.eclass for details.
var Version string

// errToCode converts an error into an exit code.
func errToCode(a subcommands.Application, err error) int {
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %s\n", a.GetName(), err)
		return 1
	}

	return 0
}

// addExistingFlags adds all currently defined flags to a CommandRun.
//
// Some packages define flags in their init functions (e.g. glog). In order for
// these flags to be defined on a command, they need to be defined in the
// CommandRun function as well.
func addExistingFlags(c subcommands.CommandRun) {
	if !flag.Parsed() {
		panic("flag.Parse() must be called before addExistingFlags()")
	}

	flag.VisitAll(func(f *flag.Flag) {
		c.GetFlags().Var(f.Value, f.Name, f.Usage)
	})
}

var application = &subcommands.DefaultApplication{
	Name:  "testplan",
	Title: "A tool to evaluate Starlark files to generate ChromeOS chromiumos.test.api.CoverageRule protos.",
	Commands: []*subcommands.Command{
		cmdGenerate,
		cmdVersion,
		cmdGetTestable,

		subcommands.CmdHelp,
	},
}

var cmdVersion = &subcommands.Command{
	UsageLine: "version",
	ShortDesc: "Prints the Portage package version information used to build the tool.",
	CommandRun: func() subcommands.CommandRun {
		return &versionRun{}
	},
}

type versionRun struct {
	subcommands.CommandRunBase
}

func (r *versionRun) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	if Version == "" {
		fmt.Println("testplan version unknown, likely was not built with Portage")
		return 1
	}

	fmt.Printf("testplan version: %s\n", Version)

	return 0
}

var cmdGenerate = &subcommands.Command{
	UsageLine: "generate -plan plan1.star [-plan plan2.star] -dutattributes PATH -buildmetadata -out OUTPUT",
	ShortDesc: "generate CoverageRule protos",
	LongDesc: `Generate CoverageRule protos.

Evaluates Starlark files to generate CoverageRules as newline-delimited json protos.
`,
	CommandRun: func() subcommands.CommandRun {
		r := &generateRun{}
		r.Flags.Var(
			luciflag.StringSlice(&r.planPaths),
			"plan",
			"Starlark file to use. Must be specified at least once.",
		)
		r.Flags.StringVar(
			&r.dutAttributeListPath,
			"dutattributes",
			"",
			"Path to a proto file containing a DutAttributeList. Can be JSON "+
				"or binary proto.",
		)
		r.Flags.StringVar(
			&r.buildMetadataListPath,
			"buildmetadata",
			"",
			"Path to a proto file containing a SystemImage.BuildMetadataList. "+
				"Can be JSON or binary proto.",
		)
		r.Flags.StringVar(
			&r.configBundleListPath,
			"configbundlelist",
			"",
			"Path to a proto file containing a ConfigBundleList. Can be JSON or "+
				"binary proto.",
		)
		r.Flags.StringVar(
			&r.chromiumosSourceRootPath,
			"crossrcroot",
			"",
			"Path to the root of a Chromium OS source checkout. Default "+
				"versions of dutattributes, buildmetadata, configbundlelist, "+
				"and boardprioritylist in this source checkout will be used, as "+
				"a convenience to avoid specifying all these full paths. "+
				"crossrcroot is mutually exclusive with the above flags.",
		)
		r.Flags.BoolVar(
			&r.ctpV1,
			"ctpv1",
			false,
			"Output GenerateTestPlanResponse protos instead of CoverageRules, "+
				"for backwards compatibility with CTP1. Output is still "+
				"to <out>. generatetestplanreq must be set if this flag is "+
				"true",
		)
		r.Flags.StringVar(
			&r.generateTestPlanReqPath,
			"generatetestplanreq",
			"",
			"Path to a proto file containing a GenerateTestPlanRequest. Can be"+
				"JSON or binary proto. Should be set iff ctpv1 is set.",
		)
		r.Flags.StringVar(
			&r.boardPriorityListPath,
			"boardprioritylist",
			"",
			"Path to a proto file containing a BoardPriorityList. Can be JSON"+
				"or binary proto. Should be set iff ctpv1 is set.",
		)
		r.Flags.StringVar(
			&r.builderConfigsPath,
			"builderconfigs",
			"",
			"Path to a proto file containing a BuilderConfigs. Can be JSON"+
				"or binary proto. Should be set iff ctpv1 is set.",
		)
		r.Flags.StringVar(
			&r.out,
			"out",
			"",
			"Path to the output CoverageRules (or GenerateTestPlanResponse if -ctpv1 is set).",
		)

		r.templateParametersFlag.Register(&r.Flags)

		addExistingFlags(r)

		return r
	},
}

type generateRun struct {
	subcommands.CommandRunBase

	planPaths                []string
	buildMetadataListPath    string
	dutAttributeListPath     string
	configBundleListPath     string
	chromiumosSourceRootPath string
	ctpV1                    bool
	generateTestPlanReqPath  string
	boardPriorityListPath    string
	builderConfigsPath       string
	templateParametersFlag   cli.TemplateParametersFlag
	out                      string
}

func (r *generateRun) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	return errToCode(a, r.run())
}

// validateFlags checks valid flags are passed to generate, e.g. all required
// flags are set.
//
// If r.chromiumosSourceRootPath is set, other flags (e.g.
// r.dutAttributeListPath) are updated to default values relative to the source
// root.
func (r *generateRun) validateFlags() error {
	if len(r.planPaths) == 0 {
		return errors.New("at least one -plan is required")
	}

	if r.chromiumosSourceRootPath == "" {
		if r.dutAttributeListPath == "" {
			return errors.New("-dutattributes is required if -crossrcroot is not set")
		}

		if r.buildMetadataListPath == "" {
			return errors.New("-buildmetadata is required if -crossrcroot is not set")
		}

		if r.configBundleListPath == "" {
			return errors.New("-configbundlelist is required if -crossrcroot is not set")
		}

		if r.ctpV1 && r.boardPriorityListPath == "" {
			return errors.New("-boardprioritylist or -crossrcroot must be set if -ctpv1 is set")
		}

		if r.ctpV1 && r.builderConfigsPath == "" {
			return errors.New("-builderconfigs or -crossrcroot must be set if -ctpv1 is set")
		}
	} else {
		if r.dutAttributeListPath != "" || r.buildMetadataListPath != "" || r.configBundleListPath != "" || r.boardPriorityListPath != "" || r.builderConfigsPath != "" {
			return errors.New("-dutattributes, -buildmetadata, -configbundlelist, and -boardprioritylist cannot be set if -crossrcroot is set")
		}

		glog.V(2).Infof("crossrcroot set to %q, updating dutattributes, buildmetadata, and configbundlelist", r.chromiumosSourceRootPath)
		r.dutAttributeListPath = filepath.Join(r.chromiumosSourceRootPath, "src", "config", "generated", "dut_attributes.jsonproto")
		r.buildMetadataListPath = filepath.Join(r.chromiumosSourceRootPath, "src", "config-internal", "build", "generated", "build_metadata.jsonproto")
		r.configBundleListPath = filepath.Join(r.chromiumosSourceRootPath, "src", "config-internal", "hw_design", "generated", "configs.jsonproto")

		if r.ctpV1 {
			glog.V(2).Infof("crossrcroot set to %q, updating boardprioritylist and builderconfigs", r.chromiumosSourceRootPath)
			r.boardPriorityListPath = filepath.Join(r.chromiumosSourceRootPath, "src", "config-internal", "board_config", "generated", "board_priority.binaryproto")
			r.builderConfigsPath = filepath.Join(r.chromiumosSourceRootPath, "infra", "config", "generated", "builder_configs.binaryproto")
		}
	}

	if r.out == "" {
		return errors.New("-out is required")
	}

	if r.ctpV1 != (r.generateTestPlanReqPath != "") {
		return errors.New("-generatetestplanreq must be set iff -ctpv1 is set")
	}

	if !r.ctpV1 && r.boardPriorityListPath != "" {
		return errors.New("-boardprioritylist cannot be set if -ctpv1 is not set")
	}

	return nil
}

// run is the actual implementation of the generate command.
func (r *generateRun) run() error {
	ctx := context.Background()

	if err := r.validateFlags(); err != nil {
		return err
	}

	pathToTemplateParametersList, err := r.templateParametersFlag.Parse()
	if err != nil {
		return err
	}

	buildMetadataList := &buildpb.SystemImage_BuildMetadataList{}
	if err := protoio.ReadBinaryOrJSONPb(r.buildMetadataListPath, buildMetadataList); err != nil {
		return err
	}

	glog.Infof("Read %d SystemImage.Metadata from %s", len(buildMetadataList.Values), r.buildMetadataListPath)

	for _, buildMetadata := range buildMetadataList.Values {
		glog.V(2).Infof("Read BuildMetadata: %s", buildMetadata)
	}

	dutAttributeList := &testpb.DutAttributeList{}
	if err := protoio.ReadBinaryOrJSONPb(r.dutAttributeListPath, dutAttributeList); err != nil {
		return err
	}

	glog.Infof("Read %d DutAttributes from %s", len(dutAttributeList.DutAttributes), r.dutAttributeListPath)

	for _, dutAttribute := range dutAttributeList.DutAttributes {
		glog.V(2).Infof("Read DutAttribute: %s", dutAttribute)
	}

	glog.Infof("Starting read of ConfigBundleList from %s", r.configBundleListPath)

	configBundleList := &payload.ConfigBundleList{}
	if err := protoio.ReadBinaryOrJSONPb(r.configBundleListPath, configBundleList); err != nil {
		return err
	}

	glog.Infof("Read %d ConfigBundles from %s", len(configBundleList.Values), r.configBundleListPath)

	hwTestPlans, vmTestPlans, err := testplan.Generate(
		ctx, r.planPaths, buildMetadataList, dutAttributeList, configBundleList, pathToTemplateParametersList,
	)
	if err != nil {
		return err
	}

	if r.ctpV1 {
		glog.Infof(
			"Outputting GenerateTestPlanRequest to %s instead of CoverageRules, for backwards compatibility with CTPV1",
			r.out,
		)

		generateTestPlanReq := &testplans.GenerateTestPlanRequest{}
		if err := protoio.ReadBinaryOrJSONPb(r.generateTestPlanReqPath, generateTestPlanReq); err != nil {
			return err
		}

		boardPriorityList := &testplans.BoardPriorityList{}
		if err := protoio.ReadBinaryOrJSONPb(r.boardPriorityListPath, boardPriorityList); err != nil {
			return err
		}

		builderConfigs := &chromiumos.BuilderConfigs{}
		if err := protoio.ReadBinaryOrJSONPb(r.builderConfigsPath, builderConfigs); err != nil {
			return err
		}

		resp, err := compatibility.ToCTP1(
			// Disable randomness when selecting boards for now, since this can
			// lead to cases where a different board is selected on the first
			// and second CQ runs, causing test history to not be reused.
			// TODO(b/278624587): Pass a list of previously-passed tests, so
			// this can be used to ensure test reuse.
			rand.New(rand.NewSource(0)),
			hwTestPlans, vmTestPlans, generateTestPlanReq, dutAttributeList, boardPriorityList, builderConfigs,
		)
		if err != nil {
			return err
		}

		outFile, err := os.Create(r.out)
		if err != nil {
			return err
		}
		defer outFile.Close()

		respBytes, err := proto.Marshal(resp)
		if err != nil {
			return err
		}

		if _, err := outFile.Write(respBytes); err != nil {
			return err
		}

		jsonprotoOut := protoio.FilepathAsJsonpb(r.out)
		if jsonprotoOut == r.out {
			glog.Warningf("Output path set to jsonpb (%q), but output will be written as binaryproto", r.out)
		} else {
			glog.Infof("Writing jsonproto version of output to %s", jsonprotoOut)

			jsonprotoOutFile, err := os.Create(jsonprotoOut)
			if err != nil {
				return err
			}
			defer jsonprotoOutFile.Close()

			jsonprotoRespBytes, err := protojson.Marshal(resp)
			if err != nil {
				return err
			}

			if _, err := jsonprotoOutFile.Write(jsonprotoRespBytes); err != nil {
				return err
			}
		}

		return nil
	}

	var allRules []*testpb.CoverageRule
	for _, m := range hwTestPlans {
		allRules = append(allRules, m.GetCoverageRules()...)
	}

	for _, m := range vmTestPlans {
		allRules = append(allRules, m.GetCoverageRules()...)
	}

	glog.Infof("Generated %d CoverageRules, writing to %s", len(allRules), r.out)

	if err := protoio.WriteJsonl(allRules, r.out); err != nil {
		return err
	}

	return nil
}

type getTestableRun struct {
	subcommands.CommandRunBase

	planPaths              []string
	builds                 []*bbpb.Build
	buildsPath             string
	buildMetadataListPath  string
	dutAttributeListPath   string
	configBundleListPath   string
	builderConfigsPath     string
	templateParametersFlag cli.TemplateParametersFlag
}

var cmdGetTestable = &subcommands.Command{
	UsageLine: `get-testable -plan plan1.star [-plan plan2.star] -builds PATH -dutattributes PATH -buildmetadata PATH -configbundlelist PATH -builderconfigs PATH`,
	ShortDesc: "get a list of builds that could possibly be tested by plans",
	LongDesc: `Get a list of builds that could possibly be tested by plans.

First compute a set of CoverageRules from plans, then compute which builds could
possibly be tested based off the CoverageRules.

This doesn't take the status, output test artifacts, etc. of builds into
account, just whether their build target, variant, and profile could be included
in one of the CoverageRules.

The list of testable builders is printed to stdout, delimited by spaces.

Note that the main use case for this is programatically deciding which builders
to collect for testing, so it is marked as advanced, and doesn't offer
conveniences such as -crossrcroot that generate does.
	`,
	Advanced: true,
	CommandRun: func() subcommands.CommandRun {
		r := &getTestableRun{}

		r.Flags.Var(
			luciflag.StringSlice(&r.planPaths),
			"plan",
			"Starlark file to use. Must be specified at least once.",
		)
		r.Flags.StringVar(
			&r.buildsPath,
			"builds",
			"",
			"Path to a file containing Buildbucket build protos to analyze, with"+
				"one JSON proto per-line. Each proto must include the "+
				"`builder.builder` field and the `build_target.name` input "+
				"property, all other fields will be ignored")
		r.Flags.StringVar(
			&r.dutAttributeListPath,
			"dutattributes",
			"",
			"Path to a proto file containing a DutAttributeList. Can be JSON "+
				"or binary proto.",
		)
		r.Flags.StringVar(
			&r.buildMetadataListPath,
			"buildmetadata",
			"",
			"Path to a proto file containing a SystemImage.BuildMetadataList. "+
				"Can be JSON or binary proto.",
		)
		r.Flags.StringVar(
			&r.configBundleListPath,
			"configbundlelist",
			"",
			"Path to a proto file containing a ConfigBundleList. Can be JSON or "+
				"binary proto.",
		)
		r.Flags.StringVar(
			&r.builderConfigsPath,
			"builderconfigs",
			"",
			"Path to a proto file containing a BuilderConfigs. Can be JSON"+
				"or binary proto. Should be set iff ctpv1 is set.",
		)

		r.templateParametersFlag.Register(&r.Flags)

		addExistingFlags(r)

		return r
	},
}

func (r *getTestableRun) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	return errToCode(a, r.run())
}

// validateFlags checks valid flags are passed to get-testable, e.g. all
// required flags are set.
func (r *getTestableRun) validateFlags() error {
	if len(r.planPaths) == 0 {
		return errors.New("at least one -plan is required")
	}

	if len(r.buildsPath) == 0 {
		return errors.New("builds must be set")
	}

	// Parse the builds into r.builds.
	parsedBuilds, err := protoio.ReadJsonl(r.buildsPath, func() *bbpb.Build { return &bbpb.Build{} })
	if err != nil {
		return err
	}

	r.builds = parsedBuilds

	if len(r.builds) == 0 {
		return errors.New("at least one build is required")
	}

	for _, build := range r.builds {
		if build.GetBuilder().GetBuilder() == "" {
			return fmt.Errorf("builds must set builder.builder, got %q", build)
		}

		inputProps := build.GetInput().GetProperties()
		btProp, ok := inputProps.GetFields()["build_target"]
		if !ok {
			return fmt.Errorf("builds must set the build_target.name input prop, got %q", build)
		}

		if _, ok := btProp.GetStructValue().GetFields()["name"]; !ok {
			return fmt.Errorf("builds must set the build_target.name input prop, got %q", build)
		}
	}

	if r.dutAttributeListPath == "" {
		return errors.New("-dutattributes is required")
	}

	if r.buildMetadataListPath == "" {
		return errors.New("-buildmetadata is required")
	}

	if r.configBundleListPath == "" {
		return errors.New("-configbundlelist is required")
	}

	if r.builderConfigsPath == "" {
		return errors.New("-builderconfigs is required")
	}

	return nil
}

func (r *getTestableRun) run() error {
	ctx := context.Background()

	if err := r.validateFlags(); err != nil {
		return err
	}

	pathToTemplateParametersList, err := r.templateParametersFlag.Parse()
	if err != nil {
		return err
	}

	buildMetadataList := &buildpb.SystemImage_BuildMetadataList{}
	if err := protoio.ReadBinaryOrJSONPb(r.buildMetadataListPath, buildMetadataList); err != nil {
		return err
	}

	glog.Infof("Read %d SystemImage.Metadata from %s", len(buildMetadataList.Values), r.buildMetadataListPath)

	for _, buildMetadata := range buildMetadataList.Values {
		glog.V(2).Infof("Read BuildMetadata: %s", buildMetadata)
	}

	dutAttributeList := &testpb.DutAttributeList{}
	if err := protoio.ReadBinaryOrJSONPb(r.dutAttributeListPath, dutAttributeList); err != nil {
		return err
	}

	glog.Infof("Read %d DutAttributes from %s", len(dutAttributeList.DutAttributes), r.dutAttributeListPath)

	for _, dutAttribute := range dutAttributeList.DutAttributes {
		glog.V(2).Infof("Read DutAttribute: %s", dutAttribute)
	}

	glog.Infof("Starting read of ConfigBundleList from %s", r.configBundleListPath)

	configBundleList := &payload.ConfigBundleList{}
	if err := protoio.ReadBinaryOrJSONPb(r.configBundleListPath, configBundleList); err != nil {
		return err
	}

	glog.Infof("Read %d ConfigBundles from %s", len(configBundleList.Values), r.configBundleListPath)

	hwTestPlans, vmTestPlans, err := testplan.Generate(
		ctx, r.planPaths, buildMetadataList, dutAttributeList, configBundleList, pathToTemplateParametersList,
	)
	if err != nil {
		return err
	}

	builderConfigs := &chromiumos.BuilderConfigs{}
	if err := protoio.ReadBinaryOrJSONPb(r.builderConfigsPath, builderConfigs); err != nil {
		return err
	}

	glog.Infof(
		"Read %d BuilderConfigs from %s",
		len(builderConfigs.GetBuilderConfigs()),
		r.builderConfigsPath,
	)

	testableBuilds, err := compatibility.TestableBuilds(
		hwTestPlans,
		vmTestPlans,
		r.builds,
		builderConfigs,
		dutAttributeList,
	)
	if err != nil {
		return err
	}

	builderNames := make([]string, 0, len(testableBuilds))
	for _, build := range testableBuilds {
		builderNames = append(builderNames, build.GetBuilder().GetBuilder())
	}

	_, err = fmt.Fprint(os.Stdout, strings.Join(builderNames, " ")+"\n")
	return err
}

func main() {
	os.Exit(subcommands.Run(application, nil))
}
