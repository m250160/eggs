package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"sort"
	"sync"
	"time"
)

// initはプログラム開始時に一度だけ呼ばれる
func init() {
	// 乱数のシード（種）を設定
	rand.Seed(time.Now().UnixNano())
}

// Pet 構造体：ペットの状態を管理
type Pet struct {
	Name       string
	FeedCount  int
	Stage      int
	Status     string
	Generation int
	IsSick     int
	Money      int
	mu         sync.Mutex
}

// Grave 構造体：死んだペットの記録
type Grave struct {
	Name       string `json:"name"`
	Stage      int    `json:"stage"`
	FeedCount  int    `json:"feed_count"`
	Generation int    `json:"generation"`
}

// グローバル変数としてペットの状態を管理
var egg = &Pet{
	Status:     "egg",
	Stage:      0,
	FeedCount:  0,
	Generation: 1,
	Money:      15,
}

// 各種設定値
var stageNames = []string{"🥚 たまご", "👶 赤ちゃん", "🧒 子供", "🧑 大人", "👴 高齢者"}
var foods = map[string]string{
	"ramen":   "ラーメン",
	"cake":    "ケーキ",
	"salad":   "サラダ",
	"onigiri": "おにぎり",
	"liver":   "レバ刺し",
}
var foodOrder = []string{"ramen", "cake", "salad", "onigiri", "liver"}
var foodGrowth = map[string]int{
	"ramen":   3,
	"cake":    4,
	"salad":   2,
	"onigiri": 3,
	"liver":   5,
}
var foodPrices = map[string]int{
	"ramen":   12,
	"cake":    20,
	"salad":   8,
	"onigiri": 5,
	"liver":   1,
}

const graveyardFile = "graves.json"

// main関数：プログラムのエントリーポイント
func main() {
	// サーバー起動時に古い墓地ファイルを削除
	_ = os.Remove(graveyardFile)

	// URLと処理関数を紐付ける（ルーティング）
	http.HandleFunc("/", statusHandler)
	http.HandleFunc("/name", nameHandler)
	http.HandleFunc("/feed/", feedHandler)
	http.HandleFunc("/next", nextHandler)
	http.HandleFunc("/graveyard", graveyardHandler)
	http.HandleFunc("/reset_graveyard", resetGraveyardHandler)
	http.HandleFunc("/images/", imageHandler)
	http.HandleFunc("/minigame", minigameHandler) // ★ミニゲーム用のハンドラを追加

	// ログのポート番号を修正
	log.Println("起動 → http://localhost:8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("サーバー起動失敗: %v", err)
	}
}

// statusHandler：メイン画面を表示
func statusHandler(w http.ResponseWriter, r *http.Request) {
	egg.mu.Lock()
	defer egg.mu.Unlock()

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintln(w, `<!DOCTYPE html><html><head><meta charset="UTF-8"><title>eggっち</title></head><body>`)

	if egg.IsSick > 0 {
		fmt.Fprintf(w, `<p style="color:red;">🤒 病気レベル %d：このまま成長すると死亡します！</p>`, egg.IsSick)
	}
	fmt.Fprintf(w, `<p>所持金: %d ぐっち</p>`, egg.Money)

	if egg.Status == "dead" {
		fmt.Fprintf(w, `<h2>%s は天に召されました🙏</h2>
<img src="/images/dead.png" alt="死んだeggっち" style="width:200px;height:200px;">
<p>世代: 第%d世代</p><p>最終ステージ: %s</p><p>食べた回数: %d</p>
<form action="/next" method="POST"><input type="submit" value="次の卵を生む"></form>
<a href="/graveyard">過去のeggっちたち</a>`, egg.Name, egg.Generation, stageNames[egg.Stage], egg.FeedCount)
		return
	}

	if egg.Name == "" {
		fmt.Fprintf(w, `<h2>第%d世代の新しい命が誕生！名前をつけてね</h2>
<img src="/images/egg.png" alt="たまご" style="width:200px;height:200px;">
<form action="/name" method="POST"><input type="text" name="name" required><input type="submit" value="決定"></form>`, egg.Generation)
		return
	}

	// ★画像表示ロジックを簡潔化
	var baseImageName string
	switch egg.Stage {
	case 1:
		baseImageName = "baby"
	case 2:
		baseImageName = "child"
	case 3:
		baseImageName = "adult"
	case 4:
		baseImageName = "elderly"
	default:
		baseImageName = "egg"
	}

	imageName := baseImageName + ".png"
	if egg.IsSick > 0 {
		imageName = baseImageName + "_sick.png"
	}

	fmt.Fprintf(w, `<h2>第%d世代 %s：%s</h2>
<img src="/images/%s" alt="%s" style="width:200px;height:200px;">
<p>食べた回数: %d / 5</p>
<h3>🍽️ 餌をあげる</h3>`, egg.Generation, egg.Name, stageNames[egg.Stage], imageName, stageNames[egg.Stage], egg.FeedCount%5)

	for _, key := range foodOrder {
		price := foodPrices[key]
		fmt.Fprintf(w, `<form action="/feed/%s" method="POST" style="display:inline; margin:5px;"><input type="submit" value="%s（%dぐっち）"></form>`, key, foods[key], price)
	}

	// ★ミニゲームへのリンクを追加
	fmt.Fprintln(w, `<hr><h3>🎲 ミニゲーム</h3><p><a href="/minigame">サイコロを振ってお金を稼ぐ</a></p>`)

	fmt.Fprintln(w, `<p><a href="/graveyard">過去のeggっちたちを見る</a></p></body></html>`)
}

// minigameHandler：ミニゲームを実行して結果を表示
func minigameHandler(w http.ResponseWriter, r *http.Request) {
	egg.mu.Lock()
	dice1, dice2, reward := RollDice()
	egg.Money += reward
	egg.mu.Unlock()

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintln(w, `<!DOCTYPE html><html><head><title>ミニゲーム結果</title></head><body>`)
	fmt.Fprintf(w, "<h2>🎲 結果は... %d と %d！</h2>", dice1, dice2)
	if dice1 == dice2 {
		fmt.Fprintf(w, `<p style="color:red; font-weight:bold;">ゾロ目ボーナス！</p>`)
	}
	fmt.Fprintf(w, "<p><strong>%d ぐっち</strong> を手に入れた！</p>", reward)
	fmt.Fprintln(w, `<p><a href="/minigame">もう一度挑戦！</a></p>`)
	fmt.Fprintln(w, `<p><a href="/">ゲームに戻る</a></p>`)
	fmt.Fprintln(w, `</body></html>`)
}


// nameHandler：ペットに名前をつける
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

// feedHandler：餌をあげてペットを成長させる
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

	food := r.URL.Path[len("/feed/"):]
	
	// ★不要なrand.Seed呼び出しを削除

	price, ok := foodPrices[food]
	if !ok {
		price = 10
	}

	if egg.Money < price {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, `<script>alert("所持金が足りません！"); window.location.href = "/";</script>`)
		return
	}
	egg.Money -= price

	var sickChance float64
	switch food {
	case "ramen":
		sickChance = 0.15
	case "cake":
		sickChance = 0.1
	case "salad":
		sickChance = 0.2
	case "onigiri":
		sickChance = 0.3
	case "liver":
		sickChance = 0.9
	}

	if rand.Float64() < sickChance {
		if egg.IsSick < 3 {
			egg.IsSick++
		}
	}

	growth, ok := foodGrowth[food]
	if !ok {
		growth = 1
	}
	egg.FeedCount += growth

	for egg.FeedCount >= 5 {
		if egg.IsSick > 0 {
			egg.Status = "dead"
			saveToGraveyard(egg.Name, egg.Stage, egg.FeedCount, egg.Generation)
			break
		} else if egg.Stage < len(stageNames)-1 {
			egg.Stage++
			egg.Status = stageNames[egg.Stage]
			egg.FeedCount -= 5
		} else {
			egg.Status = "dead"
			saveToGraveyard(egg.Name, egg.Stage, egg.FeedCount, egg.Generation)
			break
		}
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// nextHandler：次の世代のペットを準備
func nextHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	egg.mu.Lock()
	egg.Generation++
	egg.Name = ""
	egg.Stage = 0
	egg.Status = "egg"
	egg.FeedCount = 0
	egg.IsSick = 0
	egg.mu.Unlock()
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// saveToGraveyard：死んだペットをJSONファイルに保存
func saveToGraveyard(name string, stage int, feedCount int, generation int) {
	var graves []Grave
	_ = loadJSON(&graves)
	graves = append(graves, Grave{Name: name, Stage: stage, FeedCount: feedCount, Generation: generation})
	data, _ := json.MarshalIndent(graves, "", "  ")
	_ = os.WriteFile(graveyardFile, data, 0644)
}

// graveyardHandler：墓地の一覧を表示
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
		fmt.Fprintf(w, `<li>第%d世代 %s（%s） 食べた回数: %d</li>`, g.Generation, g.Name, stageNames[g.Stage], g.FeedCount)
	}
	fmt.Fprintln(w, `</ul><form action="/reset_graveyard" method="POST" onsubmit="return confirm('本当に墓地データを消去しますか？');">
<input type="submit" value="墓地をリセット"></form>
<a href="/">戻る</a></body></html>`)
}

// resetGraveyardHandler：墓地データをリセット
func resetGraveyardHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/graveyard", http.StatusSeeOther)
		return
	}
	_ = os.WriteFile(graveyardFile, []byte("[]"), 0644)
	http.Redirect(w, r, "/graveyard", http.StatusSeeOther)
}

// loadJSON：JSONファイルからデータを読み込む
func loadJSON(target interface{}) error {
	data, err := os.ReadFile(graveyardFile)
	if err != nil {
		return nil // ファイルがなくてもエラーにしない
	}
	return json.Unmarshal(data, target)
}

// imageHandler：画像ファイルを配信
func imageHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "."+r.URL.Path)
}