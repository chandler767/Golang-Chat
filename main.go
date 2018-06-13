package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/jroimartin/gocui"
	"github.com/pubnub/go/messaging"
)

func drawchat(channel string, username string) {
	// Initialize PubNub
	pubnub := messaging.NewPubnub("pub-c-73b26911-73e7-4dbe-baae-9faa04cc669e", "sub-c-1c26bb82-6e95-11e8-967c-82814fd59ac3", "", "", false, "", nil)

	// Create a new GUI.
	g, err := gocui.NewGui(gocui.OutputNormal)
	if err != nil {
		panic(err)
		return
	}
	defer g.Close()
	g.Cursor = true

	// Update the views when terminal changes size.
	g.SetManagerFunc(func(g *gocui.Gui) error {
		termwidth, termheight := g.Size()
		_, err := g.SetView("output", 0, 0, termwidth-1, termheight-4)
		if err != nil {
			return err
		}
		_, err = g.SetView("input", 0, termheight-3, termwidth-1, termheight-1)
		if err != nil {
			return err
		}
		return nil
	})

	// Subscribe (listen on) a channel.
	successChannel := make(chan []byte)
	go pubnub.Subscribe(channel, "", successChannel, false, make(chan []byte))
	go func() {
		for {
			select {
			case response := <-successChannel:
				var msg []interface{}
				err := json.Unmarshal(response, &msg)
				if err != nil {
					log.Println(err)
					return
				}
				ov, err := g.View("output")
				if err != nil {
					log.Println("Cannot get output view:", err)
					return
				}
				switch m := msg[0].(type) {
				case []interface{}:
					// Get output view and print.
					_, err = fmt.Fprintf(ov, "%s", m[0])
					if err != nil {
						log.Println("Cannot print to output view:", err)
					}

				}
				// Refresh view
				g.Update(func(g *gocui.Gui) error {
					return nil
				})
			}
		}
	}()

	// Terminal width and height.
	termwidth, termheight := g.Size()

	// Output.
	ov, err := g.SetView("output", 0, 0, termwidth-1, termheight-4)
	if err != nil && err != gocui.ErrUnknownView {
		log.Println("Failed to create output view:", err)
		return
	}
	ov.Title = " Messages  -  <" + channel + "> "
	ov.FgColor = gocui.ColorRed
	ov.Autoscroll = true
	ov.Wrap = true

	// Send a welcome message.
	_, err = fmt.Fprintln(ov, "<Go-Chat>: Welcome to Go-Chat powered by PubNub!")
	if err != nil {
		log.Println("Failed to print into output view:", err)
	}
	_, err = fmt.Fprintln(ov, "<Go-Chat>: Press Ctrl-C to quit.")
	if err != nil {
		log.Println("Failed to print into output view:", err)
	}

	// Input.
	iv, err := g.SetView("input", 0, termheight-3, termwidth-1, termheight-1)
	if err != nil && err != gocui.ErrUnknownView {
		log.Println("Failed to create input view:", err)
		return
	}
	iv.Title = " New Message  -  <" + username + "> "
	iv.FgColor = gocui.ColorWhite
	iv.Editable = true
	err = iv.SetCursor(0, 0)
	if err != nil {
		log.Println("Failed to set cursor:", err)
		return
	}

	// Bind Ctrl-C so the user can quit.
	err = g.SetKeybinding("", gocui.KeyCtrlC, gocui.ModNone, func(g *gocui.Gui, v *gocui.View) error {
		return gocui.ErrQuit
	})
	if err != nil {
		log.Println("Could not set key binding:", err)
		return
	}

	// Bind enter key to input to send new messages.
	err = g.SetKeybinding("input", gocui.KeyEnter, gocui.ModNone, func(g *gocui.Gui, iv *gocui.View) error {
		// Read buffer from the beginning.
		iv.Rewind()

		// Send message if text was entered.
		if len(iv.Buffer()) >= 2 {
			go pubnub.Publish(
				channel,
				"<"+username+">: "+iv.Buffer(),
				make(chan []byte),
				make(chan []byte),
			)

			// Reset input.
			iv.Clear()

			// Reset cursor.
			err = iv.SetCursor(0, 0)
			if err != nil {
				log.Println("Failed to set cursor:", err)
			}
			return err
		}
		return nil
	})
	if err != nil {
		log.Println("Cannot bind the enter key:", err)
	}

	// Set the focus to input.
	_, err = g.SetCurrentView("input")
	if err != nil {
		log.Println("Cannot set focus to input view:", err)
	}

	// Start the main loop.
	err = g.MainLoop()
	log.Println("Main loop has finished:", err)
}

func main() {
	// Print version info.
	fmt.Println("PubNub SDK for go;", messaging.VersionInfo())
	// Get channel and username.
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Enter Channel Name: ")
	channel, err := reader.ReadString('\n')
	if err != nil {
		log.Println("Could not set channel:", err)
	}
	fmt.Print("Enter Desired Username: ")
	username, err := reader.ReadString('\n')
	if err != nil {
		log.Println("Could not set username:", err)
	}
	// Create the GUI.
	drawchat(strings.TrimSuffix(channel, "\n"), strings.TrimSuffix(username, "\n"))
}
