package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"os"
	"os/exec"
	"path"
	"time"
)

const (
	XDG_CONFIG_HOME = "XDG_CONFIG_HOME"

	ASTATUS_BG    = "ASTATUS_BACKGROUND"
	ASTATUS_COLOR = "ASTATUS_COLOR"

	I3BAR_PROTO_HEADER = `{"version": 1, "click_events": false}
[`
)

var (
	cfgPathOption = flag.String("c", "", "config path (~/.config/astatus/astatus if not specified)")
	cfgPath       string

	commands   []string
	background = "#282936BF"
	color      = "#F5F5F5FF"
)

type CommandUpdate struct {
	idx  int
	data string
}

func GetUpdates(commands []string) chan CommandUpdate {
	updates := make(chan CommandUpdate)
	for i, cmd := range commands {
		go func(i int, cmd string, updates chan CommandUpdate) {
			command := exec.Command(cmd)
			r, err := command.StdoutPipe()
			if err != nil {
				panic(err)
			}
			if err := command.Start(); err != nil {
				//panic(err)
			}
			s := bufio.NewScanner(r)
			for s.Scan() {
				updates <- CommandUpdate{idx: i, data: s.Text()}
			}
			//panic(s.Err())
		}(i, cmd, updates)
	}
	return updates
}

type StatusItem struct {
	Background          string `json:"background"`
	Color               string `json:"color"`
	FullText            string `json:"full_text"`
	Markup              string `json:"markup"`
	Separator           bool   `json:"separator"`
	SeparatorBlockWidth int    `json:"separator_block_width"`
}

func NewStatusItem() StatusItem {
	return StatusItem{
		Background:          background,
		Color:               color,
		FullText:            "",
		Markup:              "pango",
		Separator:           false,
		SeparatorBlockWidth: 0,
	}
}

func init() {
	if newBg := os.Getenv(ASTATUS_BG); newBg != "" {
		background = newBg
	}
	if newColor := os.Getenv(ASTATUS_COLOR); newColor != "" {
		color = newColor
	}

	// default config location: ~/.config/astatus/astatus
	home, _ := os.UserHomeDir()
	cfgPath = path.Join(home, ".config", "astatus", "astatus")

	// $XDG_CONFIG_HOME is set: open ${XDG_CONFIG_HOME}/astatus/astatus
	if cfgHome := os.Getenv(XDG_CONFIG_HOME); cfgHome != "" {
		cfgPath = path.Join(cfgHome, ".config", "astatus", "astatus")
	}

	// config path is specified by command line argument
	flag.Parse()
	if *cfgPathOption != "" {
		cfgPath = *cfgPathOption
	}

	// read commands from config
	f, err := os.Open(cfgPath)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		commands = append(commands, scanner.Text())
	}
}

func main() {
	statusItems := []StatusItem{}
	for range commands {
		statusItems = append(statusItems, NewStatusItem())
	}

	encoder := json.NewEncoder(os.Stdout)
	updates := GetUpdates(commands)
	hasUpd := false
	if _, err := os.Stdout.WriteString(I3BAR_PROTO_HEADER); err != nil {
		panic(err)
	}
	for {
		select {
		case upd := <-updates:
			hasUpd = true
			statusItems[len(commands)-1-upd.idx].FullText = upd.data
		case <-time.After(1 * time.Second):
			if hasUpd {
				if err := encoder.Encode(statusItems); err != nil {
					panic(err)
				}
				if _, err := os.Stdout.WriteString(",\n"); err != nil {
					panic(err)
				}
				hasUpd = false
			}
		}
	}

}
