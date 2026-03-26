package main

import (
	"fmt"
	"os"
	"time"
)

func main() {
	fmt.Println("[Terminator] SIGTERM received. Initiating capital protection check...")

	// Timeout after 5 minutes to prevent hung nodes
	timeout := time.After(300 * time.Second)
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			fmt.Println("[Terminator] Timeout reached. Forcing termination to prevent node deadlock.")
			os.Exit(0)
		case <-ticker.C:
			if !checkActivePositions() {
				fmt.Println("[Terminator] No active trades detected. Safe to exit.")
				os.Exit(0)
			}
			fmt.Println("[Terminator] Warning: Active Hyperliquid position detected. Blocking termination...")
		}
	}
}

func checkActivePositions() bool {
	// MOCK LOGIC: In production, this would query the local agent state file
	// or the Hyperliquid API: /info -> "userOpenPositions"
	// For PoC: Check if a dummy file exists
	_, err := os.Stat("/tmp/active_trade.lock")
	return !os.IsNotExist(err)
}
