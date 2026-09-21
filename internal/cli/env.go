package cli

import (
	"flag"
	"fmt"
	"os"

	"ethenv/internal/config"
	"ethenv/internal/tf"
	"ethenv/internal/ui"
)

func runEnv(args []string) error {
	fs := flag.NewFlagSet("env", flag.ExitOnError)
	dir := fs.String("dir", defaultDir, "terraform module directory")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if sub := fs.Arg(0); sub != "" && sub != "list" {
		return fmt.Errorf("unknown env subcommand %q (try: ethenv env list)", sub)
	}

	envs, err := config.List(*dir)
	if err != nil {
		return err
	}
	if len(envs) == 0 {
		return fmt.Errorf("no terraform.tfvars.* files found in %s", *dir)
	}

	ws, _ := (tf.Runner{Dir: *dir}).Workspaces()
	created := map[string]bool{}
	for _, w := range ws {
		created[w] = true
	}

	rows := make([][]string, 0, len(envs))
	for _, e := range envs {
		state := ui.Dim + "not created" + ui.Reset
		if created[e.Name] {
			state = ui.Green + "created" + ui.Reset
		}
		rows = append(rows, []string{
			e.Name,
			fmt.Sprintf("%d x %s", e.NodeCount, e.InstanceType),
			fmt.Sprintf("%d", e.ChainID),
			fmt.Sprintf("%d GiB", e.VolumeSize),
			e.Region,
			state,
		})
	}

	fmt.Println(ui.Header("environments (tfvars + workspace state):"))
	ui.Table(os.Stdout,
		[]string{"ENV", "NODES", "CHAIN_ID", "VOLUME", "REGION", "WORKSPACE"},
		rows)
	return nil
}
