package main

import (
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/jroimartin/gocui"
	"github.com/urfave/cli/v2"
)

var menuItems = []string{
	Archon.String(),
	//Patcher.String(),
	//GenerateServerCertificates.String(),
	//RunPacketAnalyzer.String(),
	//RunAccountTool.String(),
}

func menu(cc *cli.Context) error {
	g, err := gocui.NewGui(gocui.OutputNormal)
	if err != nil {
		return err
	}
	defer g.Close()

	g.Cursor = true

	g.SetManagerFunc(layout)

	if err := keybindings(cc, g); err != nil {
		return err
	}

	if err := g.MainLoop(); err != nil && err != gocui.ErrQuit {
		return err
	}

	return nil
}

func quit(_ *gocui.Gui, _ *gocui.View) error {
	return gocui.ErrQuit
}

func layout(g *gocui.Gui) error {
	maxX, maxY := g.Size()

	if v, err := g.SetView("view", 0, 0, 30, maxY-1); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}

		v.Highlight = true
		v.SelBgColor = gocui.ColorGreen
		v.SelFgColor = gocui.ColorBlack

		for _, text := range menuItems {
			fmt.Fprintln(v, text)
		}

		if _, err := g.SetCurrentView("view"); err != nil {
			return err
		}
	}
	if v, err := g.SetView("main", 30, 0, maxX-1, maxY-1); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		v.Wrap = true
	}
	return nil
}

func cursorDown(_ *gocui.Gui, v *gocui.View) error {
	if v != nil {
		cx, cy := v.Cursor()

		cy += 1
		if cy >= len(menuItems) {
			cy = 0
		}

		if err := v.SetCursor(cx, cy); err != nil {
			return err
		}
	}
	return nil
}

func cursorUp(_ *gocui.Gui, v *gocui.View) error {
	if v != nil {
		cx, cy := v.Cursor()

		cy -= 1
		if cy < 0 {
			cy = len(menuItems) - 1
		}

		if err := v.SetCursor(cx, cy); err != nil {
			return err
		}
	}
	return nil
}

func handleSelection(cc *cli.Context) func(_ *gocui.Gui, v *gocui.View) error {
	return func(g *gocui.Gui, v *gocui.View) error {
		_, y := v.Cursor()

		cmdName := Command(y).String()
		if cmdName == "" {
			return fmt.Errorf("unknown command chosen")
		}

		command := cc.App.Command(cmdName)
		if command == nil {
			return fmt.Errorf("command not found [cmd: %s]", cmdName)
		}

		v, err := g.View("main")
		if err != nil {
			return err
		}

		reader, writer, err := os.Pipe()
		if err != nil {
			return err
		}
		os.Stdout = writer

		go func() {
			var wg sync.WaitGroup
			wg.Add(1)
			go func() {
				defer wg.Done()
				if err := command.Run(cc); err != nil {
					fmt.Fprintln(v, err.Error())
				}
			}()

			for {
				buf := make([]byte, 1024)
				n, err := reader.Read(buf)
				if err != nil {
					if err == io.EOF {
						break
					}
					fmt.Fprintf(v, "reader err: %v", err)
					return
				}
				fmt.Fprintln(v, string(buf[:n]))

				g.Update(func(gui *gocui.Gui) error {
					maxX, maxY := g.Size()
					_, err := gui.SetView("main", 30, 0, maxX-1, maxY-1)
					if err != nil && err != gocui.ErrUnknownView {
						return err
					}
					return nil
				})
			}

			wg.Wait()
		}()

		return nil
	}
}

func keybindings(cc *cli.Context, g *gocui.Gui) error {
	if err := g.SetKeybinding("view", gocui.KeyArrowDown, gocui.ModNone, cursorDown); err != nil {
		return err
	}
	if err := g.SetKeybinding("view", gocui.KeyArrowUp, gocui.ModNone, cursorUp); err != nil {
		return err
	}
	if err := g.SetKeybinding("view", gocui.KeyEnter, gocui.ModNone, handleSelection(cc)); err != nil {
		return err
	}
	if err := g.SetKeybinding("", gocui.KeyCtrlC, gocui.ModNone, quit); err != nil {
		return err
	}
	return nil
}
