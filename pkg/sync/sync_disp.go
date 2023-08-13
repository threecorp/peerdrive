package sync

import (
	"fmt"

	"github.com/gookit/color"
)

func sendDispChanged(path string) {
	eventColor := color.New(color.FgLightGreen, color.Bold).Render
	pathColor := color.New(color.Gray, color.Bold).Render
	fmt.Printf("%s %s %s\n", "⫸", eventColor("Changed"), pathColor(path))
}

func sendDispDeleted(path string) {
	eventColor := color.New(color.FgLightRed, color.Bold).Render
	pathColor := color.New(color.Gray, color.Bold).Render
	fmt.Printf("%s %s %s\n", "⫸", eventColor("Deleted"), pathColor(path))
}

func recvDispChanged(path string) {
	eventColor := color.New(color.FgLightGreen, color.Bold).Render
	pathColor := color.New(color.Gray, color.Bold).Render
	fmt.Printf("%s %s %s\n", "⫷", eventColor("Changed"), pathColor(path))
}

func recvDispDeleted(path string) {
	eventColor := color.New(color.FgLightRed, color.Bold).Render
	pathColor := color.New(color.Gray, color.Bold).Render
	fmt.Printf("%s %s %s\n", "⫷", eventColor("Deleted"), pathColor(path))
}
