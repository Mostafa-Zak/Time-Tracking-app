package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

func loadFromFile(fileName string) map[string]time.Duration {

	appTime := make(map[string]time.Duration)

	// Check if file exists
	if _, err := os.Stat(fileName); os.IsNotExist(err) {
		// File does not exist, create an empty one
		emptyData, _ := json.MarshalIndent(appTime, "", "  ")
		err := os.WriteFile(fileName, emptyData, 0644)
		if err != nil {
			fmt.Println("Error creating initial JSON file:", err)
		}
		return appTime
	}

	// File exists, read and unmarshal
	fileBytes, _ := os.ReadFile(fileName)
	err := json.Unmarshal(fileBytes, &appTime)
	if err != nil {
		fmt.Println("Error decoding JSON:", err)
	}
	return appTime
}
func saveToFile(fileName string, appTime map[string]time.Duration) {
	data, err := json.MarshalIndent(appTime, "", "  ")
	if err != nil {
		fmt.Println("Error encoding JSON:", err)
		return
	}
	err = os.WriteFile(fileName, data, 0644)
	if err != nil {
		fmt.Println("Error writing file:", err)
	}
}
func main() {
	appTime := loadFromFile("usage.json")
	var lastApp string
	var lastCheck time.Time = time.Now()
	t := time.Tick(10 * time.Second)
	for {
		<-t
		cmd := exec.Command("xdotool", "getactivewindow", "getwindowpid")
		pidBytes, err := cmd.Output()
		if err != nil {
			fmt.Println("Error getting PID:", err)
			continue
		}

		pid := strings.TrimSpace(string(pidBytes))
		cmd2 := exec.Command("ps", "-p", pid, "-o", "comm=")
		processBytes, err := cmd2.Output()
		if err != nil {
			fmt.Println("Error getting process name:", err)
			continue
		}
		currentApp := strings.TrimSpace(string(processBytes))
		//track time
		now := time.Now()
		if lastApp != "" {
			elapsed := now.Sub(lastCheck)
			appTime[lastApp] += elapsed
		}
		lastApp = currentApp
		lastCheck = now
		fmt.Println("----- App Usage So Far -----")
		for app, duration := range appTime {

			fmt.Printf("%s: %.2f minutes\n", app, duration.Minutes())
		}

		if lastApp != "" && currentApp == lastApp {
			elapsed := now.Sub(lastCheck)
			if elapsed > 0 {
				appTime[lastApp] += elapsed
				saveToFile("usage.json", appTime)
			}
		}
	}

}
