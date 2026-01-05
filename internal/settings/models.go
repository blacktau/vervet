package settings

const DefaultFontSize = 14
const DefaultWindowWidth = 1024
const DefaultWindowHeight = 768
const DefaultAsideWidth = 300

type Settings struct {
	Window   WindowSettings   `json:"window" yaml:"window"`
	General  GeneralSettings  `json:"general" yaml:"general"`
	Editor   EditorSettings   `json:"editor" yaml:"editor"`
	Terminal TerminalSettings `json:"terminal" yaml:"terminal"`
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
}

type TerminalSettings struct {
	Font        FontSettings `json:"font" yaml:"font"`
	CursorStyle string       `json:"cursorStyle" yaml:"cursorStyle,omitempty"`
}

type Font struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

type WindowState struct {
	Width  int `json:"width"`
	Height int `json:"height"`
	X      int `json:"x"`
	Y      int `json:"y"`
}
