package cli

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"ethenv/internal/config"
	"ethenv/internal/tf"
	"ethenv/internal/ui"
)

func runVerify(args []string) error {
	fs := flag.NewFlagSet("verify", flag.ExitOnError)
	dir := fs.String("dir", defaultDir, "terraform module directory")
	if err := fs.Parse(args); err != nil {
		return err
	}

	ctx := &verifyCtx{dir: *dir, envs: map[string]config.Env{}}
	envs, err := config.List(*dir)
	if err != nil {
		return err
	}
	for _, e := range envs {
		ctx.envs[e.Name] = e
	}

	checks := []struct {
		parameter string
		run       func(*verifyCtx) (string, bool)
	}{
		{"1. Workspaces (2, isolated state)", checkWorkspaces},
		{"2. Variables (5+, tfvars differ)", checkVariables},
		{"3. Data blocks (3+, no hardcoded ids)", checkDataBlocks},
		{"4. Code quality (validate + fmt + build)", checkQuality},
		{"5. Env config (dev cheap, prod powerful)", checkEnvConfig},
	}

	pass := 0
	for _, c := range checks {
		detail, ok := c.run(ctx)
		if ok {
			pass++
		}
		if ok {
			fmt.Println(ui.OK(c.parameter))
		} else {
			fmt.Println(ui.Fail(c.parameter))
		}
		fmt.Println(ui.Dim + "     " + detail + ui.Reset)
	}

	fmt.Println()
	if pass == len(checks) {
		fmt.Println(ui.OK(fmt.Sprintf("%d/%d evaluation parameters satisfied", pass, len(checks))))
		return nil
	}
	fmt.Println(ui.Fail(fmt.Sprintf("%d/%d evaluation parameters satisfied", pass, len(checks))))
	os.Exit(1)
	return nil
}

type verifyCtx struct {
	dir  string
	envs map[string]config.Env
}

func checkWorkspaces(ctx *verifyCtx) (string, bool) {
	ws, err := (tf.Runner{Dir: ctx.dir}).Workspaces()
	if err != nil {
		return "terraform workspace list failed - run a deploy first: " + err.Error(), false
	}
	dev, prod := false, false
	for _, w := range ws {
		switch w {
		case "dev":
			dev = true
		case "prod":
			prod = true
		}
	}
	detail := fmt.Sprintf("workspaces: %s (state isolated under terraform.tfstate.d/<ws>)", strings.Join(ws, ", "))
	return detail, dev && prod
}

func checkVariables(ctx *verifyCtx) (string, bool) {
	data, err := os.ReadFile(filepath.Join(ctx.dir, "variables.tf"))
	if err != nil {
		return err.Error(), false
	}
	vars := len(regexp.MustCompile(`(?m)^variable "`).FindAllString(string(data), -1))

	dev, hasDev := ctx.envs["dev"]
	prod, hasProd := ctx.envs["prod"]
	if !hasDev || !hasProd {
		return "missing terraform.tfvars.dev / terraform.tfvars.prod", false
	}
	dv, pv := dev.Values(), prod.Values()
	var diffKeys []string
	for k, v := range dv {
		if v != pv[k] {
			diffKeys = append(diffKeys, k)
		}
	}
	detail := fmt.Sprintf("%d variables declared; tfvars keys that differ: %s", vars, strings.Join(diffKeys, ", "))
	return detail, vars >= 5 && len(diffKeys) > 0
}

var hardcodedID = regexp.MustCompile(`\b(vpc|subnet|ami|sg|igw|vol)-[0-9a-f]{6,}`)

func checkDataBlocks(ctx *verifyCtx) (string, bool) {
	data, err := os.ReadFile(filepath.Join(ctx.dir, "main.tf"))
	if err != nil {
		return err.Error(), false
	}
	blocks := len(regexp.MustCompile(`(?m)^data "`).FindAllString(string(data), -1))
	hardcoded := hardcodedID.FindAllString(string(data), -1)

	detail := fmt.Sprintf("%d data blocks (aws_ami, aws_vpc, aws_subnets, aws_availability_zones)", blocks)
	if len(hardcoded) > 0 {
		detail += "; HARDCODED IDS FOUND: " + strings.Join(hardcoded, ", ")
	}
	return detail, blocks >= 3 && len(hardcoded) == 0
}

func checkQuality(ctx *verifyCtx) (string, bool) {
	var problems []string
	r := tf.Runner{Dir: ctx.dir}
	if err := r.Validate(); err != nil {
		problems = append(problems, "terraform validate failed")
	}
	if unformatted, err := r.FmtCheck(); err == nil && len(unformatted) > 0 {
		problems = append(problems, "terraform fmt needed for: "+strings.Join(strings.Fields(unformatted), " "))
	}

	build := exec.Command("go", "build", "./...")
	build.Dir = filepath.Dir(ctx.dir)
	if out, err := build.CombinedOutput(); err != nil {
		problems = append(problems, "go build failed: "+strings.TrimSpace(string(out)))
	}

	if len(problems) == 0 {
		return "terraform validate ✔, terraform fmt clean ✔, go build ✔", true
	}
	return strings.Join(problems, "; "), false
}

func checkEnvConfig(ctx *verifyCtx) (string, bool) {
	dev, hasDev := ctx.envs["dev"]
	prod, hasProd := ctx.envs["prod"]
	if !hasDev || !hasProd {
		return "missing env files", false
	}
	detail := fmt.Sprintf(
		"dev: %d x %s, %d GiB, chain %d  |  prod: %d x %s, %d GiB, chain %d",
		dev.NodeCount, dev.InstanceType, dev.VolumeSize, dev.ChainID,
		prod.NodeCount, prod.InstanceType, prod.VolumeSize, prod.ChainID)
	ok := prod.NodeCount > dev.NodeCount &&
		prod.VolumeSize >= dev.VolumeSize &&
		dev.ChainID != prod.ChainID
	return detail, ok
}
