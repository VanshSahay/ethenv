package config

import (
	"bufio"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type Env struct {
	Name         string
	Region       string
	InstanceType string
	NodeCount    int
	ChainID      int
	CliquePeriod int
	VolumeSize   int
	Extra        map[string]string
}

func List(dir string) ([]Env, error) {
	matches, err := filepath.Glob(filepath.Join(dir, "terraform.tfvars.*"))
	if err != nil {
		return nil, err
	}
	envs := make([]Env, 0, len(matches))
	for _, path := range matches {
		env, err := Parse(path)
		if err != nil {
			return nil, err
		}
		envs = append(envs, env)
	}
	return envs, nil
}

func Parse(path string) (Env, error) {
	env := Env{Name: envName(path), Extra: map[string]string{}}
	f, err := os.Open(path)
	if err != nil {
		return env, err
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if i := strings.Index(line, "#"); i >= 0 {
			line = strings.TrimSpace(line[:i])
		}
		if i := strings.Index(line, "//"); i >= 0 {
			line = strings.TrimSpace(line[:i])
		}
		if line == "" {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.Trim(strings.TrimSpace(value), `"`)

		switch key {
		case "aws_region":
			env.Region = value
		case "instance_type":
			env.InstanceType = value
		case "node_count":
			env.NodeCount = atoi(value)
		case "chain_id":
			env.ChainID = atoi(value)
		case "clique_period":
			env.CliquePeriod = atoi(value)
		case "ebs_volume_size":
			env.VolumeSize = atoi(value)
		default:
			env.Extra[key] = value
		}
	}
	return env, sc.Err()
}

func (e Env) Values() map[string]string {
	v := map[string]string{
		"aws_region":      e.Region,
		"instance_type":   e.InstanceType,
		"node_count":      strconv.Itoa(e.NodeCount),
		"chain_id":        strconv.Itoa(e.ChainID),
		"clique_period":   strconv.Itoa(e.CliquePeriod),
		"ebs_volume_size": strconv.Itoa(e.VolumeSize),
	}
	for k, val := range e.Extra {
		v[k] = val
	}
	return v
}

func envName(path string) string {
	return strings.TrimPrefix(filepath.Base(path), "terraform.tfvars.")
}

func atoi(s string) int {
	n, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil {
		return 0
	}
	return n
}
