/*
   Copyright 2014 Franc[e]sco (lolisamurai@tfwno.gf)
   This file is part of osu! unban checker.
   osu! unban checker is free software: you can redistribute it and/or modify
   it under the terms of the GNU General Public License as published by
   the Free Software Foundation, either version 3 of the License, or
   (at your option) any later version.
   osu! unban checker is distributed in the hope that it will be useful,
   but WITHOUT ANY WARRANTY; without even the implied warranty of
   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
   GNU General Public License for more details.
   You should have received a copy of the GNU General Public License
   along with osu! unban checker. If not, see <http://www.gnu.org/licenses/>.
*/

package main

import (
	"github.com/google/gxui"
	"github.com/google/gxui/drivers/gl"
	"github.com/google/gxui/gxfont"
	"github.com/google/gxui/math"
	"github.com/google/gxui/themes/dark"

	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"
)

func appMain(driver gxui.Driver) {
	fmt.Println("Driver started")

	theme := dark.CreateTheme(driver)

	window := theme.CreateWindow(285, 100, "osu! unban checker")
	window.SetPadding(math.Spacing{L: 10, T: 10, R: 10, B: 10})
	window.OnClose(driver.Terminate)

	// name label, textbox and check button
	layout := theme.CreateLinearLayout()
	layout.SetSizeMode(gxui.ExpandToContent)
	layout.SetDirection(gxui.LeftToRight)
	layout.SetVerticalAlignment(gxui.AlignMiddle)

	label := theme.CreateLabel()
	label.SetText("Player id or name: ")
	layout.AddChild(label)

	textbox := theme.CreateTextBox()
	textbox.SetText("948713") // - Hakurei Reimu-
	layout.AddChild(textbox)

	button := theme.CreateButton()
	button.SetText("Check")
	layout.AddChild(button)

	// main layout
	mainlayout := theme.CreateLinearLayout()
	mainlayout.SetSizeMode(gxui.ExpandToContent)
	mainlayout.SetDirection(gxui.TopToBottom)

	mainlayout.AddChild(layout)

	label = theme.CreateLabel()
	label.SetText("Waiting for first refresh...")
	mainlayout.AddChild(label)

	checkbox := theme.CreateButton()
	checkbox.SetText("Show pop-up on unban")
	checkbox.OnClick(func(gxui.MouseEvent) { checkbox.SetChecked(!checkbox.IsChecked()) })
	mainlayout.AddChild(checkbox)

	window.AddChild(mainlayout)

	bigfont, err := driver.CreateFont(gxfont.Default, 50)
	if err != nil {
		panic(err)
	}

	var popup gxui.Window

	checkban := func() {
		status := "still banned"
		color := gxui.Red80
		player := textbox.Text()

		driver.Call(func() {
			label.SetText("Checking...")
			label.SetColor(gxui.Gray80)
		})

		for {
			url := fmt.Sprintf("https://osu.ppy.sh/u/%s", player)
			fmt.Printf("GET %s\n", url)

			resp, err := http.Get(url)
			if err != nil {
				fmt.Println(err)
				continue
			}

			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				fmt.Println(err)
				continue
			}

			// lol too lazy to parse html
			i := strings.Index(string(body[:]), "'s profile")
			if i != -1 {
				status = "unbanned"
				color = gxui.Green80

				// lol too lazy to use regex
				namestart := i
				for ; body[namestart] != '>'; namestart-- {
				}
				player = string(body[namestart+1 : i])

				// show pop-up if enabled
				driver.Call(func() {
					if checkbox.IsChecked() {
						if popup != nil {
							popup.Close()
						}

						popup = theme.CreateWindow(276, 90, "")
						popup.SetPadding(math.Spacing{L: 20, T: 20, R: 20, B: 20})
						popup.OnClose(func() { popup = nil })
						unbanlabel := theme.CreateLabel()
						unbanlabel.SetText("Unbanned!")
						unbanlabel.SetFont(bigfont)
						unbanlabel.SetColor(gxui.Green80)
						popup.AddChild(unbanlabel)
						popup.Focus()
					}
				})
			}

			break
		}

		driver.Call(func() {
			if player != textbox.Text() {
				player = fmt.Sprintf("%s (%s)", player, textbox.Text())
			}
			label.SetText(fmt.Sprintf("Player %s is %s!", player, status))
			label.SetColor(color)
		})
	}

	button.OnClick(func(gxui.MouseEvent) { checkban() })

	// this will notify the refresh thread that the textbox has changed
	textchanged := make(chan bool)

	textbox.OnTextChanged(func(edits []gxui.TextBoxEdit) {
		textchanged <- false

		// trim newlines when text is added
		// (disabling multiline still allows the user to paste multiline strings and fuck up the layout)
		for _, edit := range edits {
			if edit.Delta > 0 {
				defer textbox.SetText(strings.Trim(textbox.Text(), "\r\n"))
				break
			}
		}
		fmt.Println(edits)
	})

	// refresh ticker
	ticker := time.NewTicker(time.Minute * 5)

	// this thread re-checks every 5 minutes and updates the banned status 500ms after you typed a new player name
	go func() {
		checkban() // initial refresh

		terminate := false
		checked := true

		for {
			if terminate {
				break
			}

			// if the banned status was refreshed last loop, wait for the next text change
			if checked {
				fmt.Println("waiting for a textbox change")
				select {
				case <-textchanged:
					checked = false

				case <-ticker.C: // refresh every 5 mins
					checkban()
					continue
				}
			}

			// if the textbox changes within the next 500ms, the refresh will be ignored.
			// otherwise, the banned status will be refreshed after 500ms elapsed.
			select {
			case terminate = <-textchanged:
				fmt.Println("stopping refresh")

			case <-time.After(500 * time.Millisecond):
				fmt.Println("checking ban after textbox changed")
				go checkban()
				checked = true
			}
		}
	}()

	window.OnClose(func() {
		// kill refresh thread
		ticker.Stop()
		textchanged <- true
	})
}

func main() {
	gl.StartDriver(appMain)
}
