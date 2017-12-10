package main

import (
	"bytes"
	"errors"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"syscall"

	"github.com/hashicorp/go-getter"
	"github.com/urfave/cli"
)

const ETF = `
terraform {
	backend "consul" {
		address = "localhost:8500"
		path = "{{.StatePath}}"
	}
}

resource "local_file" "{{.ResourceName}}" {
	content="{{.Content}}"
	filename="{{.Filename}}"
}
`

var (
	app *cli.App
)

/*

* Fetch module template from Consul
* Fetch Variables from Consul
* If needed, secrets are expected to be handled by Nomad
* Generate terraform file combining template and variables
* Run terraform init && apply
* Upload plan to Consul

If autoapply is true:
* Run Terraform apply
Else:
* abort

Uses Consul for remote state storage. This is done to handle concurrency as
well as to maintain the native terraform plan/apply state checking.

TODO:
* Integrate workspaces?
* Build working directory from name and resource?
 * Or use /tmp/+Consul value ?
 * Main thing is the absolute path must be the
   same. If run in Nomad, it will be rooted/contained so there should be no risk
   of leakage.
 * Look at using Afero to do it all in memory - the problem is go-getter downloading the version

*/

// TFModule needs to be redone to be specific for each resource type?
type TFModule struct {
	Source       string
	ResourceName string
	Content      string
	Filename     string
	StatePath    string
}

type TerraformSequence struct {
	Autoapply        bool
	Terrafile        string
	Template         string
	TemplateName     string
	WorkingDir       string
	PlanContent      string
	Result           string
	Workspace        string
	TerraformVersion string
	Name             string
	ResourceType     string
	Module           TFModule
	Config           *JobConfig
	AppConf          *AppConfig
	ChangesAvailable bool
}

func abortOnError(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func (t *TerraformSequence) getTemplate() (err error) {
	t.AppConf.Connect()
	path := fmt.Sprintf("%s/template", t.ResourceType)
	t.Template, err = t.AppConf.GetString(path, true)
	abortOnError(err)
	return nil
}

func (t *TerraformSequence) getDataFromConsul() (err error) {
	t.Config = NewJobConfig(t.Name, "zealot")
	t.AppConf = NewAppConfig("zealot")

	t.Config.Connect()

	t.Module.ResourceName, err = t.Config.GetString("module/ResourceName", true)
	abortOnError(err)
	t.Module.Content, err = t.Config.GetString("module/Content", true)
	abortOnError(err)
	t.WorkingDir, err = t.Config.GetString("WorkingDir", true)
	abortOnError(err)
	t.Module.Filename, err = t.Config.GetString("module/Filename", true)
	abortOnError(err)
	t.Autoapply, _ = t.Config.GetBool("autoapply", true)
	t.Module.StatePath = t.Config.GetBase() + "state"
	abortOnError(err)
	t.getTemplate()
	return err
}

func (t *TerraformSequence) getTerraform() error {
	var err error
	tfurl := fmt.Sprintf("https://releases.hashicorp.com/terraform/%s/terraform_%s_darwin_amd64.zip", t.TerraformVersion, t.TerraformVersion)
	err = getter.Get("bin", tfurl)
	if err != nil {
		log.Fatal(err)
	}
	err = os.Chmod("bin/terraform", 0755)
	if err != nil {
		log.Fatal(err)
	}
	return err
}

func (t *TerraformSequence) GenerateTemplate() error {
	var err error
	var b bytes.Buffer
	tmpl := template.Must(template.New("tffile").Parse(t.Template))
	if err != nil {
		log.Fatal(err)
	}
	err = tmpl.Execute(&b, t.Module)
	if err != nil {
		log.Fatal(err)
	}
	t.Terrafile = b.String()
	fmt.Printf("%v\n", t.Terrafile)
	return err
}

func (t *TerraformSequence) WriteFile() error {
	t.GenerateTemplate()
	filename := fmt.Sprintf("%s/main.tf", t.WorkingDir)
	return ioutil.WriteFile(filename, []byte(t.Terrafile), 0644)
}

func (t *TerraformSequence) Init() error {
	err := t.getDataFromConsul()
	if err != nil {
		return err
	}
	err = os.MkdirAll(t.WorkingDir, 0744)
	abortOnError(err)
	err = os.Chdir(t.WorkingDir)
	abortOnError(err)
	err = t.getTerraform()
	abortOnError(err)
	err = t.WriteFile()
	abortOnError(err)
	command := exec.Command("./bin/terraform", "init", "-input=false")
	output, err := command.CombinedOutput()
	if err != nil {
		return err
	}
	fmt.Printf("[INIT] %s\n", output)
	return err
}
func (t *TerraformSequence) Plan() error {
	println("[PLAN]")
	exitCode := 0
	command := exec.Command("./bin/terraform", "plan", "-out", ".plan", "-detailed-exitcode", "-no-color")
	output, err := command.CombinedOutput()
	if err != nil {
		// try to get the exit code
		if exitError, ok := err.(*exec.ExitError); ok {
			ws := exitError.Sys().(syscall.WaitStatus)
			exitCode = ws.ExitStatus()
		} else {
			// This will happen (in OSX) if `name` is not available in $PATH,
			// in this situation, exit code could not be get, and stderr will be
			// empty string very likely, so we use the default fail code, and format err
			// to string and set to stderr
			log.Printf("Could not get exit code\n")
		}
	} else {
		// success, exitCode should be 0 if go is ok
		ws := command.ProcessState.Sys().(syscall.WaitStatus)
		exitCode = ws.ExitStatus()
	}
	fmt.Printf("Exit code from plan is: %d\n", exitCode)
	switch exitCode {
	case 0:
		t.PlanContent = string(output)
		t.Config.SetValue("PlanText", t.PlanContent)
		planfile, err := ioutil.ReadFile(".plan")
		if err != nil {
			log.Fatal(err)
		}
		t.Config.SetValue("planfile", string(planfile))
	case 1:
		return err
	case 2:
		t.ChangesAvailable = true
		t.Config.SetValue("ChangesAvailable", "true")
		t.PlanContent = string(output)
		t.Config.SetValue("PlanText", t.PlanContent)
		planfile, err := ioutil.ReadFile(".plan")
		if err != nil {
			log.Fatal(err)
		}
		t.Config.SetValue("planfile", string(planfile))
	}

	fmt.Printf("PLAN\n%s\n", string(output))
	return nil
}

func (t *TerraformSequence) Apply() error {
	println("[APPLY]")
	if !(t.ChangesAvailable && t.Autoapply) {
		println("No changes available or autoapply not set, apply skipped.")
		return nil
	}
	command := exec.Command("./bin/terraform", "apply", "-input=false", ".plan")
	output, err := command.CombinedOutput()
	if err != nil {
		log.Fatal(err)
	}
	t.Result = string(output)
	fmt.Printf("PLAN\n%s\n", string(output))
	return err
}

func main() {
	app = cli.NewApp()
	app.Name = "zealot"
	app.Flags = []cli.Flag{
		cli.StringFlag{Name: "name,n"},
		cli.StringFlag{Name: "resource,r"}, // not sure I want to do it this way, but will try it out
	}
	app.Action = runMain
	app.Run(os.Args)
}

func runMain(c *cli.Context) error {
	name := c.String("name")
	if name == "" {
		log.Fatal("Need a name, Hoss.")
	}
	rtype := c.String("resource")
	if rtype == "" {
		abortOnError(errors.New("Need a name, Hoss."))
	}

	ts := TerraformSequence{Workspace: "development", TerraformVersion: "0.11.1", Name: name, ResourceType: rtype}
	ts.Autoapply = true
	ts.Module = TFModule{}
	err := ts.Init()
	if err != nil {
		log.Fatal(err)
	}
	err = ts.Plan()
	if err != nil {
		log.Fatal(err)
	}
	err = ts.Apply()
	return err
}
