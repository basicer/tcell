// Copyright 2021 The TCell Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use file except in compliance with the License.
// You may obtain a copy of the license at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// +build aix darwin dragonfly freebsd linux netbsd openbsd solaris zos

package tcell

import (
	"errors"
	"golang.org/x/term"
	"os/signal"
	"syscall"
)

// engage is used to place the terminal in raw mode and establish screen size, etc.
// Thing of this is as tcell "engaging" the clutch, as it's going to be driving the
// terminal interface.
func (t *tScreen) engage() error {
	t.Lock()
	defer t.Unlock()
	if t.stopQ != nil {
		return errors.New("already engaged")
	}
	if _, err := term.MakeRaw(int(t.in.Fd())); err != nil {
		return err
	}
	if w, h, err := t.getWinSize(); err == nil && w != 0 && h != 0 {
		t.cells.Resize(w, h)
	}
	stopQ := make(chan struct{})
	t.stopQ = stopQ
	t.nonBlocking(false)
	t.enableMouse(t.mouseFlags)
	t.enablePasting(t.pasteEnabled)
	t.driver.Engage()

	ti := t.ti
	t.TPuts(ti.EnterCA)
	t.TPuts(ti.EnterKeypad)
	t.TPuts(ti.HideCursor)
	t.TPuts(ti.EnableAcs)
	t.TPuts(ti.Clear)

	t.wg.Add(2)
	go t.inputLoop(stopQ)
	go t.mainLoop(stopQ)
	return nil
}

// disengage is used to release the terminal back to support from the caller.
// Think of this as tcell disengaging the clutch, so that another application
// can take over the terminal interface.  This restores the TTY mode that was
// present when the application was first started.
func (t *tScreen) disengage() {

	t.Lock()
	t.nonBlocking(true)
	stopQ := t.stopQ
	t.stopQ = nil
	close(stopQ)
	t.Unlock()

	// wait for everything to shut down
	t.wg.Wait()

	t.driver.Disengage()

	// put back normal blocking mode
	t.nonBlocking(false)

	// shutdown the screen and disable special modes (e.g. mouse and bracketed paste)
	ti := t.ti
	t.cells.Resize(0, 0)
	t.TPuts(ti.ShowCursor)
	t.TPuts(ti.AttrOff)
	t.TPuts(ti.Clear)
	t.TPuts(ti.ExitCA)
	t.TPuts(ti.ExitKeypad)
	t.enableMouse(0)
	t.enablePasting(false)

	// restore the termios that we were started with
	_ = term.Restore(int(t.in.Fd()), t.saved)

}

// initialize is used at application startup, and sets up the initial values
// including file descriptors used for terminals and saving the initial state
// so that it can be restored when the application terminates.
func (t *tScreen) initialize() error {
	var err error
	if t.in, t.out, err = t.driver.Init(t.sigwinch); err != nil {
		return err
	}

	t.saved, err = term.GetState(int(t.in.Fd()))
	if err == nil {
		return nil
	}
	return nil
}

// finalize is used to at application shutdown, and restores the terminal
// to it's initial state.  It should not be called more than once.
func (t *tScreen) finalize() {

	t.disengage()
}

// getWinSize is called to obtain the terminal dimensions.
func (t *tScreen) getWinSize() (int, int, error) {
	if w, h, err := t.driver.WinSize(); err != ErrWinSizeUnused {
		return w, h, err
	}
	return term.GetSize(int(t.in.Fd()))
}

// Beep emits a beep to the terminal.
func (t *tScreen) Beep() error {
	t.writeString(string(byte(7)))
	return nil
}

func (d *defaultTermDriver) Engage() {
	signal.Notify(d.winch, syscall.SIGWINCH)
}

func (d *defaultTermDriver) Disengage() {
	signal.Stop(d.winch)
}
