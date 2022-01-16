package hasura

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"github.com/ktr0731/go-fuzzyfinder"
)

const (
	CalledSeedApply     string = "seed apply"
	calledMigrateApply  string = "migrate apply"
	calledMigrateDelete string = "migrate delete"
)

var regex *regexp.Regexp

type HasuraCmd struct {
	called    string
	command   []string
	fileNames []string
	options   map[string]interface{}
	target    string
}

func NewHasuraCmd(called string, options map[string]interface{}) *HasuraCmd {
	if called == calledMigrateApply || called == calledMigrateDelete {
		setRegex()
	}
	return &HasuraCmd{called: called, options: options}
}

func (h *HasuraCmd) Run() (string, error) {
	if err := h.setFileNames(); err != nil {
		return "", err
	}
	if err := h.setTarget(); err != nil {
		return "", err
	}
	return h.setCommand().exec()
}

func (h *HasuraCmd) exec() (string, error) {
	fmt.Println("running... ", "hasura", strings.Join(h.command, " "))
	fmt.Println("")

	r, err := exec.Command("hasura", h.command...).CombinedOutput()
	return string(r), err
}

func (h *HasuraCmd) setCommand() *HasuraCmd {
	if h.target == "" {
		return h
	}

	switch h.called {
	case CalledSeedApply:
		h.command = []string{"seed", "apply", "--file", h.target}
	case calledMigrateApply:
		h.command = []string{"migrate", "apply", "--version", h.target}
	case calledMigrateDelete:
		h.command = []string{"migrate", "delete", "--version", h.target}
	default:
		return h
	}
	// set optional flags
	for k, v := range h.options {
		h.command = append(h.command, fmt.Sprintf("--%s", k))
		switch v := v.(type) {
		case string:
			h.command = append(h.command, v)
		case bool:
			h.command = append(h.command, strconv.FormatBool(v))
		}
	}
	return h
}

func (h *HasuraCmd) setTarget() error {
	fileName, err := h.findOne()
	if err != nil {
		return err
	}
	if h.called == calledMigrateApply || h.called == calledMigrateDelete {
		h.target = trimVersion(fileName)
	} else {
		h.target = fileName
	}
	return nil
}

func (h *HasuraCmd) setFileNames() error {
	var filePath string
	switch h.called {
	case CalledSeedApply:
		filePath = fmt.Sprintf("./seeds/%s", h.options["database-name"])
	case calledMigrateApply, calledMigrateDelete:
		filePath = fmt.Sprintf("./migrations/%s", h.options["database-name"])
	}
	files, err := ioutil.ReadDir(filePath)

	if err != nil {
		return err
	}
	if len(files) == 0 {
		return errors.New("no file")
	}

	for _, file := range files {
		if file.IsDir() {
			if h.called == calledMigrateApply || h.called == calledMigrateDelete {
				h.fileNames = append(h.fileNames, file.Name())
			}
		}
		if !file.IsDir() && h.called == CalledSeedApply {
			h.fileNames = append(h.fileNames, file.Name())
		}
	}
	return nil
}

func (h *HasuraCmd) findOne() (string, error) {
	type fileName struct {
		name string
	}
	var fileNames []fileName //nolint:prealloc // Since filenames include directory names, they are less in length than in capacity.
	for _, f := range h.fileNames {
		fileNames = append(fileNames, fileName{f})
	}
	i, err := fuzzyfinder.Find(fileNames, func(i int) string { return fileNames[i].name })
	if err != nil {
		return "", err
	}
	return fileNames[i].name, nil
}

func trimVersion(fileName string) string {
	return string(regex.Find([]byte(fileName)))
}

func setRegex() {
	regex = regexp.MustCompile(`^[0-9]+`)
}
