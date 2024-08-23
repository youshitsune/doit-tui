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
var mainErr = error(nil)

type config struct {
	url      string
	username string
	password string
}

var cfgPath = path.Join(os.ExpandEnv("$XDG_CONFIG_HOME"), "/doit/config.yaml")
var cfg config

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

type Err struct {
	s string
}

func (e *Err) Error() string {
	return e.s
}

func load_config() config {
	if err := conf.Load(file.Provider(cfgPath), yaml.Parser()); err != nil {
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
		mainErr = err
		return nil
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

func add(task string) {
	res, err := fetch.Post(cfg.url+"/new", &fetch.Config{
		Query: map[string]string{
			"user":     cfg.username,
			"password": cfg.password,
			"task":     task,
		},
	})

	if err != nil {
		mainErr = err
	}

	if res.StatusCode() != 200 {
		mainErr = &Err{"Error while adding task"}
	}
}

func done(id string) {
	res, err := fetch.Post(cfg.url+"/done", &fetch.Config{
		Query: map[string]string{
			"user":     cfg.username,
			"password": cfg.password,
			"id":       id,
		},
	})

	if err != nil {
		mainErr = err
	}

	if res.StatusCode() != 200 {
		mainErr = &Err{"Error while marking task finished"}
	}
}

func reset(id string) {
	res, err := fetch.Post(cfg.url+"/reset", &fetch.Config{
		Query: map[string]string{
			"user":     cfg.username,
			"password": cfg.password,
			"id":       id,
		},
	})

	if err != nil {
		mainErr = err
	}

	if res.StatusCode() != 200 {
		mainErr = &Err{"Error while reseting task state"}
	}
}

func deleteTask(id string) {
	res, err := fetch.Post(cfg.url+"/delete", &fetch.Config{
		Query: map[string]string{
			"user":     cfg.username,
			"password": cfg.password,
			"id":       id,
		},
	})

	if err != nil {
		mainErr = err
	}

	if res.StatusCode() != 200 {
		mainErr = &Err{"Error while deleting task"}
	}
}

func rename(id, task string) {
	res, err := fetch.Post(cfg.url+"/rename", &fetch.Config{
		Query: map[string]string{
			"user":     cfg.username,
			"password": cfg.password,
			"id":       id,
			"task":     task,
		},
	})

	if err != nil {
		mainErr = err
	}

	if res.StatusCode() != 200 {
		mainErr = &Err{"Error while renaming task"}
	}
}

type model struct {
	tasks    list.Model
	input    textinput.Model
	mode     int  // STATES: 0 - home; 1 - add task; 2 - rename task
	selected Task // Only used for renaming
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
	if mainErr != nil {
		return m, tea.Quit
	}

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
				status := m.tasks.SelectedItem().(Task).status
				if status == "true" {
					reset(id)
				} else {
					done(id)
				}
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
			case "r":
				m.selected = m.tasks.SelectedItem().(Task)
				m.mode = 2
				m.input.SetValue(m.selected.title)
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
		} else if m.mode == 2 {
			var cmd tea.Cmd
			switch msg.String() {
			case "enter":
				rename(m.selected.id, m.input.Value())
				m.selected.id = ""
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
	} else if m.mode == 2 {
		return "Rename the task:\n\n" + m.input.View()
	}
	return m.tasks.View()
}

func main() {
	cfg = load_config()

	p := tea.NewProgram(initialModel())
	if _, err := p.Run(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	if mainErr != nil {
		fmt.Println(mainErr)
	}
}
