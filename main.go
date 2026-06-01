package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"syscall"
	"time"

	"github.com/rmcluster/linux-client/fscas"
	"github.com/rmcluster/linux-client/openapi"
)

// time to wait after failed announcement
const retrySleep = time.Second

func getDataPath() string {
	// Check environment variable first
	if envPath := os.Getenv("RMCLUSTER_CLIENT_DATA_DIR"); envPath != "" {
		return envPath
	}

	// Fallback based on OS
	home, err := os.UserHomeDir()
	if err != nil {
		// Cannot determine a sane fallback
		return ""
	}

	switch runtime.GOOS {
	case "windows":
		// Use %LOCALAPPDATA% on Windows
		localAppData := os.Getenv("LOCALAPPDATA")
		if localAppData == "" {
			localAppData = filepath.Join(home, "AppData", "Local")
		}
		return filepath.Join(localAppData, "rmcluster-client")
	default:
		// Use XDG_DATA_HOME if present, otherwise ~/.local/share
		if xdgDataHome := os.Getenv("XDG_DATA_HOME"); xdgDataHome != "" {
			return filepath.Join(xdgDataHome, "rmcluster-client")
		}
		return filepath.Join(home, ".local", "share", "rmcluster-client")
	}
}

func main() {
	id := flag.String("id", "", "the id of the node")
	tracker := flag.String("tracker", "127.0.0.1:4917", "ip:port of the tracker")
	rpcPort := flag.Int("port", 1984, "port to run the RPC server on")
	dataPath := flag.String("data-path", getDataPath(), "path to the data directory. CAS storage will be placed under 'storage' subdirectory. If empty, CAS is disabled.")
	casPort := flag.Int("cas-port", 1985, "port to run the CAS server on")
	rpcCommand := flag.String("cmd", "rpc-server", "command to run the RPC server")
	nickname := flag.String("nickname", "", "nickname for the node")
	flag.Parse()

	if *id == "" {
		config, err := LoadOrCreateConfig()
		if err != nil {
			log.Fatalf("Failed to load or create config: %v", err)
		}
		*id = config.ID
		log.Printf("Generated new ID: %s", *id)
	}

	args := []string{
		"--port", fmt.Sprint(*rpcPort),
	}
	args = append(args, flag.Args()...)

	// start RPC server
	cmd := exec.Command(*rpcCommand, args...)
	cmd.Stdin = nil
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		panic(err)
	}

	// print command
	log.Printf("Running command: %s %v\n", *rpcCommand, args)

	if *dataPath != "" {
		// start CAS server
		log.Printf("Data directory: %s", *dataPath)
		casStoragePath := *dataPath + "/storage"
		cas := fscas.NewCAS(casStoragePath)
		go func() {
			log.Printf("Starting CAS server on %s", fmt.Sprintf("0.0.0.0:%d", *casPort))
			if err := openapi.NewRouter(cas).Run(fmt.Sprintf("0.0.0.0:%d", *casPort)); err != nil {
				log.Fatal(err)
			}
		}()
	} else {
		log.Println("CAS is disabled (no data directory specified)")
	}

	// start announcement loop
	go func() {

		query := make(url.Values)
		query.Add("id", *id)
		query.Add("port", fmt.Sprint(*rpcPort))
		query.Add("max_size", fmt.Sprint(getTotalMemoryBytes()))

		if *dataPath != "" {
			query.Add("storage_port", fmt.Sprint(*casPort))
		}

		if *nickname != "" {
			query.Add("nickname", fmt.Sprintf(*nickname))
		}

		announceUrl := url.URL{
			Scheme:   "http",
			Host:     *tracker,
			Path:     "/announce",
			RawQuery: query.Encode(),
		}

		for {
			// send announce request
			log.Printf("Announcing: %s", announceUrl.String())
			resp, err := http.Get(announceUrl.String())
			if err != nil {
				log.Printf("Failed to announce to tracker: %v\n", err)
				time.Sleep(retrySleep)
				continue
			}

			// parse response
			var response announcementResponse
			if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
				log.Printf("Failed to parse announcement response: %v\n", err)
				time.Sleep(retrySleep)
				continue
			}

			log.Printf("Announced to server, reannouncing in %v seconds\n", response.Interval)

			// wait for next announcement time
			time.Sleep(time.Duration(response.Interval * float64(time.Second)))
		}
	}()

	cmd.Wait()
}

type announcementResponse struct {
	Interval float64 `json:"interval"`
}

func getTotalMemoryBytes() uint64 {
	var sysinfo syscall.Sysinfo_t
	syscall.Sysinfo(&sysinfo)
	return sysinfo.Totalram
}
