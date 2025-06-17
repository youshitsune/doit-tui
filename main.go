package main

import (
	"fmt"
	"os"
	"path"
	"strings"

	list "github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/go-zoox/fetch"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
)

var style = lipgloss.NewStyle().Padding(3, 3)
var styleTasks = lipgloss.NewStyle().PaddingLeft(3)

var conf = koanf.New(".")
var auth *fetch.Config
var mainErr = error(nil)

type config struct {
	url      string
	username string
	password string
}

const (
	Home int = iota
	Add
	AddTag
	Rename
	Note
	EditTag
)

var cfgPath = path.Join(os.ExpandEnv("$XDG_CONFIG_HOME"), "/doit/config.yaml")
var cfg config

type Task struct {
	id     string
	title  string
	status string
}

func (t Task) FilterValue() string { return t.title}
func (t Task) Title() string       { return t.title }
func (t Task) Description() string {
	if t.status == "true" {
		return "\nFinished!"
	}
	return "\nNot started!"
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
	res, err := fetch.Get(cfg.url+"/list", auth)

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
	auth.Query = map[string]string{
		"task": task,
	}

	res, err := fetch.Post(cfg.url+"/new", auth)

	if err != nil {
		mainErr = err
	}

	if res.StatusCode() != 200 {
		mainErr = &Err{"Error while adding task"}
	}
}

func change(id string) {
	auth.Query = map[string]string{"id": id}
	res, err := fetch.Post(cfg.url+"/change", auth)

	if err != nil {
		mainErr = err
	}

	if res.StatusCode() != 200 {
		mainErr = &Err{"Error while marking task finished"}
	}
}


func deleteTask(id string) {
	auth.Query = map[string]string{"id": id}
	res, err := fetch.Post(cfg.url+"/delete", auth)

	if err != nil {
		mainErr = err
	}

	if res.StatusCode() != 200 {
		mainErr = &Err{"Error while deleting task"}
	}
}

func rename(id, task string) {
	auth.Query = map[string]string{
		"id":   id,
		"task": task,
	}
	res, err := fetch.Post(cfg.url+"/rename", auth)

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
	note     textarea.Model
	mode     int
	selected Task
	new      string
}

func initialModel() model {
	input := textinput.New()
	input.Focus()
	input.CharLimit = 256

	itemDelegate := list.NewDefaultDelegate()
	itemDelegate.SetHeight(3)
	itemDelegate.Styles.NormalTitle.Bold(true)
	itemDelegate.Styles.SelectedTitle.Bold(true)
	itemDelegate.Styles.DimmedTitle.Bold(true)
	listTasks := list.New(list_tasks(), itemDelegate, 0, 0)
	listTasks.Title = "Tasks"
	listTasks.Styles.TitleBar.Align(3, 3)
	listTasks.SetShowHelp(false)
	listTasks.DisableQuitKeybindings()

	ta := textarea.New()
	ta.Focus()

	return model{
		tasks: listTasks,
		mode:  Home,
		input: input,
		note:  ta,
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(textinput.Blink, textarea.Blink)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if mainErr != nil {
		return m, tea.Quit
	}

	var cmds []tea.Cmd

	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.tasks.SetSize(msg.Width, msg.Height)
		m.input.Width = msg.Width

	case tea.KeyMsg:
		if m.mode == Home {
			if m.tasks.FilterState() == list.Filtering {
				break
			}
			switch msg.String() {
			case "enter":
				id := m.tasks.SelectedItem().(Task).id
				change(id)
				cmd := m.tasks.SetItems(list_tasks())
				cmds = append(cmds, cmd)
			case "ctrl+c", "q":
				return m, tea.Quit
			case "d":
				id := m.tasks.SelectedItem().(Task).id
				deleteTask(id)
				cmd := m.tasks.SetItems(list_tasks())
				cmds = append(cmds, cmd)
			case "a":
				m.mode = Add
				m.input.SetValue("")
			case "r":
				m.selected = m.tasks.SelectedItem().(Task)
				m.mode = Rename
				m.input.SetValue(m.selected.title)
			}
		} else if m.mode == Add {
			var cmd tea.Cmd
			switch msg.String() {
			case "enter":
				m.new = m.input.Value()
				add(m.new)
				cmd = m.tasks.SetItems(list_tasks())
				cmds = append(cmds, cmd)
				m.mode = Home
			case "ctrl+c":
				return m, tea.Quit
			case "esc":
				m.new = ""
				m.mode = Home
			}
			m.input, cmd = m.input.Update(msg)
			cmds = append(cmds, cmd)
		}  else if m.mode == Rename {
			var cmd tea.Cmd
			switch msg.String() {
			case "enter":
				rename(m.selected.id, m.input.Value())
				m.selected = Task{}
				cmd = m.tasks.SetItems(list_tasks())
				cmds = append(cmds, cmd)
				m.mode = Home
			case "ctrl+c":
				return m, tea.Quit
			case "esc":
				m.mode = Home
			}
			m.input, cmd = m.input.Update(msg)
			cmds = append(cmds, cmd)
		} 	
	}
	var cmd tea.Cmd
	m.tasks, cmd = m.tasks.Update(msg)
	cmds = append(cmds, cmd)
	return m, tea.Batch(cmds...)
}

func (m model) View() string {
	if m.mode == Add {
		return style.Render("Name of the task:\n\n" + m.input.View())
	} else if m.mode == Rename {
		return style.Render("Rename the task:\n\n" + m.input.View())
	} 	
	return styleTasks.Render(m.tasks.View())
}

func main() {
	cfg = load_config()
	auth = &fetch.Config{
		Username: cfg.username,
		Password: cfg.password,
	}
	p := tea.NewProgram(initialModel())
	if _, err := p.Run(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	if mainErr != nil {
		fmt.Println(mainErr)
	}
}
