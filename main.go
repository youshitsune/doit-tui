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
	tag    string
	title  string
	status string
}

func (t Task) FilterValue() string { return t.title + t.tag }
func (t Task) Title() string       { return t.title }
func (t Task) Description() string {
	if t.status == "true" {
		return t.tag + "\nFinished!"
	}
	return t.tag + "\nNot started!"
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
		if len(tmp) == 4 {
			r = append(r, Task{id: tmp[0], title: tmp[1], status: tmp[2], tag: tmp[3]})
		}
	}

	return r
}

func get_note(id string) string {
	res, err := fetch.Post(cfg.url+"/getnote", &fetch.Config{
		Query: map[string]string{
			"user":     cfg.username,
			"password": cfg.password,
			"id":       id,
		},
	})

	if err != nil {
		mainErr = err
		return ""
	}

	return string(res.Body)
}

func add(task, tag string) {
	res, err := fetch.Post(cfg.url+"/new", &fetch.Config{
		Query: map[string]string{
			"user":     cfg.username,
			"password": cfg.password,
			"task":     task,
			"tag":      tag,
		},
	})

	if err != nil {
		mainErr = err
	}

	if res.StatusCode() != 200 {
		mainErr = &Err{"Error while adding task"}
	}
}

func addnote(id, note string) {
	res, err := fetch.Post(cfg.url+"/newnote", &fetch.Config{
		Query: map[string]string{
			"user":     cfg.username,
			"password": cfg.password,
			"id":       id,
			"note":     note,
		},
	})

	if err != nil {
		mainErr = err
	}

	if res.StatusCode() != 200 {
		mainErr = &Err{"Error while adding note"}
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

func deleteNote(id string) {
	res, err := fetch.Post(cfg.url+"/deletenote", &fetch.Config{
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
		mainErr = &Err{"Error while deleting note"}
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

func editTag(id, tag string) {
	res, err := fetch.Post(cfg.url+"/edittag", &fetch.Config{
		Query: map[string]string{
			"user":     cfg.username,
			"password": cfg.password,
			"id":       id,
			"tag":      tag,
		},
	})

	if err != nil {
		mainErr = err
	}

	if res.StatusCode() != 200 {
		mainErr = &Err{"Eror while editing tag"}
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
				status := m.tasks.SelectedItem().(Task).status
				if status == "true" {
					reset(id)
				} else {
					done(id)
				}
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
			case "n":
				m.selected = m.tasks.SelectedItem().(Task)
				note := get_note(m.selected.id)
				if note == "" {
					m.note.SetValue("")
				} else {
					m.note.SetValue(note)
				}
				m.mode = Note
			case "t":
				m.selected = m.tasks.SelectedItem().(Task)
				m.mode = EditTag
				m.input.SetValue(m.selected.tag)
			}

		} else if m.mode == Add {
			var cmd tea.Cmd
			switch msg.String() {
			case "enter":
				m.new = m.input.Value()
				cmd = m.tasks.SetItems(list_tasks())
				cmds = append(cmds, cmd)
				m.mode = AddTag
				m.input.SetValue("")
			case "ctrl+c":
				return m, tea.Quit
			case "esc":
				m.new = ""
				m.mode = Home
			}
			m.input, cmd = m.input.Update(msg)
			cmds = append(cmds, cmd)
		} else if m.mode == AddTag {
			var cmd tea.Cmd
			switch msg.String() {
			case "enter":
				add(m.new, m.input.Value())
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
		} else if m.mode == Rename {
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
		} else if m.mode == Note {
			switch msg.String() {
			case "ctrl+s":
				addnote(m.selected.id, m.note.Value())
				m.selected = Task{}
				m.mode = Home
			case "ctrl+c":
				return m, tea.Quit
			case "esc":
				m.mode = Home
			case "ctrl+d":
				deleteNote(m.selected.id)
				m.mode = Home
			}
			var cmd tea.Cmd
			m.note, cmd = m.note.Update(msg)
			cmds = append(cmds, cmd)
		} else if m.mode == EditTag {
			switch msg.String() {
			case "enter":
				editTag(m.selected.id, m.input.Value())
				m.tasks.SetItems(list_tasks())
				m.selected = Task{}
				m.mode = Home
			case "ctrl+c":
				return m, tea.Quit
			case "esc":
				m.mode = Home
			}
			var cmd tea.Cmd
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
	} else if m.mode == AddTag {
		return style.Render("Tag of the task:\n\n" + m.input.View())
	} else if m.mode == Rename {
		return style.Render("Rename the task:\n\n" + m.input.View())
	} else if m.mode == Note {
		return style.Render("Note of the task:\n\n" + m.note.View())
	} else if m.mode == EditTag {
		return style.Render("Edit tag of the task:\n\n" + m.input.View())
	}
	return styleTasks.Render(m.tasks.View())
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
