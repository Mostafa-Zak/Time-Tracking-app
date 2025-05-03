package main

import (
	"fmt"
	"os/exec"
	"strings"
	"time"
)

func main() {
	appTime := make(map[string]time.Duration)
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
			fmt.Printf("%s: %v\n", app, duration)
		}
	}
}
