package fansiterm

// Config defines runtime settings for a Device.
type Config struct {
	LocalEcho                bool
	TabSize                  int  // Number of spaces per tab.
	StrikethroughHeight      int  // Pixel height offset for strike-through.
	CursorStyle              int  // Default cursor style.
	BoldColors               bool // Whether bold colors are enabled.
	AltScreen                bool // Enable alternate screen buffer (expensive on MCUs).
	Wraparound               bool // Whether text wraps at the screen edge.
	CursorKeyApplicationMode bool // Enable application mode for cursor keys.
	MouseEvents              int  // 0, 1000, 10002, or 1003
	MouseSGR                 bool // if false, use \e[Mcbxbyb reporting; else use \e[<

	// Miscellaneous properties, like "Window Title"
	Properties map[Property]string
}

// ConfigDefault provides the default configuration values for a Device.
var ConfigDefault = Config{
	TabSize:             8,
	StrikethroughHeight: 7,
	BoldColors:          true,
}

func NewConfig() Config {
	conf := ConfigDefault
	conf.Properties = make(map[Property]string)
	return conf
}

func (d *Device) configChange() {
	if d.ConfigUpdate != nil {
		d.ConfigUpdate(d.Config)
	}
}
