package cmd

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/castle-x/gve/internal/i18n"
	"github.com/castle-x/gve/internal/lock"
	"github.com/castle-x/gve/internal/version"
	"github.com/spf13/cobra"
)

func newDoctorCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: i18n.T("doctor_short"),
		Long:  i18n.T("doctor_long"),
		RunE:  runDoctor,
	}
}

type checkResult struct {
	name     string
	ok       bool
	version  string
	message  string
	optional bool
}

func runDoctor(cmd *cobra.Command, args []string) error {
	fmt.Printf("%s\n\n", i18n.Tf("doctor_title", version.Full()))

	results := []checkResult{
		checkGo(),
		checkGoPath(),
		checkNode(),
		checkPnpm(),
		checkGit(),
		checkAir(),
		checkGveLock(),
	}

	allOk := true
	for _, r := range results {
		if r.ok {
			fmt.Printf("  ✓ %-12s %s\n", r.name, r.version)
		} else if r.optional {
			fmt.Printf("  ○ %-12s %s\n", r.name, r.message)
		} else {
			fmt.Printf("  ✗ %-12s %s\n", r.name, r.message)
			allOk = false
		}
	}

	fmt.Println()
	if allOk {
		fmt.Println(i18n.T("doctor_all_ok"))
	} else {
		fmt.Println(i18n.T("doctor_some_failed"))
	}

	return nil
}

func checkGoPath() checkResult {
	gopath, err := exec.Command("go", "env", "GOPATH").Output()
	if err != nil {
		return checkResult{name: "GOPATH/bin", ok: false, message: i18n.T("doctor_gopath_cannot")}
	}
	bin := strings.TrimSpace(string(gopath)) + "/bin"
	_, err = exec.LookPath("gve")
	if err != nil {
		return checkResult{
			name:    "GOPATH/bin",
			ok:      false,
			message: i18n.Tf("doctor_gopath_not_in_path", bin),
		}
	}
	return checkResult{name: "GOPATH/bin", ok: true, version: i18n.T("doctor_gopath_ok")}
}

func checkGo() checkResult {
	out, err := exec.Command("go", "version").Output()
	if err != nil {
		return checkResult{name: "Go", ok: false, message: i18n.T("doctor_go_not_found")}
	}
	ver := extractVersion(string(out))
	major, minor := parseGoVersion(ver)
	if major < 1 || (major == 1 && minor < 22) {
		return checkResult{name: "Go", ok: false, message: i18n.Tf("doctor_go_too_old", ver)}
	}
	return checkResult{name: "Go", ok: true, version: ver}
}

func checkNode() checkResult {
	out, err := exec.Command("node", "--version").Output()
	if err != nil {
		return checkResult{name: "Node.js", ok: false, message: "not found (install from https://nodejs.org)"}
	}
	ver := strings.TrimSpace(string(out))
	major := parseNodeMajor(ver)
	if major < 18 {
		return checkResult{name: "Node.js", ok: false, message: fmt.Sprintf("%s (requires >= 18)", ver)}
	}
	return checkResult{name: "Node.js", ok: true, version: ver}
}

func checkPnpm() checkResult {
	out, err := exec.Command("pnpm", "--version").Output()
	if err != nil {
		return checkResult{name: "pnpm", ok: false, message: "not found (install: npm install -g pnpm)"}
	}
	ver := strings.TrimSpace(string(out))
	return checkResult{name: "pnpm", ok: true, version: ver}
}

func checkGit() checkResult {
	out, err := exec.Command("git", "--version").Output()
	if err != nil {
		return checkResult{name: "Git", ok: false, message: "not found"}
	}
	ver := extractVersion(string(out))
	return checkResult{name: "Git", ok: true, version: ver}
}

func checkAir() checkResult {
	out, err := exec.Command("air", "-v").Output()
	if err != nil {
		return checkResult{
			name:     "Air",
			ok:       false,
			optional: true,
			message:  "not found (optional, install: go install github.com/air-verse/air@latest)",
		}
	}
	ver := strings.TrimSpace(string(out))
	return checkResult{name: "Air", ok: true, version: ver}
}

func checkGveLock() checkResult {
	projectDir, err := findProjectRoot()
	if err != nil {
		return checkResult{name: "gve.lock", ok: true, version: "not in a GVE project (ok)"}
	}

	lockPath := filepath.Join(projectDir, "gve.lock")
	_, err = lock.Load(lockPath)
	if err != nil {
		return checkResult{name: "gve.lock", ok: false, message: fmt.Sprintf("invalid: %v", err)}
	}
	return checkResult{name: "gve.lock", ok: true, version: "valid"}
}

var versionRe = regexp.MustCompile(`(\d+\.\d+[\.\d]*)`)

func extractVersion(s string) string {
	match := versionRe.FindString(s)
	if match != "" {
		return match
	}
	return strings.TrimSpace(s)
}

func parseGoVersion(ver string) (major, minor int) {
	parts := strings.Split(ver, ".")
	if len(parts) >= 2 {
		major, _ = strconv.Atoi(parts[0])
		minor, _ = strconv.Atoi(parts[1])
	}
	return
}

func parseNodeMajor(ver string) int {
	ver = strings.TrimPrefix(ver, "v")
	parts := strings.Split(ver, ".")
	if len(parts) >= 1 {
		major, _ := strconv.Atoi(parts[0])
		return major
	}
	return 0
}
