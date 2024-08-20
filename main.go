package main

import (
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/go-zoox/fetch"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
)

var conf = koanf.New(".")

type config struct {
	url      string
	username string
	password string
}

var cfg = load_config()

func load_config() config {
	config_path := path.Join(os.ExpandEnv("$XDG_CONFIG_HOME"), "/doit/config.yaml")
	if err := conf.Load(file.Provider(config_path), yaml.Parser()); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	cfg := config{}
	cfg.url = conf.String("protocol") + "://" + conf.String("url") + ":" + conf.String("port")
	cfg.password = conf.String("password")
	cfg.username = conf.String("username")
	return cfg
}

func list() [][]string {
	res, err := fetch.Post(cfg.url+"/list", &fetch.Config{
		Query: map[string]string{
			"user":     cfg.username,
			"password": cfg.password,
		},
	})

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	r := make([][]string, 0)
	t := strings.Split(string(res.Body), "\n")
	for i := range t {
		tmp := strings.Split(t[i], "``")
		if len(tmp) == 3 {
			r = append(r, tmp)
		}
	}

	return r
}

func add(task string) bool {
	res, err := fetch.Post(cfg.url+"/new", &fetch.Config{
		Query: map[string]string{
			"user":     cfg.username,
			"password": cfg.password,
			"task":     task,
		},
	})

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	if res.StatusCode() == 200 {
		return true
	} else {
		return false
	}
}

func done(id string) bool {
	res, err := fetch.Post(cfg.url+"/done", &fetch.Config{
		Query: map[string]string{
			"user":     cfg.username,
			"password": cfg.password,
			"id":       id,
		},
	})

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	if res.StatusCode() == 200 {
		return true
	} else {
		return false
	}
}

func deleteTask(id string) bool {
	res, err := fetch.Post(cfg.url+"/delete", &fetch.Config{
		Query: map[string]string{
			"user":     cfg.username,
			"password": cfg.password,
			"id":       id,
		},
	})

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	if res.StatusCode() == 200 {
		return true
	} else {
		return false
	}
}

type model struct {
	tasks    [][]string
	cursor   int
	selected map[int]struct{}
	update   bool
	status   int // STATES: -1 - no status; 0 - failed status; 1 - successful status
	input    textinput.Model
	mode     int // STATES: 0 - change of state; 1 - delete tasks; 2 - add tasks
}

func initialModel() model {
	input := textinput.New()
	input.Focus()
	input.CharLimit = 256
	input.Width = 20
	return model{
		tasks:    list(),
		selected: make(map[int]struct{}),
		mode:     0,
		update:   false,
		input:    input,
		status:   -1,
	}
}

func (m model) Init() tea.Cmd {
	return textinput.Blink
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.mode == 0 {
			switch msg.String() {
			case "up", "k":
				if m.cursor > 0 {
					m.cursor--
				}
			case "down", "j":
				if m.cursor < len(m.tasks)-1 {
					m.cursor++
				}
			case "y":
				m.update = true
			case "enter", " ":
				_, ok := m.selected[m.cursor]
				if ok {
					delete(m.selected, m.cursor)
				} else {
					m.selected[m.cursor] = struct{}{}
				}
			}
		} else if m.mode == 1 {
			var cmd tea.Cmd
			switch msg.String() {
			case "enter":
				m.update = true
			case "ctrl+c":
				return m, tea.Quit
			case "ctrl+h":
				m.mode = 0
			case "ctrl+d":
				m.mode = 2
			}
			m.input, cmd = m.input.Update(msg)
			return m, cmd
		} else {
			switch msg.String() {
			case "up", "k":
				if m.cursor > 0 {
					m.cursor--
				}
			case "down", "j":
				if m.cursor < len(m.tasks)-1 {
					m.cursor++
				}
			case "y":
				m.update = true
			case "enter", " ":
				_, ok := m.selected[m.cursor]
				if ok {
					delete(m.selected, m.cursor)
				} else {
					m.selected[m.cursor] = struct{}{}
				}
			}
		}
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "ctrl+h":
			m.mode = 0
		case "ctrl+a":
			m.mode = 1
		case "ctrl+d":
			m.mode = 2
		}

	}

	if m.update {
		switch m.mode {
		case 0:
			for i := range m.tasks {
				if _, ok := m.selected[i]; ok {
					if done(m.tasks[i][0]) {
						m.status = 1
					} else {
						m.status = 0
					}
					delete(m.selected, i)
				}
			}
			m.tasks = list()
		case 1:
			if add(m.input.Value()) {
				m.status = 1
			} else {
				m.status = 0
			}
			m.input.SetValue("")
		case 2:
			for i := range m.tasks {
				if _, ok := m.selected[i]; ok {
					if deleteTask(m.tasks[i][0]) {
						m.status = 1
					} else {
						m.status = 0
					}
					delete(m.selected, i)
				}
			}
		}
		m.update = false
	}
	return m, nil
}

func (m model) View() string {
	if m.mode == 0 {
		s := "Tasks:\n\n"

		for i := range m.tasks {
			cursor := " "
			if m.cursor == i {
				cursor = ">"
			}

			checked := " "
			_, ok := m.selected[i]
			if ok || m.tasks[i][2] == "true" {
				checked = "x"
			}

			s += fmt.Sprintf("%s [%s] %s\n", cursor, checked, m.tasks[i][1])
			if m.status == 0 {
				s += "\nFailed!\n"
			} else if m.status == 1 {
				s += "\nSuccess!\n"
			}

			m.status = -1
		}
		return s
	} else if m.mode == 1 {
		s := "Type in the name of the task:\n\n" + m.input.View()
		if m.status == 0 {
			s += "\nFailed!\n"
		} else if m.status == 1 {
			s += "\nSuccess!\n"
		}
		m.status = -1
		return s

	} else {
		s := "Select tasks for deleting:\n\n"

		for i := range m.tasks {
			cursor := " "
			if m.cursor == i {
				cursor = ">"
			}

			checked := " "
			if _, ok := m.selected[i]; ok {
				checked = "x"
			}

			s += fmt.Sprintf("%s [%s] %s\n", cursor, checked, m.tasks[i][1])
			if m.status == 0 {
				s += "\nFailed!\n"
			} else if m.status == 1 {
				s += "\nSuccess!\n"

			}

			m.status = -1
		}
		return s
	}
}

func main() {
	p := tea.NewProgram(initialModel())
	if _, err := p.Run(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
