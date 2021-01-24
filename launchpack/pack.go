package launchpack

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

var CONFIG_FILE_NAME = "launch.toml"
var CONFIG_ENV_KEY = "LAUNCH_CONFIG"
var DEFAULT_SHELL = "sh"

func GetPathList() []string {
	ret := []string{}
	home, err := os.UserHomeDir()
	if err == nil {
		home = filepath.Join(home, CONFIG_FILE_NAME)
		ret = append(ret, home)
	}
	conf := os.Getenv(CONFIG_ENV_KEY)
	if conf != "" {
		ret = append(ret, conf)
	}
	cwd, err := os.Getwd()
	if err == nil {
		cwd = filepath.Join(cwd, CONFIG_FILE_NAME)
		ret = append(ret, cwd)
	}
	return ret
}

func LoadPacks() []Pack {
	ret := []Pack{}
	for _, fname := range GetPathList() {
		pack := Pack{}
		pack.Shell = DEFAULT_SHELL
		e := pack.Load(fname)
		if e == nil {
			ret = append(ret, pack)
		}
	}
	return ret
}

func ShowPacks(packs []Pack, null bool) error {
	set := map[string]struct{}{}
	width := 0
	for _, pack := range packs {
		for _, action := range pack.Actions {
			if width < len(action.Name) {
				width = len(action.Name)
			}
		}
	}
	for _, pack := range packs {
		for _, action := range pack.Actions {
			if _, ok := set[action.Name]; ok {
				continue
			}
			_, e := action.Show(width, null)
			if e != nil {
				return e
			}
			set[action.Name] = struct{}{}
		}
	}
	return nil
}

func LookUpAction(packs []Pack, key string) *Action {
	for _, pack := range packs {
		for _, action := range pack.Actions {
			if action.Name == key {
				action.pack = &pack
				return &action
			}
		}
	}
	return nil
}

func LookUpDefault(packs []Pack) *Action {
	for _, pack := range packs {
		if pack.Default != nil {
			pack.Default.pack = &pack
			return pack.Default
		}
	}
	return nil
}

type Pack struct {
	Default *Action           `toml:"default"`
	Actions []Action          `toml:"actions"`
	Env     map[string]string `toml:"env"`
	Shell   string            `toml:"shell"`
}

func (pack *Pack) Load(fname string) error {
	_, e := toml.DecodeFile(fname, &pack)
	if e != nil {
		return e
	}
	return nil
}

func (p *Pack) Show(null bool) error {
	width := 0
	for _, a := range p.Actions {
		if width < len(a.Name) {
			width = len(a.Name)
		}
	}
	for _, a := range p.Actions {
		w := a.validate()
		if w == nil {
			if _, e := a.Show(width, null); e != nil {
				return e
			}
		}
	}
	return nil
}

func (p *Pack) ExecDefault() (int, bool) {
	if p.Default == nil {
		return 0, false
	}
	code := p.Default.Exec()
	return code, true
}

func (p *Pack) Exec(name string) (int, bool) {
	for _, a := range p.Actions {
		if name == a.Name {
			code := a.Exec()
			return code, true
		}
	}
	return 0, false
}

type Action struct {
	Name   string            `toml:"name"`
	Desc   string            `toml:"desc"`
	Script string            `toml:"script"`
	Shell  string            `toml:"shell"`
	Env    map[string]string `toml:"env"`
	pack   *Pack
}

func (a *Action) validate() error {
	if a.Name == "" {
		return errors.New("name field is empty")
	}
	if a.Script == "" {
		return errors.New("script field is empty")
	}
	return nil
}

func (a Action) Exec() int {
	tmp, err := ioutil.TempFile(os.TempDir(), "launchpack-")
	if err != nil {
		panic(err)
	}
	defer os.Remove(tmp.Name())
	_, err = tmp.WriteString(a.Script)
	if err != nil {
		panic(err)
	}
	err = tmp.Close()
	if err != nil {
		panic(err)
	}

	shell := a.Shell
	if shell == "" && a.pack != nil {
		shell = a.pack.Shell
	}
	cmd := exec.Command(shell, tmp.Name())
	cmd.Env = os.Environ()
	if a.pack != nil {
		for k, v := range a.pack.Env {
			cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
		}
	}
	for k, v := range a.Env {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}
	exe, _ := os.Executable()
	cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", "LAUNCHPACK", exe))
	/*
		w, e := cmd.StdinPipe()
		if e != nil {
			panic(e)
		}
		w.Write([]byte(a.Script))
		w.Close()
	*/
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if e := cmd.Run(); e != nil {
		fmt.Fprintf(os.Stderr, "%s\n", e)
	}
	return cmd.ProcessState.ExitCode()
}

func (a Action) Show(w int, null bool) (int, error) {
	if a.Name == "" {
		return 0, nil
	}
	desc := a.Desc
	if desc == "" {
		desc = "<NO DESCRIPTION>"
	}
	t := fmt.Sprintf(" %%-%ds | %%s\x00%%s\n", w)
	return fmt.Printf(t, a.Name, desc, a.Name)
}
