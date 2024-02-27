# FANSITERM

Fake (virtual) ANSI TERMinal. 

Fansiterm is a golang package for implementing a partially compatible ANSI terminal, rendered to an image.Image (really, a golang.org/x/image/draw.Image). This is suitable for the graphical backend of a virtual terminal emulator.

The intent is for implementing a terminal on micro controllers connected to graphical displays. This provides an easy way to make a TUI for hte micro controller and take advantage of packages like github.com/charmbracelet/bubbletea or making a simple dumb terminal-type device.

# Overview

The (*fansiterm).Device object implements image.Image/draw.Image and io.Writer. To push data (text) to the terminal, you simply call Write() against the Device object.

The text isn't buffered anywhere, if you need the text or want to implement more advanced features like scrolling, that's up to whatever is writing to (*fansiterm).Device. Since this is meant to.

If you want to push your own graphics or other operations, you can draw directly to the Device object as well, as it implements draw.Image.

If Device is initialized with a nil image buffer, it allocates its own buffer. Otherwise, you can pass a draw.Image object (like what the driver for an OLED or TFT screen provides you) to it and any Write()s to the (*fansiterm).Device will be immediately rendered to the backing screen. Whether the screen buffers image data and needs to be manually blitted is screen driver dependant. So 

# Features

 - Cursor styles: Block, Beam, Underscore
 - Bell is supported: a callback is provided for when the terminal receives a \a (bell character). So you could trigger a beep via a speaker and PWM or blink an LED or blink the backlight, etc.
 - Standard cursor manipulation supported,
 - Regular and Bold Font
 - Underline, Double Underline, Strikethrough
 - Custom Tile loading for alternate character set (shift-out character set, commonly used for line-drawing/pseudo graphics)
 	

# Non-Features

The main purpose of this package is for use on rather low-power microcontrollers, so some standard features for terminal emulators are not implemented.

	- Blinking text and blink cursors
		- this would require a some kind of timer-callback
	- Resizable Text
		- Right now, the pre-rendered inconsolata.Regular8x16 is used.
	- Color Pallet / 256-color
		- The standard 4-bit color pallet is supported and 24-bit True Color is supported

# TODO

 - Test on real hardware
 - 1-bit color/rendering support for very-very-constrained systems

# Future

I want to keep a very stripped down, barebones version of fansiterm that will work on very resource constrained microcontrollers. Not having blinking However, I'm very open to having a more featureful v2 that is suitable for using as a backend for a desktop terminal emulator.