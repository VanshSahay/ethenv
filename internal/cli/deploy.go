package cli

import (
	"flag"
	"fmt"

	"ethenv/internal/tf"
	"ethenv/internal/ui"
)

func runEnvCommand(args []string, verb string) error {
	fs := flag.NewFlagSet(verb, flag.ExitOnError)
	env := fs.String("env", "dev", "environment: dev or prod")
	auto := fs.Bool("auto-approve", false, "skip the interactive approval prompt")
	dir := fs.String("dir", defaultDir, "terraform module directory")
	if err := fs.Parse(args); err != nil {
		return err
	}
	r := tf.Runner{Dir: *dir}

	fmt.Println(ui.Header("[" + verb + "] environment: " + *env))

	fmt.Println(ui.Dim + "→ terraform init (skipped when already initialized)" + ui.Reset)
	if err := r.EnsureInit(); err != nil {
		return err
	}

	fmt.Println(ui.Dim + "→ workspace select/create: " + *env + ui.Reset)
	if err := r.Select(*env); err != nil {
		return err
	}

	var err error
	switch verb {
	case "deploy":
		fmt.Println(ui.Dim + "→ terraform plan -var-file=terraform.tfvars." + *env + ui.Reset)
		fmt.Println(ui.Dim + "→ terraform apply" + ui.Reset)
		err = r.Apply(*env, *auto)
	case "destroy":
		fmt.Println(ui.Dim + "→ terraform destroy -var-file=terraform.tfvars." + *env + ui.Reset)
		err = r.Destroy(*env, *auto)
	}
	if err != nil {
		return err
	}

	if verb == "deploy" {
		if outs, oerr := r.Outputs(); oerr == nil {
			if urls, uerr := tf.StringList(outs, "node_rpc_urls"); uerr == nil && len(urls) > 0 {
				fmt.Println()
				fmt.Println(ui.OK(fmt.Sprintf("deployed %d node(s)", len(urls))))
				for _, u := range urls {
					fmt.Println("   RPC " + u)
				}
				fmt.Println(ui.Dim + "next: ethenv status --env " + *env +
					"  (nodes need ~3-4 min to install docker and boot)" + ui.Reset)
			}
		}
	} else {
		fmt.Println(ui.OK("environment " + *env + " destroyed - workspace state is now empty"))
	}
	return nil
}

func runDeploy(args []string) error { return runEnvCommand(args, "deploy") }

func runDestroy(args []string) error { return runEnvCommand(args, "destroy") }
