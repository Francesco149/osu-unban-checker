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

	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"
)

type OsuUser struct {
	Username string `json:"username"`
}

type OsuError struct {
	Error string `json:"error"`
}

func loadApiKey() (string, error) {
	buf := bytes.NewBuffer(nil)

	f, err := os.Open("apikey.txt")
	if err != nil {
		return "", err
	}
	defer f.Close()

	_, err = io.Copy(buf, f)
	if err != nil {
		return "", err
	}

	return string(buf.Bytes()), nil
}

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

	// larger font for the pop-up
	bigfont, err := driver.CreateFont(gxfont.Default, 50)
	if err != nil {
		panic(err)
	}

	// unban pop-up
	var popup gxui.Window

	// checks for ban and updates the gui
	checkban := func() {
		apikey, err := loadApiKey()
		if err != nil {
			driver.Call(func() {
				label.SetText(fmt.Sprintf("Failed to load api key (%v)", err))
				label.SetColor(gxui.Yellow)
			})
			return
		}
		if len(apikey) == 0 {
			driver.Call(func() {
				label.SetText("Please enter your osu! api key in apikey.txt!")
				label.SetColor(gxui.Yellow)
			})
			return
		}

		apikey = strings.Replace(apikey, "\r", "", -1)
		apikey = strings.Replace(apikey, "\n", "", -1)

		status := "still banned"
		color := gxui.Red80
		player := textbox.Text()

		driver.Call(func() {
			label.SetText("Checking...")
			label.SetColor(gxui.Gray80)
		})

		for {
			url := fmt.Sprintf("https://osu.ppy.sh/api/get_user?k=%s&u=%s", apikey, player)
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

			fmt.Println(string(body[:]))

			var errorResp OsuError
			err = json.Unmarshal(body[:], &errorResp)
			if err == nil && len(errorResp.Error) > 0 {
				driver.Call(func() {
					label.SetText(errorResp.Error)
					label.SetColor(gxui.Yellow)
				})
				return
			}

			var userResp []OsuUser
			err = json.Unmarshal(body[:], &userResp)
			if err != nil {
				fmt.Println(err)
				driver.Call(func() {
					label.SetText("Failed to decode response.")
					label.SetColor(gxui.Yellow)
				})
				return
			}
			if len(userResp) < 1 {
				break // still banned
			}

			status = "unbanned"
			color = gxui.Green80
			player = userResp[0].Username

			// TODO: allow user to choose a specific user when multiple results are returned

			// show pop-up if enabled
			driver.Call(func() {
				if !checkbox.IsChecked() {
					return
				}

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
			})

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

	// bind check button
	button.OnClick(func(gxui.MouseEvent) { go checkban() })

	// this will notify the refresh thread that the textbox has changed
	textchanged := make(chan bool)

	// bind textbox changed event
	textbox.OnTextChanged(func(edits []gxui.TextBoxEdit) {
		textchanged <- false

		// strip newlines when text is added
		// (disabling multiline still allows the user to paste multiline strings and fuck up the layout)
		for _, edit := range edits {
			if edit.Delta > 0 {
				defer textbox.SetText(
					strings.Replace(strings.Replace(textbox.Text(), "\n", "", -1),
						"\r", "", -1))
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
					go checkban()
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
