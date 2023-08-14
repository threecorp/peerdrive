package event

import (
	"fmt"

	"github.com/gookit/color"
)

func dispStyle(ev *Event) func(a ...any) string {
	switch ev.Op {
	case Write:
		return color.New(color.FgLightGreen, color.Bold).Render
	case Read:
		return color.New(color.Gray, color.Bold).Render
	case Remove:
		return color.New(color.FgLightRed, color.Bold).Render
	default:
		return color.New(color.FgWhite).Render
	}
}

func DispSender(ev *Event) {
	fmt.Printf("%s %s %s\n", "⫸", dispStyle(ev)(ev.String()), color.Gray.Render(ev.Path))
}

func DispRecver(ev *Event) {
	fmt.Printf("%s %s %s\n", "⫷", dispStyle(ev)(ev.String()), color.Gray.Render(ev.Path))
}

func DispSendRenamed(path string) {
	eventColor := color.New(color.FgLightYellow, color.Bold).Render
	pathColor := color.New(color.Gray, color.Bold).Render
	fmt.Printf("%s %s %s\n", "⫸", eventColor("Renamed"), pathColor(path))
}

func DispSendCreated(path string) {
	eventColor := color.New(color.FgLightBlue, color.Bold).Render
	pathColor := color.New(color.Gray, color.Bold).Render
	fmt.Printf("%s %s %s\n", "⫸", eventColor("Created"), pathColor(path))
}

func DispSendWritten(path string) {
	eventColor := color.New(color.FgLightGreen, color.Bold).Render
	pathColor := color.New(color.Gray, color.Bold).Render
	fmt.Printf("%s %s %s\n", "⫸", eventColor("Written"), pathColor(path))
}

func DispSendRemoved(path string) {
	eventColor := color.New(color.FgLightRed, color.Bold).Render
	pathColor := color.New(color.Gray, color.Bold).Render
	fmt.Printf("%s %s %s\n", "⫸", eventColor("Removed"), pathColor(path))
}

func DispSendChanged(path string) {
	eventColor := color.New(color.FgLightGreen, color.Bold).Render
	pathColor := color.New(color.Gray, color.Bold).Render
	fmt.Printf("%s %s %s\n", "⫸", eventColor("Changed"), pathColor(path))
}

func DispRecvChanged(path string) {
	eventColor := color.New(color.FgLightGreen, color.Bold).Render
	pathColor := color.New(color.Gray, color.Bold).Render
	fmt.Printf("%s %s %s\n", "⫷", eventColor("Changed"), pathColor(path))
}

func DispRecvDeleted(path string) {
	eventColor := color.New(color.FgLightRed, color.Bold).Render
	pathColor := color.New(color.Gray, color.Bold).Render
	fmt.Printf("%s %s %s\n", "⫷", eventColor("Deleted"), pathColor(path))
}
