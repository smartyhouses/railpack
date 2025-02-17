package generate

import (
	"fmt"
	"maps"
	"sort"
	"strings"

	a "github.com/railwayapp/railpack/core/app"
	"github.com/railwayapp/railpack/core/mise"
	"github.com/railwayapp/railpack/core/plan"
	"github.com/railwayapp/railpack/core/resolver"
)

const (
	MisePackageStepName = "packages:mise"
	BuilderBaseImage    = "ghcr.io/railwayapp/railpack-builder-base:latest"
)

type MiseStepBuilder struct {
	DisplayName           string
	Resolver              *resolver.Resolver
	SupportingAptPackages []string
	MisePackages          []*resolver.PackageRef
	SupportingMiseFiles   []string
	Assets                map[string]string
	Outputs               *[]string
	app                   *a.App
	env                   *a.Environment
}

func (c *GenerateContext) newMiseStepBuilder() *MiseStepBuilder {
	step := &MiseStepBuilder{
		DisplayName:           MisePackageStepName,
		Resolver:              c.Resolver,
		MisePackages:          []*resolver.PackageRef{},
		SupportingAptPackages: []string{},
		Assets:                map[string]string{},
		Outputs:               &[]string{"/mise/shims", "/mise/installs", "/usr/local/bin/mise", "/etc/mise/config.toml", "/root/.local/state/mise"},
		app:                   c.App,
		env:                   c.Env,
	}

	c.Steps = append(c.Steps, step)

	return step
}

func (b *MiseStepBuilder) AddSupportingAptPackage(name string) {
	b.SupportingAptPackages = append(b.SupportingAptPackages, name)
}

func (b *MiseStepBuilder) Default(name string, defaultVersion string) resolver.PackageRef {
	for _, pkg := range b.MisePackages {
		if pkg.Name == name {
			return *pkg
		}
	}

	pkg := b.Resolver.Default(name, defaultVersion)
	b.MisePackages = append(b.MisePackages, &pkg)
	return pkg
}

func (b *MiseStepBuilder) Version(name resolver.PackageRef, version string, source string) {
	b.Resolver.Version(name, version, source)
}

func (b *MiseStepBuilder) Name() string {
	return b.DisplayName
}

func (b *MiseStepBuilder) Build(options *BuildStepOptions) (*plan.Step, error) {
	step := plan.NewStep(b.DisplayName)

	if len(b.MisePackages) == 0 {
		return step, nil
	}

	step.StartingImage = BuilderBaseImage

	// Setup mise
	step.AddCommands([]plan.Command{
		plan.NewPathCommand("/mise/shims"),
	})
	maps.Copy(step.Variables, map[string]string{
		"MISE_DATA_DIR":     "/mise",
		"MISE_CONFIG_DIR":   "/mise",
		"MISE_CACHE_DIR":    "/mise/cache",
		"MISE_SHIMS_DIR":    "/mise/shims",
		"MISE_INSTALLS_DIR": "/mise/installs",
	})

	if verbose := b.env.GetVariable("MISE_VERBOSE"); verbose != "" {
		step.Variables["MISE_VERBOSE"] = verbose
	}

	// Add user mise config files if they exist
	supportingMiseConfigFiles := b.GetSupportingMiseConfigFiles(b.app.Source)
	for _, file := range supportingMiseConfigFiles {
		step.AddCommands([]plan.Command{
			plan.NewCopyCommand(file, "/app/"+file),
		})

		// We want to make sure the file is copied into the next step
		*b.Outputs = append(*b.Outputs, "/app/"+file)
	}

	// Setup apt commands
	if len(b.SupportingAptPackages) > 0 {
		step.AddCommands([]plan.Command{
			options.NewAptInstallCommand(b.SupportingAptPackages),
		})
		step.Caches = options.Caches.GetAptCaches()
	}

	// Setup mise commands
	if len(b.MisePackages) > 0 {
		packagesToInstall := make(map[string]string)
		for _, pkg := range b.MisePackages {
			resolved, ok := options.ResolvedPackages[pkg.Name]
			if ok && resolved.ResolvedVersion != nil {
				packagesToInstall[pkg.Name] = *resolved.ResolvedVersion
			}
		}

		miseToml, err := mise.GenerateMiseToml(packagesToInstall)
		if err != nil {
			return nil, fmt.Errorf("failed to generate mise.toml: %w", err)
		}

		b.Assets["mise.toml"] = miseToml

		pkgNames := make([]string, 0, len(packagesToInstall))
		for k := range packagesToInstall {
			pkgNames = append(pkgNames, k)
		}
		sort.Strings(pkgNames)

		step.AddCommands([]plan.Command{
			plan.NewFileCommand("/etc/mise/config.toml", "mise.toml", plan.FileOptions{
				CustomName: "create mise config",
			}),
			plan.NewExecCommand("sh -c 'mise trust -a && mise install'", plan.ExecOptions{
				CustomName: "install mise packages: " + strings.Join(pkgNames, ", "),
			}),
		})
	}

	step.Assets = b.Assets
	step.Outputs = b.Outputs
	step.UseSecrets = &[]bool{false}[0]

	return step, nil
}

var miseConfigFiles = []string{
	"mise.toml",
	".tool-versions",
	".python-version",
	".nvmrc",
}

func (b *MiseStepBuilder) GetSupportingMiseConfigFiles(path string) []string {
	files := []string{}

	for _, file := range miseConfigFiles {
		if b.app.HasMatch(file) {
			files = append(files, file)
		}
	}

	return files
}
