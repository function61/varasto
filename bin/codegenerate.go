package main

import (
	"github.com/function61/eventkit/codegen"
	"github.com/function61/eventkit/codegen/codegentemplates"
	"github.com/function61/gokit/dynversion/precompilationversion"
	"os"
)

//go:generate go run codegenerate.go

// FIXME: this is a dirty hack for fixing non-compiling generated code
//go:generate rm ../frontend/generated/stofuse/stofusetypes_endpoints.ts

func main() {
	if err := mainInternal(); err != nil {
		panic(err)
	}
}

func mainInternal() error {
	// normalize to root of the project
	if err := os.Chdir(".."); err != nil {
		return err
	}

	modules := []*codegen.Module{
		codegen.NewModule("stoserver/stoservertypes", "pkg/stoserver/stoservertypes/types.json", "", "pkg/stoserver/stoservertypes/commands.json"),
		codegen.NewModule("stofuse/stofusetypes", "pkg/stofuse/stofusetypes/types.json", "", ""),
	}

	opts := codegen.Opts{
		BackendModulePrefix:  "github.com/function61/varasto/pkg/",
		FrontendModulePrefix: "generated/",
		// AutogenerateModuleDocs: true,
	}

	if err := codegen.ProcessModules(modules, opts); err != nil {
		return err
	}

	// PreCompilationVersion = code generation doesn't have access to version via regular method
	if err := codegen.ProcessFile(
		codegen.Inline("frontend/generated/version.ts", codegentemplates.FrontendVersion),
		codegen.NewVersionData(precompilationversion.PreCompilationVersion()),
	); err != nil {
		return err
	}

	return nil
}
