package core

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/railwayapp/railpack-go/core/plan"
	"github.com/railwayapp/railpack-go/core/resolver"
)

// Define styles
var (
	subtle    = lipgloss.AdaptiveColor{Light: "#D9DCCF", Dark: "#383838"}
	highlight = lipgloss.AdaptiveColor{Light: "#874BFD", Dark: "#7D56F4"}
	header    = lipgloss.AdaptiveColor{Light: "#F9F9F9", Dark: "26"}
	special   = lipgloss.AdaptiveColor{Light: "#43BF6D", Dark: "#73F59F"}

	headerStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(highlight).
			MarginTop(1).
			Padding(0, 1)

	sectionHeaderStyle = lipgloss.NewStyle().
				Bold(true).
				Width(10).
				MarginLeft(1).
				MarginTop(2).
				BorderStyle(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("238")).
				BorderBottom(true)

	packageNameStyle = lipgloss.NewStyle().
				MarginLeft(1).
				Foreground(lipgloss.Color("13"))

	versionStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("14"))

	sourceStyle = lipgloss.NewStyle()

	separatorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("238")).
			Margin(0, 2)

	stepHeaderStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("13"))

	commandPrefixStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("245")).
				MarginLeft(2)

	commandStyle = lipgloss.NewStyle().
			Bold(true)
)

func PrettyPrintBuildResult(buildResult *BuildResult) {
	output := FormatBuildResult(buildResult)
	fmt.Print(output)
}

func FormatBuildResult(br *BuildResult) string {
	var output strings.Builder

	// Header section
	header := "Railpack v0.0.1"
	output.WriteString(headerStyle.Render(header))
	output.WriteString("\n")

	// Packages section
	if len(br.ResolvedPackages) > 0 {
		output.WriteString(sectionHeaderStyle.MarginTop(1).Render("Packages"))
		output.WriteString("\n")

		// Calculate column widths
		nameWidth := 1
		versionWidth := 1
		for _, pkg := range br.ResolvedPackages {
			nameWidth = max(nameWidth, len(pkg.Name))
			if pkg.ResolvedVersion != nil {
				versionWidth = max(versionWidth, len(*pkg.ResolvedVersion))
			}
		}

		// Adjust styles with calculated widths
		packageNameStyle = packageNameStyle.Width(nameWidth).MaxWidth(20)
		versionStyle = versionStyle.Width(versionWidth).MaxWidth(20)

		separator := separatorStyle.Render("│")

		// Sort and format packages
		for _, pkg := range br.ResolvedPackages {
			name := packageNameStyle.Render(pkg.Name)

			version := "-"
			if pkg.ResolvedVersion != nil {
				version = *pkg.ResolvedVersion
			}
			version = versionStyle.Render(version)
			source := sourceStyle.Render(formatSource(pkg))
			output.WriteString(fmt.Sprintf("%s%s%s%s%s", name, separator, version, separator, source))
			output.WriteString("\n")
		}
	}

	// Steps section
	if br.Plan != nil && len(br.Plan.Steps) > 0 {
		output.WriteString(sectionHeaderStyle.Render("Steps"))
		output.WriteString("\n")

		stepCount := 0
		for _, step := range br.Plan.Steps {
			if step.Name != "packages" && step.Commands != nil { // Skip the packages step
				hasExecCommands := false
				var execCommands []string

				for _, cmd := range step.Commands {
					if execCmd, ok := cmd.(plan.ExecCommand); ok {
						hasExecCommands = true
						execCommands = append(execCommands, execCmd.Cmd)
					}
				}

				if hasExecCommands {
					customStepHeaderStyle := stepHeaderStyle
					if stepCount > 0 {
						customStepHeaderStyle = customStepHeaderStyle.MarginTop(1)
					}

					output.WriteString(customStepHeaderStyle.Render(fmt.Sprintf("▸ %s", step.Name)))
					output.WriteString("\n")

					for _, cmd := range execCommands {
						output.WriteString(fmt.Sprintf("%s %s", commandPrefixStyle.Render("$"), commandStyle.Render(cmd)))
						output.WriteString("\n")
					}
				}

				stepCount++
			}

		}
	}

	if br.Plan.Start.Command != "" {
		output.WriteString(sectionHeaderStyle.MarginTop(1).Render("Start"))
		output.WriteString("\n")
		output.WriteString(fmt.Sprintf("%s %s", commandPrefixStyle.Render("$"), commandStyle.Render(br.Plan.Start.Command)))
	}

	output.WriteString("\n\n")
	return output.String()
}

func formatSource(pkg *resolver.ResolvedPackage) string {
	if pkg.RequestedVersion != nil {
		return fmt.Sprintf("%s (%s)", pkg.Source, *pkg.RequestedVersion)
	}
	return pkg.Source
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
