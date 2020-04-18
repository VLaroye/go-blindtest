package godebug

import (
	"bytes"
	"fmt"
	"io"
	"os"

	"github.com/pschlump/MiscLib"
)

/*
TODO
	1. Add in JSON format
	2. Add in %(LINE) %(FUNC) etc.
	3. Add in %(TB:4) 	Trace back 4 levels


func x() {
	var buffer bytes.Buffer

	for i := 0; i < 1000; i++ {
		buffer.WriteString("a")
	}

	fmt.Println(buffer.String())
}
*/

var ColorTab map[string]string

func init() {
	ColorTab = make(map[string]string)

	ColorTab["red"] = MiscLib.ColorRed
	ColorTab["green"] = MiscLib.ColorGreen
	ColorTab["yellow"] = MiscLib.ColorYellow
	ColorTab["blue"] = MiscLib.ColorBlue
	ColorTab["cyan"] = MiscLib.ColorCyan
	ColorTab["reset"] = MiscLib.ColorReset

	ColorTab["Red"] = MiscLib.ColorRed
	ColorTab["Green"] = MiscLib.ColorGreen
	ColorTab["Yellow"] = MiscLib.ColorYellow
	ColorTab["Blue"] = MiscLib.ColorBlue
	ColorTab["Cyan"] = MiscLib.ColorCyan
	ColorTab["Reset"] = MiscLib.ColorReset

	ColorTab["!"] = MiscLib.ColorReset
}

func ProcessFormat(format string) string {
	var buffer bytes.Buffer
	colorFound := false
	var i, j int
	for i = 0; i < len(format); i++ {
		if format[i] == '%' && i+1 < len(format) && format[i+1] == '(' {
			color := "Red"
			for j = i + 2; j < len(format); j++ {
				if format[j] == ')' && i+2 < j {
					// fmt.Printf("Pick of %d:%d\n", i+2, j)
					color = format[i+2 : j]
					break
				}
			}
			i = j
			var ct string
			var ok bool
			switch color {
			case "LF":
				ct = LF(3)
			default:
				// fmt.Printf("--->%s<---\n", color)
				ct, ok = ColorTab[color]
				if !ok {
					ct = ColorTab["Red"]
				}
				colorFound = true
			}
			buffer.WriteString(ct)

			if color == "!" || color == "Reset" {
				colorFound = false
			}

			//		} else if format[i] == '%' && i+1 < len(format) && format[i+1] == 'J' {
			//			buffer.WriteString(SVarI(arg[nth]))
			//			removeNthArg()
			//		} else if format[i] == '%' && i+1 < len(format) && format[i+1] == 'j' {
			//			buffer.WriteString(SVarI(arg[nth]))
			//			removeNthArg()

		} else {
			buffer.WriteByte(format[i])
		}
	}
	if colorFound {
		buffer.WriteString(ColorTab["Reset"])
	}
	return buffer.String()
}

func Printf(format string, a ...interface{}) (n int, err error) {
	return fmt.Printf(ProcessFormat(format), a...)
}

func Fprintf(w io.Writer, format string, a ...interface{}) (n int, err error) {
	return fmt.Fprintf(w, ProcessFormat(format), a...)
}

func DbPf(db bool, format string, a ...interface{}) (n int, err error) {
	if db {
		return fmt.Printf(ProcessFormat(format), a...)
	}
	return 0, nil
}

func DbFpf(db bool, w io.Writer, format string, a ...interface{}) (n int, err error) {
	if db {
		return fmt.Fprintf(w, ProcessFormat(format), a...)
	}
	return 0, nil
}

func DbPfb(db bool, format string, a ...interface{}) (n int, err error) {
	if db {
		ff := ProcessFormat(format)
		fmt.Fprintf(os.Stderr, ff, a...)
		return fmt.Printf(ff, a...)
	}
	return 0, nil
}

func DbPfe(envVar string, format string, a ...interface{}) (n int, err error) {
	return DbPfb(ChkEnv(envVar), format, a...)
}
