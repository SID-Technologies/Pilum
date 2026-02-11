package output

import "fmt"

// PrintBuildHelp prints the styled build command reference.
func PrintBuildHelp() {
	fmt.Printf("\n%s%sBuild Command Reference%s\n\n", Bold, Primary, Reset)

	fmt.Printf("  Build commands are configured in your %spilum.yaml%s under the %sbuild%s section.\n", Accent, Reset, Accent, Reset)
	fmt.Printf("  The %scmd%s field specifies the shell command to build your service.\n\n", Accent, Reset)

	// Go
	printLanguageHeader("Go")
	printYAMLSnippet(`    build:
      language: go
      cmd: "go build -o ./dist"
      env_vars:
        CGO_ENABLED: "0"
        GOOS: linux
        GOARCH: amd64
      flags:
        ldflags:
          - "-s"
          - "-w"`)

	printHint("With version injection:")
	printYAMLSnippet(`    build:
      cmd: "go build -o ./dist"
      flags:
        ldflags:
          - "-X main.version=${TAG}"`)

	// Node
	printLanguageHeader("Node")

	printSubHeader("npm:")
	printYAMLSnippet(`    build:
      language: node
      cmd: "npm run build"
    npm:
      scope: "@my-org"
      registry: https://npm.pkg.github.com`)

	printSubHeader("pnpm:")
	printYAMLSnippet(`    build:
      language: node
      cmd: "pnpm run build"
    npm:
      package_manager: pnpm`)

	printSubHeader("pnpm monorepo:")
	printYAMLSnippet(`    build:
      language: node
      cmd: "pnpm --filter @scope/package... build"
    npm:
      package_manager: pnpm`)
	fmt.Printf("  %sThe '...' suffix builds the package AND all its workspace dependencies.%s\n\n", Muted, Reset)

	printSubHeader("yarn:")
	printYAMLSnippet(`    build:
      language: node
      cmd: "yarn build"
    npm:
      package_manager: yarn`)

	printSubHeader("bun:")
	printYAMLSnippet(`    build:
      language: node
      cmd: "bun run build"
    npm:
      package_manager: bun`)

	// Python
	printLanguageHeader("Python")
	printYAMLSnippet(`    build:
      language: python
      cmd: "pip install -r requirements.txt"
      env_vars:
        PYTHONDONTWRITEBYTECODE: "1"`)

	// Rust
	printLanguageHeader("Rust")
	printYAMLSnippet(`    build:
      language: rust
      cmd: "cargo build --release"
      flags:
        target:
          - "x86_64-unknown-linux-musl"`)

	// Package manager notes
	printSectionHeader("Package Managers")
	fmt.Printf("  The %snpm.package_manager%s field controls the install command:\n\n", Accent, Reset)
	fmt.Printf("    %snpm%s   %s(default)%s  %s→%s  npm ci\n", Bold, Reset, Muted, Reset, Muted, Reset)
	fmt.Printf("    %spnpm%s            %s→%s  pnpm install --frozen-lockfile\n", Bold, Reset, Muted, Reset)
	fmt.Printf("    %syarn%s            %s→%s  yarn install --frozen-lockfile\n", Bold, Reset, Muted, Reset)
	fmt.Printf("    %sbun%s             %s→%s  bun install --frozen-lockfile\n\n", Bold, Reset, Muted, Reset)

	// Tips
	printSectionHeader("Tips")
	fmt.Printf("  %s%s%s Use %spilum check%s to validate your configuration\n", SuccessColor, SymbolInfo, Reset, Accent, Reset)
	fmt.Printf("  %s%s%s Use %s--verbose%s to stream build output in real-time\n", SuccessColor, SymbolInfo, Reset, Accent, Reset)
	fmt.Printf("  %s%s%s Use %s--dry-run%s to see what would run without executing\n", SuccessColor, SymbolInfo, Reset, Accent, Reset)
	fmt.Printf("  %s%s%s The %slanguage%s field is informational — %scmd%s is what executes\n", WarningColor, SymbolWarning, Reset, Accent, Reset, Accent, Reset)
}

func printLanguageHeader(name string) {
	fmt.Printf("%s%s%s%s\n\n", Bold, Primary, name, Reset)
}

func printSectionHeader(name string) {
	fmt.Printf("%s%s%s%s\n\n", Bold, Primary, name, Reset)
}

func printSubHeader(name string) {
	fmt.Printf("  %s%s%s\n\n", Bold, name, Reset)
}

func printHint(text string) {
	fmt.Printf("  %s%s%s\n\n", Muted, text, Reset)
}

func printYAMLSnippet(yaml string) {
	fmt.Printf("%s%s%s\n\n", Muted, yaml, Reset)
}
