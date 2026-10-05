package app

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/ghostlawless/xdl/internal/config"
	"github.com/ghostlawless/xdl/internal/downloader"
	"github.com/ghostlawless/xdl/internal/utils"
)

const htmlContent = `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>XDL Downloader</title>
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif; padding: 20px; background: #f4f6f8; color: #333; }
        .container { max-width: 800px; margin: 0 auto; background: white; padding: 20px; border-radius: 8px; box-shadow: 0 2px 4px rgba(0,0,0,0.1); }
        h1 { margin-top: 0; color: #1da1f2; }
        .form-group { margin-bottom: 15px; }
        label { display: block; font-weight: bold; margin-bottom: 5px; }
        input[type="text"] { width: 100%; padding: 8px; box-sizing: border-box; border: 1px solid #ccc; border-radius: 4px; }
        .buttons { margin-top: 20px; display: flex; gap: 10px; }
        button { padding: 10px 20px; border: none; border-radius: 4px; cursor: pointer; font-weight: bold; }
        .btn-test { background: #e1e8ed; color: #14171a; }
        .btn-start { background: #1da1f2; color: white; }
        .btn-stop { background: #e0245e; color: white; }
        button:disabled { opacity: 0.6; cursor: not-allowed; }
        #log { margin-top: 20px; background: #14171a; color: #fff; padding: 10px; border-radius: 4px; height: 300px; overflow-y: auto; font-family: monospace; white-space: pre-wrap; word-wrap: break-word;}
        .status { margin-top: 10px; font-weight: bold; }
    </style>
</head>
<body>
    <div class="container">
        <h1>XDL Web UI</h1>
        <div class="form-group">
            <label for="username">Twitter Username(s) (space separated):</label>
            <input type="text" id="username" placeholder="e.g. elonmusk">
        </div>
        <div class="form-group">
            <label for="outroot">Download Folder (optional):</label>
            <input type="text" id="outroot" placeholder="e.g. C:\Downloads\XDL">
        </div>
        
        <div class="buttons">
            <button class="btn-test" id="btnTest" onclick="testConnection()">Test Connection</button>
            <button class="btn-start" id="btnStart" onclick="startDownload()">Start Download</button>
            <button class="btn-test" id="btnFix" onclick="startFix()">Fix Local Times</button>
            <button class="btn-stop" id="btnStop" onclick="stopDownload()" disabled>Stop</button>
        </div>
        <div class="status" id="statusMsg"></div>
        <div id="log"></div>
    </div>

    <script>
        let eventSource = null;
        let currentLine = null;

        function setStatus(msg, color) {
            const el = document.getElementById('statusMsg');
            el.innerText = msg;
            el.style.color = color;
        }

        function appendLog(msg) {
            const logEl = document.getElementById('log');
            if (currentLine) {
                currentLine.innerText = msg;
                currentLine = null;
            } else {
                const div = document.createElement('div');
                div.innerText = msg;
                logEl.appendChild(div);
            }
            logEl.scrollTop = logEl.scrollHeight;
        }

        function replaceLog(msg) {
            const logEl = document.getElementById('log');
            if (!currentLine) {
                currentLine = document.createElement('div');
                logEl.appendChild(currentLine);
            }
            currentLine.innerText = msg;
            logEl.scrollTop = logEl.scrollHeight;
        }

        async function testConnection() {
            const username = document.getElementById('username').value.trim();
            if (!username) {
                setStatus("Please enter a username to test.", "red");
                return;
            }
            
            const btn = document.getElementById('btnTest');
            btn.disabled = true;
            setStatus("Testing connection...", "blue");
            
            try {
                const target = username.split(" ")[0];
                const res = await fetch('/api/test?user=' + encodeURIComponent(target));
                const data = await res.json();
                if (data.ok) {
                    setStatus("Connection successful! Target UID: " + data.uid, "green");
                } else {
                    setStatus("Connection failed: " + data.error, "red");
                }
            } catch (err) {
                setStatus("Error: " + err, "red");
            } finally {
                btn.disabled = false;
            }
        }

        function startFix() {
            const outdir = document.getElementById('outroot').value.trim();
            document.getElementById('log').innerHTML = "";
            currentLine = null;
            document.getElementById('btnStart').disabled = true;
            document.getElementById('btnTest').disabled = true;
            document.getElementById('btnFix').disabled = true;
            document.getElementById('btnStop').disabled = false;
            setStatus("Fixing local file times...", "blue");

            let url = '/api/fix?';
            if (outdir) {
                url += 'outroot=' + encodeURIComponent(outdir);
            }

            eventSource = new EventSource(url);
            eventSource.addEventListener("append", function(e) {
                appendLog(e.data);
            });
            eventSource.addEventListener("replace", function(e) {
                replaceLog(e.data);
            });
            eventSource.addEventListener("done", function(e) {
                appendLog("--- Fix Completed ---");
                setStatus("Fix completed.", "green");
                resetButtons();
            });
            eventSource.addEventListener("error_msg", function(e) {
                appendLog("ERROR: " + e.data);
                setStatus("Fix failed.", "red");
                resetButtons();
            });
            eventSource.onerror = function(e) {
                if (eventSource.readyState === EventSource.CLOSED) {
                    resetButtons();
                }
            };
        }

        function startDownload() {
            const users = document.getElementById('username').value.trim();
            const outdir = document.getElementById('outroot').value.trim();
            if (!users) {
                setStatus("Please enter at least one username.", "red");
                return;
            }

            document.getElementById('log').innerHTML = "";
            currentLine = null;
            document.getElementById('btnStart').disabled = true;
            document.getElementById('btnTest').disabled = true;
            document.getElementById('btnStop').disabled = false;
            setStatus("Downloading...", "blue");

            let url = '/api/download?users=' + encodeURIComponent(users);
            if (outdir) {
                url += '&outroot=' + encodeURIComponent(outdir);
            }

            eventSource = new EventSource(url);
            eventSource.addEventListener("append", function(e) {
                appendLog(e.data);
            });
            eventSource.addEventListener("replace", function(e) {
                replaceLog(e.data);
            });
            eventSource.addEventListener("done", function(e) {
                appendLog("--- Task Completed ---");
                setStatus("Download completed.", "green");
                resetButtons();
            });
            eventSource.addEventListener("error_msg", function(e) {
                appendLog("ERROR: " + e.data);
                setStatus("Download failed.", "red");
                resetButtons();
            });
            eventSource.onerror = function(e) {
                if (eventSource.readyState === EventSource.CLOSED) {
                    resetButtons();
                }
            };
        }

        function stopDownload() {
            if (eventSource) {
                eventSource.close();
            }
            fetch('/api/stop', { method: 'POST' });
            setStatus("Download stopped by user.", "orange");
            appendLog("--- Task Stopped ---");
            resetButtons();
        }

        function resetButtons() {
            if (eventSource) {
                eventSource.close();
                eventSource = null;
            }
            document.getElementById('btnStart').disabled = false;
            document.getElementById('btnTest').disabled = false;
            document.getElementById('btnFix').disabled = false;
            document.getElementById('btnStop').disabled = true;
        }
    </script>
</body>
</html>
`

var currentCmd *exec.Cmd

func StartGUI() error {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(htmlContent))
	})

	http.HandleFunc("/api/test", handleTest)
	http.HandleFunc("/api/download", handleDownload)
	http.HandleFunc("/api/fix", handleFix)
	http.HandleFunc("/api/stop", handleStop)

	port := "8080"
	url := "http://localhost:" + port
	fmt.Printf("Starting Web UI on %s\n", url)

	go func() {
		utils.OpenBrowser(url)
	}()

	return http.ListenAndServe(":"+port, nil)
}

func handleTest(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	user := r.URL.Query().Get("user")
	if user == "" {
		json.NewEncoder(w).Encode(map[string]any{"ok": false, "error": "empty user"})
		return
	}

	paths := []string{
		filepath.Join(".", "config", "essentials.json"),
		filepath.Join(".", "essentials.json"),
	}
	cf, err := config.LoadEssentialsWithFallback(paths)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]any{"ok": false, "error": "config error: " + err.Error()})
		return
	}
	
	_ = config.ApplyCookiesFromFile(cf, "")
	if err := cf.ValidateRequiredCookies(""); err != nil {
		json.NewEncoder(w).Encode(map[string]any{"ok": false, "error": "cookie error: " + err.Error()})
		return
	}

	client := buildAPIClient(cf.HTTPTimeout())
	
	ctx := RunContext{Mode: ModeQuiet}
	uid, err := resolveUserID(ctx, cf, client, user, nil)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]any{"ok": false, "error": err.Error()})
		return
	}

	json.NewEncoder(w).Encode(map[string]any{"ok": true, "uid": uid})
}

func handleDownload(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	users := r.URL.Query().Get("users")
	outroot := r.URL.Query().Get("outroot")
	
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported!", http.StatusInternalServerError)
		return
	}

	exe, err := os.Executable()
	if err != nil {
		fmt.Fprintf(w, "event: error_msg\ndata: %s\n\n", err.Error())
		flusher.Flush()
		return
	}

	args := []string{}
	args = append(args, strings.Split(users, " ")...)
	
	cmd := exec.Command(exe, args...)
	if outroot != "" {
		cmd.Env = append(os.Environ(), "XDL_OUTROOT="+outroot)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		fmt.Fprintf(w, "event: error_msg\ndata: %s\n\n", err.Error())
		flusher.Flush()
		return
	}
	cmd.Stderr = cmd.Stdout

	currentCmd = cmd

	if err := cmd.Start(); err != nil {
		fmt.Fprintf(w, "event: error_msg\ndata: %s\n\n", err.Error())
		flusher.Flush()
		return
	}

	go func() {
		buf := make([]byte, 1024)
		var line strings.Builder
		for {
			n, err := stdout.Read(buf)
			if n > 0 {
				str := string(buf[:n])
				for _, ch := range str {
					if ch == '\n' {
						msg := strings.ReplaceAll(line.String(), "\n", " ")
						fmt.Fprintf(w, "event: append\ndata: %s\n\n", msg)
						flusher.Flush()
						line.Reset()
					} else if ch == '\r' {
						if line.Len() > 0 {
							msg := strings.ReplaceAll(line.String(), "\n", " ")
							fmt.Fprintf(w, "event: replace\ndata: %s\n\n", msg)
							flusher.Flush()
							line.Reset()
						}
					} else {
						line.WriteRune(ch)
					}
				}
			}
			if err != nil {
				break
			}
		}
		if line.Len() > 0 {
			msg := strings.ReplaceAll(line.String(), "\n", " ")
			fmt.Fprintf(w, "event: append\ndata: %s\n\n", msg)
			flusher.Flush()
		}
		
		cmd.Wait()
		fmt.Fprintf(w, "event: done\ndata: completed\n\n")
		flusher.Flush()
	}()

	<-r.Context().Done()
	if currentCmd != nil && currentCmd.Process != nil {
		currentCmd.Process.Kill()
	}
}

func handleFix(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	outroot := r.URL.Query().Get("outroot")
	if outroot == "" {
		if env := os.Getenv("XDL_OUTROOT"); env != "" {
			outroot = env
		} else {
			outroot = "xDownloads"
		}
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported!", http.StatusInternalServerError)
		return
	}

	go func() {
		fmt.Fprintf(w, "event: append\ndata: Scanning directory: %s\n\n", outroot)
		flusher.Flush()

		err := downloader.FixLocalFiles(outroot, func(msg string) {
			fmt.Fprintf(w, "event: replace\ndata: %s\n\n", msg)
			flusher.Flush()
		})

		if err != nil {
			fmt.Fprintf(w, "event: error_msg\ndata: %s\n\n", err.Error())
		} else {
			fmt.Fprintf(w, "event: done\ndata: completed\n\n")
		}
		flusher.Flush()
	}()

	<-r.Context().Done()
}

func handleStop(w http.ResponseWriter, r *http.Request) {
	if currentCmd != nil && currentCmd.Process != nil {
		currentCmd.Process.Kill()
	}
	w.WriteHeader(http.StatusOK)
}
