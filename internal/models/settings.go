package models

type Font struct {
	Family       string `json:"family,omitempty" yaml:"family,omitempty"`
	Path         string `json:"-" yaml:"-"`
	IsFixedWidth bool   `json:"isFixedWidth,omitempty" yaml:"-"`
}

type Settings struct {
	Window     WindowSettings     `json:"window" yaml:"window"`
	General    GeneralSettings    `json:"general" yaml:"general"`
	Editor     EditorSettings     `json:"editor" yaml:"editor"`
	Terminal   TerminalSettings   `json:"terminal" yaml:"terminal"`
	Workspaces WorkspacesSettings `json:"workspaces" yaml:"workspaces"`
	Updates    UpdatesSettings    `json:"updates" yaml:"updates"`
	Logging    LoggingSettings    `json:"logging" yaml:"logging"`
}

type WorkspacesSettings struct {
	FileExtensions []string `json:"fileExtensions" yaml:"fileExtensions"`
}

type UpdatesSettings struct {
	Frequency        string `json:"frequency" yaml:"frequency"`
	LastCheckedAt    string `json:"lastCheckedAt" yaml:"lastCheckedAt,omitempty"`
	DismissedVersion string `json:"dismissedVersion" yaml:"dismissedVersion,omitempty"`
}

type WindowSettings struct {
	Width      int  `json:"width" yaml:"width"`
	Height     int  `json:"height" yaml:"height"`
	AsideWidth int  `json:"asideWidth" yaml:"asideWidth"`
	Maximized  bool `json:"maximized" yaml:"maximized"`
	PositionX  int  `json:"positionX" yaml:"positionX"`
	PositionY  int  `json:"positionY" yaml:"positionY"`
}

type GeneralSettings struct {
	Theme    string       `json:"theme" yaml:"theme"`
	Language string       `json:"language" yaml:"language"`
	Font     FontSettings `json:"font" yaml:"font,omitempty"`
}

type FontSettings struct {
	Family string `json:"family" yaml:"family,omitempty"`
	Size   int    `json:"size" yaml:"size"`
	Name   string `json:"name" yaml:"name,omitempty"`
}

type EditorSettings struct {
	LineNumbers bool         `json:"lineNumbers" yaml:"lineNumbers"`
	Font        FontSettings `json:"font" yaml:"font,omitempty"`
	ShowFolding bool         `json:"showFolding" yaml:"showFolding"`
	DropText    bool         `json:"dropText" yaml:"dropText"`
	Links       bool         `json:"links" yaml:"links"`
	QueryEngine string       `json:"queryEngine" yaml:"queryEngine"`
}

type TerminalSettings struct {
	Font        FontSettings `json:"font" yaml:"font"`
	CursorStyle string       `json:"cursorStyle" yaml:"cursorStyle,omitempty"`
}

type LoggingSettings struct {
	Level          string `json:"level" yaml:"level,omitempty"`
	ConsoleEnabled bool   `json:"consoleEnabled" yaml:"consoleEnabled"`
	FileEnabled    bool   `json:"fileEnabled" yaml:"fileEnabled"`
	MaxSizeMB      int    `json:"maxSizeMB" yaml:"maxSizeMB"`
	MaxBackups     int    `json:"maxBackups" yaml:"maxBackups"`
}

// Normalize clamps loaded values into safe ranges and coerces an unknown
// Level to "info". Callers should invoke this after unmarshalling settings
// from disk, so corrupt or hand-edited YAML cannot propagate into the runtime.
func (s *LoggingSettings) Normalize() {
	switch s.Level {
	case "debug", "info", "warn", "warning", "error":
	case "":
		s.Level = "info"
	default:
		s.Level = "info"
	}
	if s.MaxSizeMB < 1 {
		s.MaxSizeMB = 1
	}
	if s.MaxBackups < 0 {
		s.MaxBackups = 0
	}
}

type WindowState struct {
	Width  int `json:"width"`
	Height int `json:"height"`
	X      int `json:"x"`
	Y      int `json:"y"`
}
