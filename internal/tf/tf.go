package tf

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type Runner struct {
	Dir string
}

func (r Runner) cmd(args ...string) *exec.Cmd {
	c := exec.Command("terraform", args...)
	c.Dir = r.Dir
	return c
}

func (r Runner) capture(args ...string) (string, error) {
	out, err := r.cmd(args...).CombinedOutput()
	return strings.TrimSpace(string(out)), err
}

func (r Runner) stream(args ...string) error {
	c := r.cmd(args...)
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c.Run()
}

func (r Runner) varFile(env string) string {
	return "-var-file=terraform.tfvars." + env
}

func (r Runner) EnsureInit() error {
	if _, err := os.Stat(filepath.Join(r.Dir, ".terraform")); err == nil {
		return nil
	}
	return r.stream("init", "-input=false", "-no-color")
}

func (r Runner) Workspaces() ([]string, error) {
	out, err := r.capture("workspace", "list", "-no-color")
	if err != nil {
		return nil, fmt.Errorf("terraform workspace list: %w\n%s", err, out)
	}
	var ws []string
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), "*"))
		if line != "" {
			ws = append(ws, line)
		}
	}
	return ws, nil
}

func (r Runner) Select(ws string) error {
	existing, err := r.Workspaces()
	if err != nil {
		return err
	}
	for _, w := range existing {
		if w == ws {
			_, err := r.capture("workspace", "select", ws, "-no-color")
			return err
		}
	}
	return r.stream("workspace", "new", ws, "-no-color")
}

func (r Runner) Validate() error {
	return r.stream("validate", "-no-color")
}

func (r Runner) FmtCheck() (string, error) {
	return r.capture("fmt", "-check", "-recursive", "-no-color")
}

func (r Runner) Plan(env string) error {
	return r.stream("plan", r.varFile(env), "-out", "tfplan-"+env, "-no-color")
}

func (r Runner) Apply(env string, autoApprove bool) error {
	if err := r.Plan(env); err != nil {
		return err
	}
	args := []string{"apply"}
	if autoApprove {
		args = append(args, "-auto-approve")
	}
	return r.stream(append(args, "-no-color", "tfplan-"+env)...)
}

func (r Runner) Destroy(env string, autoApprove bool) error {
	args := []string{"destroy", r.varFile(env)}
	if autoApprove {
		args = append(args, "-auto-approve")
	}
	return r.stream(append(args, "-no-color")...)
}

type Output struct {
	Value json.RawMessage `json:"value"`
}

func (r Runner) OutputsFor(env string) (map[string]Output, error) {
	statePath := filepath.Join("terraform.tfstate.d", env, "terraform.tfstate")
	if _, err := os.Stat(filepath.Join(r.Dir, statePath)); err != nil {
		return nil, fmt.Errorf("environment %q has no terraform state - deploy it first", env)
	}
	out, err := r.capture("output", "-json", "-no-color", "-state="+statePath)
	if err != nil {
		return nil, fmt.Errorf("terraform output: %w", err)
	}
	var outs map[string]Output
	if err := json.Unmarshal([]byte(out), &outs); err != nil {
		return nil, err
	}
	return outs, nil
}

func StringList(outs map[string]Output, name string) ([]string, error) {
	o, ok := outs[name]
	if !ok {
		return nil, fmt.Errorf("missing terraform output %q", name)
	}
	var list []string
	if err := json.Unmarshal(o.Value, &list); err != nil {
		return nil, err
	}
	return list, nil
}

func (r Runner) HasState() bool {
	if _, err := os.Stat(filepath.Join(r.Dir, "terraform.tfstate")); err == nil {
		return true
	}
	_, err := os.Stat(filepath.Join(r.Dir, "terraform.tfstate.d"))
	return err == nil
}
