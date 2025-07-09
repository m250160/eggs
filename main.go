package main

import (
	"fmt"
	"log"
	"net/http"
	"sync"
)

type Pet struct {
	ClickCount int
	Status     string
	mu         sync.Mutex
}

var egg = &Pet{
	ClickCount: 0,
	Status:     "egg",
}

func main() {
	http.HandleFunc("/", statusHandler)
	http.HandleFunc("/feed", feedHandler)
	http.HandleFunc("/reset", resetHandler)

	log.Println("起動中 → http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

// 現在の状態を表示（GET）
func statusHandler(w http.ResponseWriter, r *http.Request) {
	egg.mu.Lock()
	defer egg.mu.Unlock()

	fmt.Fprintln(w, `<!DOCTYPE html><html><head><meta charset="UTF-8"><title>eggっち</title></head><body>`)

	if egg.Status == "dead" {
		fmt.Fprintln(w, `
<h2>✝️ eggっちは死んでしまった ✝️</h2>

<form action="/reset" method="POST">
    <input type="submit" value="🔄 リセットする">
</form>
</body></html>
`)
		return
	}

	fmt.Fprintf(w, `
<h2>🐣 現在の状態: %s</h2>

<form action="/feed" method="POST">
    <input type="submit" value="🍚 餌をあげる">
</form>

<form action="/reset" method="POST" style="margin-top:10px;">
    <input type="submit" value="🔄 リセット">
</form>

</body></html>
`, egg.Status)
}

// 餌をあげて成長（POST）
func feedHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	egg.mu.Lock()
	defer egg.mu.Unlock()

	if egg.Status == "dead" {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	egg.ClickCount++
	updateStatus()
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// 状態リセット（POST）
func resetHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	egg.mu.Lock()
	defer egg.mu.Unlock()

	egg.ClickCount = 0
	egg.Status = "egg"
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// 状態をクリック数で更新
func updateStatus() {
	switch {
	case egg.ClickCount >= 12:
		egg.Status = "dead"
	case egg.ClickCount >= 9:
		egg.Status = "old"
	case egg.ClickCount >= 6:
		egg.Status = "child"
	case egg.ClickCount >= 3:
		egg.Status = "baby"
	default:
		egg.Status = "egg"
	}
}
