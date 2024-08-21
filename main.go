package main

import (
	"fmt"
	"os"
	"path"
	"strings"

	list "github.com/charmbracelet/bubbles/list"
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

type Task struct {
	id     string
	title  string
	status string
}

func (t Task) FilterValue() string { return t.title }
func (t Task) Title() string       { return t.title }
func (t Task) Description() string {
	if t.status == "true" {
		return "Finished!"
	}
	return "Not started!"
}

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

func list_tasks() []list.Item {
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
	r := make([]list.Item, 0)
	t := strings.Split(string(res.Body), "\n")
	for i := range t {
		tmp := strings.Split(t[i], "``")
		if len(tmp) == 3 {
			r = append(r, Task{id: tmp[0], title: tmp[1], status: tmp[2]})
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
	tasks list.Model
	input textinput.Model
	mode  int // STATES: 0 - home; 1 - add task
}

func initialModel() model {
	input := textinput.New()
	input.Focus()
	input.CharLimit = 256
	input.Width = 20

	listTasks := list.New(list_tasks(), list.NewDefaultDelegate(), 0, 0)
	listTasks.Title = "Tasks"
	listTasks.SetShowHelp(false)

	return model{
		tasks: listTasks,
		mode:  0,
		input: input,
	}
}

func (m model) Init() tea.Cmd {
	return textinput.Blink
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.tasks.SetSize(msg.Width, msg.Height)

	case tea.KeyMsg:
		if m.mode == 0 {
			if m.tasks.FilterState() == list.Filtering {
				var cmd tea.Cmd
				m.tasks, cmd = m.tasks.Update(msg)
				cmds = append(cmds, cmd)
				break
			}
			switch msg.String() {
			case "enter":
				id := m.tasks.SelectedItem().(Task).id
				done(id)
				cmd := m.tasks.SetItems(list_tasks())
				cmds = append(cmds, cmd)
			case "ctrl+c":
				return m, tea.Quit
			case "d":
				id := m.tasks.SelectedItem().(Task).id
				deleteTask(id)
				cmd := m.tasks.SetItems(list_tasks())
				cmds = append(cmds, cmd)
			case "a":
				m.mode = 1
			}
			var cmd tea.Cmd
			m.tasks, cmd = m.tasks.Update(msg)
			cmds = append(cmds, cmd)

		} else if m.mode == 1 {
			var cmd tea.Cmd
			switch msg.String() {
			case "enter":
				add(m.input.Value())
				cmd = m.tasks.SetItems(list_tasks())
				cmds = append(cmds, cmd)
				m.mode = 0
			case "ctrl+c":
				return m, tea.Quit
			case "esc":
				m.mode = 0
			}
			m.input, cmd = m.input.Update(msg)
			cmds = append(cmds, cmd)
		}
	}

	return m, tea.Batch(cmds...)
}

func (m model) View() string {
	if m.mode == 1 {
		return "Name of the task:\n\n" + m.input.View()
	}
	return m.tasks.View()
}

func main() {
	p := tea.NewProgram(initialModel())
	if _, err := p.Run(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
