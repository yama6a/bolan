package main

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func printstdout() {
	println("Hello World!")
	for i := 1; i <= 10; i++ {
		time.Sleep(time.Second * 1)
		println("I'm alive for " + strconv.Itoa(i) + " seconds...")
	}
	println("Good-Bye World!")
}

func main() {
	go printstdout()

	http.HandleFunc("/", healthy)
	http.HandleFunc("/healthz", healthy)
	http.HandleFunc("/ping", pong)
	http.ListenAndServe(":8090", nil)
}

func healthy(w http.ResponseWriter, req *http.Request) {
	fmt.Fprintf(w, "healthy... - "+time.Now().Local().String()+"\n")
	fmt.Fprintf(w, "user-agent: "+req.Header.Get("User-Agent")+"\n")
	println(formatRequest(req))
}

func pong(w http.ResponseWriter, req *http.Request) {
	fmt.Fprintf(w, "pong - "+time.Now().Local().String())
}

func formatRequest(r *http.Request) string {
	request := []string{
		".",
		"\n##### Request " + time.Now().Local().String() + " #####",
		fmt.Sprintf("# %v %v %v", r.Method, r.URL, r.Proto),
		fmt.Sprintf("# Host: %v", r.Host),
		"#",
	}

	// Loop through headers
	for name, headers := range r.Header {
		name = strings.ToLower(name)
		for _, h := range headers {
			request = append(request, fmt.Sprintf("# %v: %v", name, h))
		}
	}

	request = append(request, "##### Request "+time.Now().Local().String()+" #####\n", ".")

	return strings.Join(request, "\n")

}
