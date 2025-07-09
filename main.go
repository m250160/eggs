package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sort"
	"sync"
)

type Pet struct {
	Name       string
	FeedCount  int
	Stage      int
	Status     string
	Generation int
	mu         sync.Mutex
}

type Grave struct {
	Name       string `json:"name"`
	Stage      int    `json:"stage"`
	FeedCount  int    `json:"feed_count"`
	Generation int    `json:"generation"`
}

var egg = &Pet{
	Status:     "egg",
	Stage:      0,
	FeedCount:  0,
	Generation: 1,
}

var stageNames = []string{"🥚 たまご", "👶 赤ちゃん", "🧒 子供", "🧑 大人", "👴 高齢者"}
var foods = map[string]string{"ramen": "ラーメン", "cake": "ケーキ", "natto": "納豆", "onigiri": "おにぎり"}
var foodOrder = []string{"ramen", "cake", "natto", "onigiri"}
const graveyardFile = "graves.json"

func main() {
	// サーバ起動時に古い墓地を削除
	_ = os.Remove(graveyardFile)

	http.HandleFunc("/", statusHandler)
	http.HandleFunc("/name", nameHandler)
	http.HandleFunc("/feed/", feedHandler)
	http.HandleFunc("/next", nextHandler)
	http.HandleFunc("/graveyard", graveyardHandler)
	http.HandleFunc("/reset_graveyard", resetGraveyardHandler)

	log.Println("起動 → http://localhost:8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("サーバー起動失敗: %v", err)
	}
}

func statusHandler(w http.ResponseWriter, r *http.Request) {
	egg.mu.Lock()
	defer egg.mu.Unlock()

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintln(w, `<!DOCTYPE html><html><head><meta charset="UTF-8"><title>eggっち</title></head><body>`)

	if egg.Status == "dead" {
		fmt.Fprintf(w, `<h2>%s は天に召されました🙏</h2>
<p>世代: 第%d世代</p>
<p>最終ステージ: %s</p>
<p>食べた回数: %d</p>
<form action="/next" method="POST">
    <input type="submit" value="次の卵を生む">
</form>
<a href="/graveyard">過去のeggっちたち</a>
</body></html>`, egg.Name, egg.Generation, stageNames[egg.Stage], egg.FeedCount)
		return
	}

	if egg.Name == "" {
		fmt.Fprintf(w, `<h2>第%d世代の新しい命が誕生！名前をつけてね</h2>
<form action="/name" method="POST">
<input type="text" name="name" required>
<input type="submit" value="決定">
</form>
</body></html>`, egg.Generation)
		return
	}

	fmt.Fprintf(w, `<h2>第%d世代 %s：%s</h2>
<p>食べた回数: %d / 5</p>
<h3>🍽️ 餌をあげる</h3>
`, egg.Generation, egg.Name, stageNames[egg.Stage], egg.FeedCount%5)

	for _, key := range foodOrder {
		fmt.Fprintf(w, `<form action="/feed/%s" method="POST" style="display:inline;">
<input type="submit" value="%s">
</form>
`, key, foods[key])
	}

	fmt.Fprintln(w, `<p><a href="/graveyard">過去のeggっちたちを見る</a></p>
</body></html>`)
}

func nameHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	name := r.FormValue("name")
	if name == "" {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	egg.mu.Lock()
	egg.Name = name
	egg.Status = stageNames[egg.Stage]
	egg.mu.Unlock()

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func feedHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	egg.mu.Lock()
	defer egg.mu.Unlock()
	if egg.Status == "dead" || egg.Name == "" {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	egg.FeedCount++
	if egg.FeedCount%5 == 0 {
		if egg.Stage < len(stageNames)-1 {
			egg.Stage++
			egg.Status = stageNames[egg.Stage]
		} else {
			egg.Status = "dead"
			saveToGraveyard(egg.Name, egg.Stage, egg.FeedCount, egg.Generation)
		}
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func nextHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	egg.mu.Lock()
	if egg.Name != "" {
		saveToGraveyard(egg.Name, egg.Stage, egg.FeedCount, egg.Generation)
	}
	egg.Generation++
	egg.Name = ""
	egg.Stage = 0
	egg.Status = "egg"
	egg.FeedCount = 0
	egg.mu.Unlock()

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func saveToGraveyard(name string, stage int, feedCount int, generation int) {
	var graves []Grave
	_ = loadJSON(&graves)
	graves = append(graves, Grave{Name: name, Stage: stage, FeedCount: feedCount, Generation: generation})
	data, _ := json.MarshalIndent(graves, "", "  ")
	_ = os.WriteFile(graveyardFile, data, 0644)
}

func graveyardHandler(w http.ResponseWriter, r *http.Request) {
	var graves []Grave
	_ = loadJSON(&graves)

	sort.Slice(graves, func(i, j int) bool {
		return graves[i].Generation < graves[j].Generation
	})

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintln(w, `<!DOCTYPE html><html><head><meta charset="UTF-8"><title>墓地</title></head><body>`)
	fmt.Fprintln(w, `<h2>過去のeggっちたち</h2><ul>`) 
	for _, g := range graves {
		fmt.Fprintf(w, `<li>第%d世代 %s（%s） 食べた回数: %d</li>
`, g.Generation, g.Name, stageNames[g.Stage], g.FeedCount)
	}
	fmt.Fprintln(w, `</ul><form action="/reset_graveyard" method="POST" onsubmit="return confirm('本当に墓地データを消去しますか？');">
<input type="submit" value="墓地をリセット">
</form>
<a href="/">戻る</a>
</body></html>`)
}

func resetGraveyardHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/graveyard", http.StatusSeeOther)
		return
	}
	_ = os.WriteFile(graveyardFile, []byte("[]"), 0644)
	http.Redirect(w, r, "/graveyard", http.StatusSeeOther)
}

func loadJSON(target interface{}) error {
	data, err := os.ReadFile(graveyardFile)
	if err != nil {
		return nil
	}
	return json.Unmarshal(data, target)
}
