package manifest

import (
	"fmt"
	"io/ioutil"
	"log"
	"strconv"
	"strings"

	"github.com/Jeffail/gabs"
	"github.com/codegangsta/cli"
	"github.com/fatih/color"
	"github.com/ghodss/yaml"
	"github.com/valyala/fasttemplate"
)

func LoadManifest(c *cli.Context) *Manifest {
	data, err := ioutil.ReadFile(c.String("manifest"))
	if err != nil {
		log.Fatalln(color.RedString("Manifest file `%s` not found: %v", c.String("manifest"), err))
	}

	jsonData, err := yaml.YAMLToJSON(data)
	if err != nil {
		log.Fatalln(color.RedString("Error on parse manifest: %v!", err))
	}

	tree, _ := gabs.ParseJSON(jsonData)

	for _, fn := range append(c.GlobalFlagNames(), c.FlagNames()...) {
		if v := c.String(fn); v != "" {
			tree.Set(v, "args", fn)
		} else if v := c.GlobalString(fn); v != "" {
			tree.Set(v, "args", fn)
		}
	}

	m := &Manifest{tree: tree, ctx: c}
	m.parent = m
	return m
}

type Manifest struct {
	tree   *gabs.Container
	ctx    *cli.Context
	parent *Manifest
}

func (m Manifest) String() string {
	return m.tree.String()
}

func (m *Manifest) Has(path string) bool {
	return m.tree.ExistsP(path)
}

func (m *Manifest) Sub(path string) *Manifest {
	return &Manifest{m.tree.Path(path), m.ctx, m.parent}
}

func (m *Manifest) Array(path string) []*Manifest {
	result := make([]*Manifest, 0)

	if chs, err := m.tree.Path(path).Children(); err == nil {
		for _, ch := range chs {
			result = append(result, &Manifest{ch, m.ctx, m.parent})
		}
	}

	return result
}

func (m *Manifest) FirstKey() (string, error) {
	if s, ok := m.tree.Data().(string); ok {
		return s, nil
	}

	if res, err := m.tree.ChildrenMap(); err == nil {
		for k, _ := range res {
			return k, nil
		}
	}

	return "", fmt.Errorf("Object %v has no keys!", m)
}

func (m *Manifest) Args(name string) string {
	return m.parent.GetStringOr("args."+name, "")
}

func (m *Manifest) Template(path string) string {
	return fasttemplate.New(m.GetStringOr(path, ""), "{{", "}}").ExecuteString(map[string]interface{}{
		"feature": m.Args("feature"),
	})
}

func (m *Manifest) TemplateOr(path string, defaultVal string) string {
	if m.tree.ExistsP(path) {
		return m.Template(path)
	} else {
		return defaultVal
	}
}

func (m *Manifest) GetString(path string) string {
	return fmt.Sprintf("%v", m.value(path))
}

func (m *Manifest) GetStringOr(path string, defaultVal string) string {
	if m.tree.ExistsP(path) {
		return m.GetString(path)
	} else {
		return defaultVal
	}
}

func (m *Manifest) GetInt(path string) int {
	i, err := strconv.Atoi(m.GetString(path))

	if err != nil {
		log.Fatalln(color.RedString("Value is not integer: %s = %s", path, m.GetString(path)))
	}

	return i
}

func (m *Manifest) GetIntOr(path string, defaultVal int) int {
	if m.tree.ExistsP(path) {
		return m.GetInt(path)
	} else {
		return defaultVal
	}
}

func (m *Manifest) GetBool(path string) bool {
	v, err := strconv.ParseBool(m.GetStringOr(path, "false"))
	if err != nil {
		log.Fatalln(color.RedString("Value is not a boolean: %s = %s", path, m.GetString(path)))
	}
	return v
}

func (m *Manifest) value(path string) interface{} {
	if m.tree.ExistsP(path) {
		d := m.tree.Path(path).Data()

		if obj, ok := d.(map[string]interface{}); ok {
			if v, ok := obj[m.Args("env")]; ok {
				d = v
			} else {
				log.Fatalln(color.RedString("manifest: not found '%s' in %s", m.Args("env"), m.tree.Path(path).String()))
			}
		}

		return d
	} else {
		log.Fatalln(color.RedString("manifest: path `%s` not found in %v", path, m))
		return nil
	}
}

func (m *Manifest) ServiceName() string {
	suffix := ""

	if f := m.Args("feature"); f != "" {
		suffix = "-" + f
	}

	return m.GetString("info.name") + suffix
}

func (m *Manifest) BuildVersion() string {
	return fmt.Sprintf("%s.%s", m.GetString("info.version"), m.Args("build-number"))
}

func (m *Manifest) ServiceFullName(separator string) string {
	return strings.TrimPrefix(strings.Replace(m.GetStringOr("info.category", "")+"/", "/", separator, -1)+m.ServiceName(), separator)
}

func (m *Manifest) ServiceFullNameWithVersion(separator string) string {
	return m.ServiceFullName(separator) + "-v" + m.BuildVersion()
}
