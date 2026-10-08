package internal

import (
	"bufio"
	"bytes"
	"cmp"
	"fmt"
	"net/http"
	"net/http/httputil"
	"os"
	"runtime"
	"strconv"
	"strings"
)

const dunno = "???"

// Credit: github.com/gin-gonic/gin/recovery.go
func DumpRequest(r *http.Request) string {
	httpRequest, _ := httputil.DumpRequest(r, false)
	output := strconv.Quote(string(httpRequest))
	output = strings.ReplaceAll(output, `\n`, "\n")
	output = strings.ReplaceAll(output, `\r`, "")
	return strings.Trim(output, `"`)
}

// Credit: github.com/gin-gonic/gin/recovery.go
func DumpStack() string {
	out := new(bytes.Buffer)
	// As we loop, we open files and read them. These variables record the currently
	// loaded file.
	var (
		nLine    string
		lastFile string
		err      error
	)
	for i := 3; ; i++ { // Skip the expected number of frames
		pc, file, line, ok := runtime.Caller(i)
		if !ok {
			break
		}
		// Print this much at least.  If we can't find the source, it won't show.
		fmt.Fprintf(out, "%s:%d (0x%x)\n", file, line, pc)
		if file != lastFile {
			nLine, err = readNthLine(file, line-1)
			if err != nil {
				continue
			}
			lastFile = file
		}
		fmt.Fprintf(out, "\t%s: %s\n", function(pc), cmp.Or(nLine, dunno))
	}
	return out.String()
}

// Credit: github.com/gin-gonic/gin/recovery.go
// readNthLine reads the nth line from the file.
// It returns the trimmed content of the line if found,
// or an empty string if the line doesn't exist.
// If there's an error opening the file, it returns the error.
func readNthLine(file string, n int) (string, error) {
	if n < 0 {
		return "", nil
	}

	f, err := os.Open(file)
	if err != nil {
		return "", err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for range n {
		if !scanner.Scan() {
			return "", nil
		}
	}

	if scanner.Scan() {
		return strings.TrimSpace(scanner.Text()), nil
	}

	return "", scanner.Err()
}

// Credit: github.com/gin-gonic/gin/recovery.go
// function returns, if possible, the name of the function containing the PC.
func function(pc uintptr) string {
	fn := runtime.FuncForPC(pc)
	if fn == nil {
		return dunno
	}
	name := fn.Name()
	// The name includes the path name to the package, which is unnecessary
	// since the file name is already included.  Plus, it has center dots.
	// That is, we see
	//	runtime/debug.*T·ptrmethod
	// and want
	//	*T.ptrmethod
	// Also the package path might contain dot (e.g. code.google.com/...),
	// so first eliminate the path prefix
	if lastSlash := strings.LastIndexByte(name, '/'); lastSlash >= 0 {
		name = name[lastSlash+1:]
	}
	if period := strings.IndexByte(name, '.'); period >= 0 {
		name = name[period+1:]
	}
	name = strings.ReplaceAll(name, "·", ".")
	return name
}
