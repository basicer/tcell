package tcell

import (
	"errors"
	"os"
)

// ErrWinSizeUnused is for TermDrivers to signal to use the default platform
// window size lookup method
var ErrWinSizeUnused = errors.New("driver does not provide WinSize")

// TermDriver allows you to customize the TTY used by Screen,
// most notably to support a PTY pair that can be used with SSH servers.
type TermDriver interface {
	// Init sets up two file TTY/PTY file descriptors, which may be the same
	// in some cases. It also takes a chan that is used to notify the Screen
	// refresh the window size.
	Init(winch chan os.Signal) (in *os.File, out *os.File, err error)

	// WinSize returns the current window width and height. It can also return
	// ErrWinSizeUnused to tell Screen to use platform syscalls to get the
	// window size from the out file descriptor.
	WinSize() (width int, height int, err error)
	GetTerm() string
	Engage()
	Disengage()
}

// defaultTermDriver is what's used when you don't specify a custom TermDriver
type defaultTermDriver struct {
	winch chan os.Signal
	out   *os.File
}

func (d *defaultTermDriver) Init(winch chan os.Signal) (in *os.File, out *os.File, err error) {
	in, err = os.OpenFile("/dev/tty", os.O_RDONLY, 0)
	if err != nil {
		return
	}
	out, err = os.OpenFile("/dev/tty", os.O_WRONLY, 0)
	if err != nil {
		return
	}

	d.winch = winch
	d.out = out
	d.Engage()
	return
}
func (d *defaultTermDriver) GetTerm() string {
	return os.Getenv("TERM")
}

func (d *defaultTermDriver) WinSize() (int, int, error) {
	return 0, 0, ErrWinSizeUnused
}
