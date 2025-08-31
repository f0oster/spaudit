//go:build mage

package main

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// ---- Config ------------------------------------------------------------------

var (
	CmdDir   = "cmd/server"
	BuildDir = "bin"
)

// ---- Helpers -----------------------------------------------------------------

func sh(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout, cmd.Stderr, cmd.Stdin = os.Stdout, os.Stderr, os.Stdin
	return cmd.Run()
}

// helper: run a command with extra env vars
func shEnv(env map[string]string, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	// inherit current env, then override/add
	cmd.Env = append(os.Environ(), func() []string {
		out := make([]string, 0, len(env))
		for k, v := range env {
			out = append(out, k+"="+v)
		}
		return out
	}()...)
	cmd.Stdout, cmd.Stderr, cmd.Stdin = os.Stdout, os.Stderr, os.Stdin
	return cmd.Run()
}

func out(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	err := cmd.Run()
	return strings.TrimSpace(buf.String()), err
}

func ensureDir(dir string) error { return os.MkdirAll(dir, 0o755) }

func which(bin string) bool {
	_, err := exec.LookPath(bin)
	return err == nil
}

func outBinPath() string {
	name := "server"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	return filepath.Join(BuildDir, name)
}

// ---- Tasks -------------------------------------------------------------------

// Bootstrap: prepare the workspace
func Bootstrap() error {
	steps := []func() error{
		ModDownload, // fetch app deps
		Deps,        // install build tools (sqlc/templ/linters/etc)
		Gen,         // run generators
	}
	for _, f := range steps {
		if err := f(); err != nil {
			return err
		}
	}
	return nil
}

// ModDownload: prefetch all module dependencies into the module cache.
func ModDownload() error {
	return sh("go", "mod", "download", "all")
}

// Deps: install CLI tooling for builds
func Deps() error {
	cmds := [][]string{
		{"go", "install", "golang.org/x/tools/cmd/goimports@latest"},
		{"go", "install", "honnef.co/go/tools/cmd/staticcheck@latest"},
		{"go", "install", "github.com/golangci/golangci-lint/cmd/golangci-lint@latest"},
		{"go", "install", "github.com/go-delve/delve/cmd/dlv@latest"},
		{"go", "install", "golang.org/x/vuln/cmd/govulncheck@latest"},
	}
	for _, c := range cmds {
		if err := sh(c[0], c[1:]...); err != nil {
			return err
		}
	}
	return nil
}

// Gen: run code generators
func Gen() error {
	return sh("go", "generate", "./...")
}

// Build: build binary into ./bin (set SKIP_GEN=1 to skip codegen)
func Build() error {
	if os.Getenv("SKIP_GEN") == "" {
		if err := Gen(); err != nil {
			return err
		}
	}
	if err := ensureDir(BuildDir); err != nil {
		return err
	}
	return sh("go", "build",
		"-trimpath", "-buildvcs=false",
		"-ldflags", "-s -w",
		"-o", outBinPath(),
		"./"+CmdDir,
	)
}

// Run: run from source
func Run() error {
	if err := Gen(); err != nil {
		return err
	}
	return sh("go", "run", "./"+CmdDir)
}

// Debug: run with delve (headless)
func Debug() error {
	if err := Gen(); err != nil {
		return err
	}
	if !which("dlv") {
		return errors.New("delve (dlv) not found; install it with 'go install github.com/go-delve/delve/cmd/dlv@latest'")
	}
	return sh("dlv", "debug", "./"+CmdDir, "--headless", "--listen=:2345", "--api-version=2", "--accept-multiclient")
}

// Vuln: check for known vulnerabilities
func Vuln() error {
	if !which("govulncheck") {
		return fmt.Errorf("govulncheck not found; run 'mage deps'")
	}
	return sh("govulncheck", "./...")
}

// Test: run unit tests with the race detector (enables cgo just for this run)
// set NO_RACE=1 if you want to skip the race detector
func Test() error {
	if os.Getenv("NO_RACE") == "1" {
		return sh("go", "test", "./...")
	}
	return shEnv(map[string]string{"CGO_ENABLED": "1"}, "go", "test", "-race", "./...")
}

// Cover: coverage report (also with race, unless NO_RACE=1)
func Cover() error {
	args := []string{"go", "test", "-coverprofile=coverage.out", "./..."}
	if os.Getenv("NO_RACE") != "1" {
		// enable cgo for race build
		if err := shEnv(map[string]string{"CGO_ENABLED": "1"}, "go", "test", "-race", "-coverprofile=coverage.out", "./..."); err != nil {
			return err
		}
	} else {
		if err := sh(args[0], args[1:]...); err != nil {
			return err
		}
	}
	fmt.Println("Coverage HTML -> coverage.html")
	return sh("go", "tool", "cover", "-html=coverage.out", "-o", "coverage.html")
}

// Lint: vet + staticcheck + golangci-lint
func Lint() error {
	for _, b := range []string{"staticcheck", "golangci-lint"} {
		if !which(b) {
			return fmt.Errorf("%s not found; run 'mage deps'", b)
		}
	}
	if err := sh("go", "vet", "./..."); err != nil {
		return err
	}
	if err := sh("staticcheck", "./..."); err != nil {
		return err
	}
	return sh("golangci-lint", "run")
}

// Fmt: go fmt + goimports -w
func Fmt() error {
	if err := sh("go", "fmt", "./..."); err != nil {
		return err
	}
	return sh("goimports", "-w", ".")
}

// FmtCheck: fail if formatting/imports needed
func FmtCheck() error {
	gofmtOut, _ := out("gofmt", "-l", ".")
	goimpOut, _ := out("goimports", "-l", ".")
	var msgs []string
	if gofmtOut != "" {
		msgs = append(msgs, "Needs gofmt:\n"+gofmtOut)
	}
	if goimpOut != "" {
		msgs = append(msgs, "Needs goimports:\n"+goimpOut)
	}
	if len(msgs) > 0 {
		return errors.New(strings.Join(msgs, "\n\n"))
	}
	return nil
}

// TidyCheck: ensure go.mod/go.sum are tidy (works with or without commits)
func TidyCheck() error {
	before, _ := out("git", "status", "--porcelain", "--", "go.mod", "go.sum")
	if err := sh("go", "mod", "tidy"); err != nil {
		return err
	}
	after, _ := out("git", "status", "--porcelain", "--", "go.mod", "go.sum")
	if before != after {
		diff, _ := out("git", "--no-pager", "diff", "--", "go.mod", "go.sum")
		return fmt.Errorf("go.mod/sum changed; run 'go mod tidy' and commit.\n%s", diff)
	}
	return nil
}

// GenerateCheck: ensure generated code is up to date
func GenerateCheck() error {
	before, _ := out("git", "status", "--porcelain")
	if err := Gen(); err != nil {
		return err
	}
	after, _ := out("git", "status", "--porcelain")
	if before != after {
		diff, _ := out("git", "--no-pager", "diff", "--", ".")
		return fmt.Errorf("generated code out of date; run 'mage gen' and commit.\n%s", diff)
	}
	return nil
}

// Clean: remove build artifacts and generated files
func Clean() error {
	_ = os.RemoveAll(BuildDir)

	removeBySuffix := func(root, suffix string) error {
		return filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
			if err != nil {
				return nil // ignore traversal errors
			}
			if d.IsDir() {
				return nil
			}
			if strings.HasSuffix(d.Name(), suffix) {
				_ = os.Remove(p)
			}
			return nil
		})
	}

	_ = removeBySuffix("gen/db", ".sql.go")
	_ = removeBySuffix("interfaces/web/templates", "_templ.go")

	return nil
}

// Verify: fast read-only checks
func Verify() error {
	steps := []func() error{FmtCheck, TidyCheck, GenerateCheck, Lint, Vuln, Build, Test}
	for _, f := range steps {
		if err := f(); err != nil {
			return err
		}
	}
	fmt.Println("✓ Build + checks passed")
	return nil
}
