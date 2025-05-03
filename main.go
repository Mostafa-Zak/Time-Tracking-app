package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

type DayUsage map[string]time.Duration
type UsageData map[string]DayUsage // date -> { app -> duration }

func loadFromFile(fileName string) UsageData {

	data := make(UsageData)

	// Check if file exists
	if _, err := os.Stat(fileName); os.IsNotExist(err) {
		// File does not exist, create an empty one
		emptyData, _ := json.MarshalIndent(data, "", "  ")
		err := os.WriteFile(fileName, emptyData, 0644)
		if err != nil {
			fmt.Println("Error creating initial JSON file:", err)
		}
		return data
	}

	// File exists, read and unmarshal
	fileBytes, _ := os.ReadFile(fileName)
	err := json.Unmarshal(fileBytes, &data)
	if err != nil {
		fmt.Println("Error decoding JSON:", err)
	}
	return data
}
func saveToFile(fileName string, data UsageData) {
	bytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		fmt.Println("Error encoding JSON:", err)
		return
	}
	err = os.WriteFile(fileName, bytes, 0644)
	if err != nil {
		fmt.Println("Error writing file:", err)
	}
}
func main() {
	const fileName = "usage.json"
	data := loadFromFile(fileName)
	var lastApp string
	var lastCheck time.Time = time.Now()
	ticker := time.Tick(10 * time.Second)

	// Handle Ctrl+C
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		fmt.Println("\nReceived interrupt signal. Saving usage data and exiting...")
		saveToFile(fileName, data)
		os.Exit(0)
	}()
	for {
		<-ticker
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
		dateKey := now.Format("2006-01-02")
		if _, ok := data[dateKey]; !ok {
			data[dateKey] = make(DayUsage)
		}

		if lastApp != "" {
			elapsed := now.Sub(lastCheck)
			data[dateKey][lastApp] += elapsed
		}
		lastApp = currentApp
		lastCheck = now
		fmt.Println("----- App Usage So Far -----")
		for date, apps := range data {

			fmt.Println("Date:", date)
			for app, duration := range apps {
				fmt.Printf("%s: %.2f minutes\n", app, duration.Minutes())
			}
		}

		if lastApp != "" && currentApp == lastApp {
			elapsed := now.Sub(lastCheck)
			if elapsed > 0 {
				data[dateKey][lastApp] += elapsed
				saveToFile("usage.json", data)
			}
		}
	}

}
