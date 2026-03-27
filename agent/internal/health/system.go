package health

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type Check struct {
	Name    string `json:"name"`
	Status  string `json:"status"`
	Message string `json:"message"`
}

type SystemReport struct {
	Status string  `json:"status"`
	Checks []Check `json:"checks"`
}

func Diagnose(agentListen string) SystemReport {
	checks := []Check{}

	checks = append(checks, checkTargetcli())
	checks = append(checks, checkPathExists("configfs", "/sys/kernel/config"))
	checks = append(checks, checkPathExists("config_target", "/sys/kernel/config/target"))
	checks = append(checks, checkPathExists("config_iscsi", "/sys/kernel/config/target/iscsi"))
	checks = append(checks, checkPathExists("module_target_core_mod", "/sys/module/target_core_mod"))
	checks = append(checks, checkPathExists("module_iscsi_target_mod", "/sys/module/iscsi_target_mod"))
	checks = append(checks, checkAgentEndpoint(agentListen))

	overall := "ok"
	for _, c := range checks {
		if c.Status == "fail" {
			overall = "degraded"
			break
		}
	}

	return SystemReport{Status: overall, Checks: checks}
}

func checkTargetcli() Check {
	for _, candidate := range []string{"targetcli", "targetcli-fb"} {
		if p, err := exec.LookPath(candidate); err == nil {
			return Check{
				Name:    "targetcli",
				Status:  "pass",
				Message: "found " + filepath.Base(p),
			}
		}
	}
	return Check{
		Name:    "targetcli",
		Status:  "fail",
		Message: "targetcli/targetcli-fb not found in PATH",
	}
}

func checkPathExists(name, p string) Check {
	if _, err := os.Stat(p); err == nil {
		return Check{Name: name, Status: "pass", Message: p + " exists"}
	}
	return Check{Name: name, Status: "fail", Message: p + " missing"}
}

func checkAgentEndpoint(agentListen string) Check {
	if strings.HasPrefix(agentListen, "/") {
		if fi, err := os.Stat(agentListen); err == nil {
			mode := fi.Mode()
			if mode&os.ModeSocket != 0 {
				return Check{Name: "agent_endpoint", Status: "pass", Message: "unix socket ready: " + agentListen}
			}
			return Check{Name: "agent_endpoint", Status: "fail", Message: "path exists but is not socket: " + agentListen}
		}
		return Check{Name: "agent_endpoint", Status: "fail", Message: "socket not found: " + agentListen}
	}
	return Check{Name: "agent_endpoint", Status: "pass", Message: "tcp listen configured: " + agentListen}
}
