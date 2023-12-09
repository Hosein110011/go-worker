package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	probing "github.com/prometheus-community/pro-bing"
)

// type Server struct {
// 	conns map[*websocket.Conn]bool
// }

// func NewServer() *Server {
// 	return &Server{
// 		conns: make(map[*websocket.Conn]bool),
// 	}
// }

// func (s *Server) handleWS(ws *websocket.Conn) {
// 	fmt.Println("new incomming connection from client: ", ws.RemoteAddr())

// 	s.conns[ws] = true

// 	s.readLoop(ws)
// }

// func (s *Server) readLoop(ws *websocket.Conn) {
// 	buf := make([]byte, 1024)
// 	for {
// 		n, err := ws.Read(buf)
// 		if err != nil {
// 			if err == io.EOF {
// 				break
// 			}
// 			fmt.Println("read error:", err)
// 			continue
// 		}
// 		msg := buf[:n]
// 		fmt.Println(string(msg))

// 		ws.Write([]byte("thank u for the message!!!"))
// 	}
// }

// func main() {
// 	server := NewServer()
// 	http.Handle("/ws", websocket.Handler(server.handleWS))
// 	http.ListenAndServe(":3000", nil)
// }

type PingResult struct {
	RTTAvg          float64 `json:"rtt_avg"`
	Destination     string  `json:"destination"`
	PacketLossCount float64 `json:"packet_loss_count"`
}

type DataCenterResult struct {
	Data       PingResult `json:"data"`
	DataCenter string     `json:"data_center"`
	Type       string     `json:"type"`
}

func main() {

	http.HandleFunc("/", indexHandler)
	http.HandleFunc("/upload", uploadHandler)

	// ch := make(chan string)
	ip1 := PingResult{
		Destination: "8.8.8.8",
	}
	ip2 := PingResult{
		Destination: "1.1.1.1",
	}
	ips1 := DataCenterResult{
		DataCenter: "dffgd",
		Type:       "dfdfg",
		Data:       ip1,
	}
	ips2 := DataCenterResult{
		DataCenter: "zzzzz",
		Type:       "zzzzzz",
		Data:       ip2,
	}
	ips := []DataCenterResult{ips1, ips2}

	for _, ip := range ips {
		go func(ip string) {
			// for true {
			pingResult := <-getPing(ip)
			result := DataCenterResult{
				Data:       pingResult,
				DataCenter: "your_data_center", // Replace with your actual data center
				Type:       "your_type",        // Replace with your actual type
			}
			jsonStr, err := json.Marshal(result)
			if err != nil {
				panic(err)
			}
			fmt.Println(string(jsonStr), result)
			time.Sleep(1 * time.Second)

			// }
		}(ip)
	}
	// fmt.Println(<-ch)
	port := 8001
	serverAddr := fmt.Sprintf(":%d", port)
	fmt.Printf("Server listening on http://localhost%s\n", serverAddr)
	err := http.ListenAndServe(serverAddr, nil)
	if err != nil {
		fmt.Println("Error:", err)
	}
	select {}

}

func getPing(dataCenter DataCenterResult) chan DataCenterResult {
	ch := make(chan DataCenterResult)
	ipData := dataCenter.Data
	ip := ipData.Destination
	for true {
		go func() {

			pinger, err := probing.NewPinger(ip)
			if err != nil {
				panic(err)
			}
			pinger.Count = 1
			err = pinger.Run() // Blocks until finished.
			if err != nil {
				panic(err)
			}
			stats := pinger.Statistics()
			// s := fmt.Sprintf("packet lost: %v , packet receive: %v , total packet: %v , ttlavg: %v", stats.PacketLoss, stats.PacketsRecv, stats.PacketsSent, stats.AvgRtt)
			ipData.PacketLossCount = stats.PacketLoss
			ipData.RTTAvg = stats.AvgRtt.Seconds()
			// result := PingResult{
			// 	RTTAvg:          stats.AvgRtt.Seconds(),
			// 	Destination:     ip,
			// 	PacketLossCount: stats.PacketLoss,
			// }
			result := dataCenter
			ch <- result

		}()
	}
	return ch
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	// Render the HTML form
	tmpl := template.Must(template.New("index").Parse(`
	<!DOCTYPE html>
	<html>
	<head>
		<title>File Upload</title>
	</head>
	<body>
		<form action="/upload" method="post" enctype="multipart/form-data">
			<input type="file" name="file" accept=".txt">
			<input type="submit" value="Upload">
		</form>
	</body>
	</html>`))
	tmpl.Execute(w, nil)
}

func uploadHandler(w http.ResponseWriter, r *http.Request) {
	// Parse the form data, including the uploaded file
	err := r.ParseMultipartForm(10 << 20) // 10 MB limit
	if err != nil {
		http.Error(w, "Error parsing form", http.StatusBadRequest)
		return
	}

	// Get the file from the form data
	file, handler, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Error retrieving file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Create the "uploads" directory if it doesn't exist
	err = os.MkdirAll("./uploads", os.ModePerm)
	if err != nil {
		http.Error(w, "Error creating directory", http.StatusInternalServerError)
		return
	}

	// Create a new file on the server to store the uploaded file
	dst, err := os.Create(filepath.Join("./uploads", handler.Filename))
	if err != nil {
		http.Error(w, "Error creating file", http.StatusInternalServerError)
		return
	}
	defer dst.Close()

	// Copy the contents of the uploaded file to the new file
	_, err = io.Copy(dst, file)
	if err != nil {
		http.Error(w, "Error copying file", http.StatusInternalServerError)
		return
	}

	scanner := bufio.NewScanner(file)

	// Read lines from the file
	readFile(filepath.Join("./uploads", handler.Filename))

	// Check for scanning errors
	if err := scanner.Err(); err != nil {
		fmt.Println("Error reading file:", err)
	}

	// Respond with a success message
	w.Write([]byte("File uploaded successfully"))
}

func readFile(filePath string) {
	file, err := os.Open(filePath)
	if err != nil {
		fmt.Println("Error opening file:", err)
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lines := []string{}
	for scanner.Scan() {
		line := scanner.Text()
		lines = append(lines, line)
		fmt.Println(line)
	}
	dataCenter := DataCenterResult{}

	for index, line := range lines {
		parts := strings.Split(line, ":")
		ping := PingResult{}
		// Check if there are at least two parts
		if index == 0 {
			if len(parts) >= 2 {
				dataCenter.DataCenter = parts[1]
			} else {
				fmt.Println("Invalid data")
			}
			// Process the data (you can customize this part based on your needs)
			// fmt.Printf("Datacenter: %s, Status: %s\n", datacenter, status)
		}
		if index > 0 {
			if len(parts) >= 2 {
				ping.Destination = parts[0]
				dataCenter.Type = parts[1]
				dataCenter.Data = ping
			}
		} else {
			fmt.Println("Invalid line:", line)
		}

		fmt.Println(dataCenter, ping)

	}

	if err := scanner.Err(); err != nil {
		fmt.Println("Error reading file:", err)
	}
}
