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

//TODO:Error Logging Instead of Printing for later improvement

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

func showStat(data UsageData) {
	go func() {
		for {
			var input string
			//TODO: improve user input handling using buffio instead or getred of it
			fmt.Scan(&input)
			if input == "s" {
				fmt.Println("----- App Usage So Far -----")
				for date, apps := range data {
					fmt.Println("Date:", date)
					for app, duration := range apps {
						//TODO:improve formatting style
						fmt.Printf("%s: %.2f minutes\n", app, duration.Minutes())
					}
				}
			}
		}
	}()
}

// findInterestingProcess looks for a more meaningful process name in the process tree

// TODO: improving this function
func findInterestingProcess(pid string) string {
	// These are applications we want to track specifically
	interestingApps := map[string]bool{
		"vim":       true,
		"nvim":      true,
		"neovim":    true,
		"emacs":     true,
		"code":      true,
		"python":    true,
		"python3":   true,
		"python2":   true,
		"node":      true,
		"java":      true,
		"ruby":      true,
		"perl":      true,
		"go":        true,
		"cargo":     true,
		"rustc":     true,
		"gcc":       true,
		"clang":     true,
		"g++":       true,
		"make":      true,
		"cmake":     true,
		"docker":    true,
		"kubectl":   true,
		"terraform": true,
		"npm":       true,
		"yarn":      true,
		"pip":       true,
		"mysql":     true,
		"psql":      true,
		"mongodb":   true,
		"redis-cli": true,
	}

	// First get the original process name
	cmd := exec.Command("ps", "-p", pid, "-o", "comm=")
	processBytes, err := cmd.Output()
	if err != nil {
		fmt.Println("Error getting original process name:", err)
		return "unknown"
	}
	originalProcess := strings.TrimSpace(string(processBytes))

	// Get all child processes recursively
	childProcesses := getAllChildProcesses(pid)

	// Look for interesting applications in child processes
	for _, childPid := range childProcesses {
		cmd := exec.Command("ps", "-p", childPid, "-o", "comm=")
		processBytes, err := cmd.Output()
		if err != nil {
			continue
		}

		processName := strings.TrimSpace(string(processBytes))

		// Check if this is an interesting application
		if interestingApps[processName] {
			return processName
		}
	}

	// If no interesting apps found in children, return original process
	return originalProcess
}

// getAllChildProcesses returns all descendant processes of a given PID
func getAllChildProcesses(pid string) []string {
	allChildren := []string{}

	// Get immediate children
	cmd := exec.Command("pgrep", "-P", pid)
	out, err := cmd.Output()
	if err != nil {
		return allChildren // No children found
	}

	children := strings.Fields(string(out))
	allChildren = append(allChildren, children...)

	// Recursively get children of children
	for _, childPid := range children {
		grandchildren := getAllChildProcesses(childPid)
		allChildren = append(allChildren, grandchildren...)
	}

	return allChildren
}

// getWindowTitle gets the title of the active window
func getWindowTitle() (string, error) {
	cmd := exec.Command("xdotool", "getactivewindow", "getwindowname")
	titleBytes, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(titleBytes)), nil
}

// getApplicationInfo returns application name and enriched info
func getApplicationInfo() (string, error) {
	// Get window PID
	cmd := exec.Command("xdotool", "getactivewindow", "getwindowpid")
	pidBytes, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("error getting PID: %w", err)
	}
	pid := strings.TrimSpace(string(pidBytes))

	// Get process name
	cmd = exec.Command("ps", "-p", pid, "-o", "comm=")
	processBytes, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("error getting process name: %w", err)
	}
	appName := strings.TrimSpace(string(processBytes))

	// For terminal apps, look deeper
	terminalApps := map[string]bool{
		"kitty":          true,
		"alacritty":      true,
		"gnome-terminal": true,
		"konsole":        true,
		"xterm":          true,
		"urxvt":          true,
		"terminator":     true,
		"terminal":       true,
		"iterm2":         true,
	}

	if terminalApps[appName] {
		interestingApp := findInterestingProcess(pid)
		if interestingApp != appName && interestingApp != "unknown" {
			// Return both the terminal and the app inside it
			return interestingApp, nil
		}
	}

	return appName, nil
}

func main() {
	const fileName = "usage.json"
	data := loadFromFile(fileName)
	var lastApp string
	var lastCheck time.Time = time.Now()
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	// Handle Ctrl+C
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		fmt.Println("\nReceived interrupt signal. Saving usage data and exiting...")
		saveToFile(fileName, data)
		os.Exit(0)
	}()

	showStat(data)

	go func() {
		for range ticker.C {
			// Get app info
			appName, err := getApplicationInfo()
			if err != nil {
				fmt.Println(err)
				continue
			}
			// Track time
			now := time.Now()
			dateKey := now.Format("2006-01-02")

			// Ensure map for current date exists
			if _, ok := data[dateKey]; !ok {
				data[dateKey] = make(DayUsage)
			}

			// Add elapsed time for the previous app
			if lastApp != "" {
				elapsed := now.Sub(lastCheck)
				data[dateKey][lastApp] += elapsed
			}

			// Update tracking info
			lastApp = appName
			lastCheck = now

			// Save data periodically
			//TODO:Prevent Unnecessary File Writes Every Tick to be when changes occur
			if now.Second()%10 == 0 { // Save every 10 seconds
				saveToFile(fileName, data)
			}
		}
	}()
	done := make(chan struct{})
	<-done // keep main alive forever
}
