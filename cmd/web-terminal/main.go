package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/farazzshaikh/web-terminal/cmd/web-terminal/helpers"
	"github.com/rs/cors"
)

type Command struct {
	CMD string
}

type Response struct {
	Stdout string
	Stderr string
}

var currentDir string = os.Getenv("HOME")

func WebTerminal(w http.ResponseWriter, r *http.Request) {
	var c Command

	err := helpers.DecodeJSONBody(w, r, &c)
	if err != nil {
		var mr *helpers.MalformedRequest
		if errors.As(err, &mr) {
			http.Error(w, mr.Msg, mr.Status)
		} else {
			log.Println(err.Error())
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
		return
	}

	whitelist := helpers.GetConfig().Whitelist

	cmdRaw := strings.Fields(c.CMD)
	cmd := c.CMD

	if len(cmdRaw) > 0 {
		cmd = cmdRaw[0]
	}

	if helpers.Contains(whitelist, cmd) {

		if cmd == "cd" {

			joined := filepath.Join(currentDir, cmdRaw[1])
			newDir, err := filepath.Abs(joined)
			if err != nil {
				panic(err)
			}

			c := fmt.Sprintf("cd %s", newDir)
			child := exec.Command("bash", "-c", c)
			err = child.Run()

			if err == nil {
				currentDir = newDir
			}

		} else {

			_c := fmt.Sprintf("cd %s && %s", currentDir, c.CMD)
			child := exec.Command("bash", "-c", _c)

			var stdout bytes.Buffer
			var stderr bytes.Buffer

			child.Stdout = &stdout
			child.Stderr = &stderr

			err = child.Run()

			r := Response{
				Stdout: stdout.String(),
				Stderr: stderr.String(),
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)

			json.NewEncoder(w).Encode(r)
		}

	} else {
		r := Response{
			Stdout: "",
			Stderr: fmt.Sprintf("web-terminal: command not found: %s", cmd),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(r)
		return
	}

}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/", WebTerminal)

	handler := cors.Default().Handler(mux)
	log.Println("Listing for requests at http://localhost:8000")
	log.Fatal(http.ListenAndServe(":8000", handler))
}
