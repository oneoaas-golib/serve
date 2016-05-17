package deploy

import (
	"fmt"
	"log"
	"strconv"
	"strings"

	marathon "github.com/gambol99/go-marathon"

	"github.com/InnovaCo/serve/app/build"
	"github.com/InnovaCo/serve/manifest"
	"github.com/fatih/color"
)

type SiteDeploy struct {}
type SiteRelease struct {}

func (_ SiteDeploy) Run(m *manifest.Manifest, sub *manifest.Manifest) error {
	conf := marathon.NewDefaultConfig()
	conf.URL = fmt.Sprintf("http://%s:8080", m.GetString("marathon.marathon-host"))
	marathonApi, _ := marathon.NewClient(conf)

	name := m.ServiceName() + "-v" + m.BuildVersion()

	app := &marathon.Application{}
	app.Name(m.GetStringOr("info.category", "") + "/" + name)
	app.Command(fmt.Sprintf("serve consul supervisor --service '%s' --port \\${PORT0} start %s", name, sub.GetStringOr("marathon.cmd", "bin/start")))
	app.Count(sub.GetIntOr("marathon.instances", 1))
	app.Memory(float64(sub.GetIntOr("marathon.mem", 256)))

	if cpu, err := strconv.ParseFloat(sub.GetStringOr("marathon.cpu", "0.1"), 64); err == nil {
		app.CPU(cpu)
	}

	if constrs := sub.GetStringOr("marathon.constraints", ""); constrs != "" {
		cs := strings.SplitN(constrs, ":", 2)
		app.AddConstraint(cs[0], "CLUSTER", cs[1])
	}

	app.AddEnv("ENV", m.Args("env"))
	app.AddEnv("SERVICE_NAME", m.ServiceName())
	app.AddEnv("MEMORY", sub.GetStringOr("marathon.mem", ""))

	app.AddUris(build.TaskRegistryUrl(m))

	if _, err := marathonApi.UpdateApplication(app, false); err != nil {
		color.Yellow("marathon <- %s", app)
		return err
	}

	color.Green("marathon <- %s", app)

	// todo: дожидаемся тут появления сервиса в консуле
	return nil
}

func (_ SiteRelease) Run(m *manifest.Manifest, sub *manifest.Manifest) error {
	log.Println("Release done!", sub)
	return nil
}
