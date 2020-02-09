package cli

import (
	"fmt"

	"github.com/fatih/color"
)

// Silent silences all the non-error messages
var Silent bool

// Verbose allows printing info messages.
var Verbose bool

// Warningln formats warning message
func Warningln(content ...interface{}) {
	if Silent {
		return
	}
	fmt.Println("[" + color.YellowString("!") + "] " + fmt.Sprint(content...))
}

// Successln formats success message
func Successln(content ...interface{}) {
	if Silent {
		return
	}
	fmt.Println("[" + color.GreenString("✓") + "] " + fmt.Sprint(content...))
}

// Infoln formats info message
func Infoln(content ...interface{}) {
	if Silent {
		return
	}
	fmt.Println("[" + color.BlueString("•") + "] " + fmt.Sprint(content...))
}

// Verboseln formats info message
func Verboseln(content ...interface{}) {
	if Silent || !Verbose {
		return
	}
	fmt.Println("[" + color.BlueString("•") + "] " + fmt.Sprint(content...))
}

// Failureln formats failure message
func Failureln(content ...interface{}) {
	fmt.Println("[" + color.RedString("x") + "] " + fmt.Sprint(content...))
}

// Warningf formats warning message
func Warningf(format string, values ...interface{}) {
	if Silent {
		return
	}
	fmt.Print("[" + color.YellowString("!") + "] " + fmt.Sprintf(format, values...))
}

// Successf formats success message
func Successf(format string, values ...interface{}) {
	if Silent {
		return
	}
	fmt.Print("[" + color.GreenString("✓") + "] " + fmt.Sprintf(format, values...))
}

// Infof formats info message
func Infof(format string, values ...interface{}) {
	if Silent {
		return
	}
	fmt.Print("[" + color.BlueString("•") + "] " + fmt.Sprintf(format, values...))
}

// Verbosef formats info message
func Verbosef(format string, values ...interface{}) {
	if Silent || !Verbose {
		return
	}
	fmt.Print("[" + color.BlueString("•") + "] " + fmt.Sprintf(format, values...))
}

// Failuref formats failure message
func Failuref(format string, values ...interface{}) {
	fmt.Print("[" + color.RedString("x") + "] " + fmt.Sprintf(format, values...))
}
