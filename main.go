package main

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/hypebeast/go-osc/osc"
)

var client *osc.Client
var heartRateHistory []float64
var config Config

func main() {
	_, err := os.Stat("config.json")
	// No config let's create one
	if err != nil {
		CreateConfig()
	}
	ReadConfig()

	if runtime.GOOS != "windows" {
		config.ShowSpotify = false
	}

	fmt.Printf("=== HR-VR-OSC ===\nOSC Port: %d\nSpotify Enabled: %t\nTrend Enabled: %t\nHR Source: %s\n=================\n", config.OSCPort, config.ShowSpotify, config.ShowTrend, config.HeartRateSource)

	client = osc.NewClient("127.0.0.1", config.OSCPort)
	for true {
		stringHR, floatHR := GetHeartRate()
		message := stringHR + " BPM"

		// Trend logic
		if config.ShowTrend {
			if len(heartRateHistory) >= 4 {
				heartRateHistory = heartRateHistory[1:]
			}
			heartRateHistory = append(heartRateHistory, floatHR)
			if len(heartRateHistory) > 1 {
				slope := CalculateSlope(heartRateHistory)
				if slope > 0.65 {
					message += "⤴️"
				} else if slope < -0.65 {
					message += "⤵️"
				}
			}
		}

		// Spotify logic
		if config.ShowSpotify {
			currentlyPlaying := strings.TrimSpace(GetSpotifyPlaying())
			// If we're playing a song then add on the currently playing track
			if currentlyPlaying != "Spotify" && currentlyPlaying != "Spotify Free" {
				message += "\v" + "♫ " + currentlyPlaying + " ♫"
			}
		}

		SendOSCMessage(message)

		// VRChats chatbox has a ratelimit of about 3 seconds per.
		// It may support burst updates but doesn't really matter.
		time.Sleep(3 * time.Second)
	}
}

func SendOSCMessage(message string) {
	if client == nil {
		return
	}
	var msg *osc.Message = osc.NewMessage("/chatbox/input")
	msg.Append(message)
	msg.Append(true)
	client.Send(msg)
}

// This is a little messy
func GetHeartRate() (string, float64) {
	var url string

	// TODO: add more such as just accessing a file
	// theoretically you could use other services as well by using the HeartRateSource config option as long as they output the HR as just a number and have the API key as a URL param
	// ^ ex: https://example.com/hr?key=xxxxx
	if config.HeartRateSource == "PULSOID" {
		url = "https://dev.pulsoid.net/api/v1/data/heart_rate/latest?response_mode=text_plain_only_heart_rate" // text_plain_only_heart_rate only gives us the raw number of the HR instead of extra data such as JSON
	} else if strings.HasPrefix(config.HeartRateSource, "http") { // Source is just a url, just pull directly from it
		url = config.HeartRateSource
	} else {
		fmt.Printf("GetHeartRate(): '%s' is an invalid source\n", config.HeartRateSource)
		return "0", 0
	}

	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		fmt.Printf("GetHeartRate(): Failed creating HTTP request\nErr: %s\n=================\n", err)
		return "0", 0
	}
	req.Header.Set("Authorization", "Bearer "+config.HeartRateAPIKey)

	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("GetHeartRate(): Failed grabbing HR\nErr: %s\n=================\n", err)
		return "0", 0
	}
	defer resp.Body.Close()
	// PULSOID HR not ready
	if resp.StatusCode == 412 {
		return "0", 0
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("GetHeartRate(): Error reading HR\nErr: %s\n=================\n", err)
		return "0", 0
	}
	stringHR := string(data)
	floatHR, err := strconv.ParseFloat(stringHR, 64)
	if err != nil {
		fmt.Printf("GetHeartRate(): Failed to parse float\nErr: %s\n=================\n", err)
		return "0", 0
	}

	return stringHR, floatHR
}

func GetSpotifyPlaying() string {
	// Grab the data using Powershell. Calling WinAPI funcs directly is kinda ass.
	// tldr; filter processes and just get ones called 'Spotify' and the windowtitle isn't null.
	// 65001 instructs powershell to use a specific encoding to fix issues where it would output replacement chars for some songs/artists
	// Note: This doesn't fix all song names apparently. I've only tested JP song titles/artists so far.
	cmd := exec.Command("powershell.exe", "-c", "chcp", "65001", ">", "$null", ";", "Get-Process | Where-Object { $_.ProcessName -eq 'Spotify' -and $_.MainWindowTitle -ne '' } | Select-Object MainWindowTitle | Format-Table -AutoSize")
	out, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("GetSpotifyPlaying(): Error grabbing Error parsing output\nErr: %s\n=================\n", err)
		return "Spotify"
	}
	// Split the output otherwise we'll get data from powershell with useless bits
	data := strings.Split(string(out), "\n")
	// Spotify not running?
	if len(data) <= 1 {
		return "Spotify"
	}
	return data[3]
}

// https://github.com/BoiHanny/vrcosc-magicchatbox/blob/master/vrcosc-magicchatbox/Classes/HeartRateConnector.cs#L22C40-L22C40
func CalculateSlope(values []float64) float64 {
	count := len(values)
	avgX := float64(count) / 2.0
	avgY := Sum(values) / float64(count)

	var sumXY float64
	var sumXX float64

	for i := 0; i < count; i++ {
		sumXY += (float64(i) - avgX) * (values[i] - avgY)
		sumXX += math.Pow(float64(i)-avgX, 2)
	}

	slope := sumXY / sumXX
	return slope
}

func Sum(values []float64) float64 {
	var total float64
	for _, val := range values {
		total += val
	}
	return total
}

type Config struct {
	OSCPort         int
	HeartRateSource string
	HeartRateAPIKey string
	ShowSpotify     bool
	ShowTrend       bool
}

func ReadConfig() {
	data, err := os.ReadFile("config.json")
	if err != nil {
		fmt.Printf("ReadConfig(): Failed to read config\nErr: %s\n=================\n", err)
	}
	json.Unmarshal(data, &config)
	if err != nil {
		fmt.Printf("ReadConfig(): Failed to parse JSON\nErr: %s\n=================\n", err)
	}
	fmt.Println(`ReadConfig(): Successfully read config`)
}

func CreateConfig() {
	defaultConfig := &Config{
		OSCPort:         9000,
		HeartRateSource: "PULSOID",
		HeartRateAPIKey: "xxxx",
		ShowSpotify:     true,
		ShowTrend:       true,
	}
	bytes, _ := json.Marshal(defaultConfig)
	err := os.WriteFile("config.json", bytes, 0666)
	if err != nil {
		fmt.Printf("CreateConfig(): Failed to generate config\nErr: %s\n=================\n", err)
	}
	fmt.Println(`CreateConfig(): Successfully generated config`)
}
