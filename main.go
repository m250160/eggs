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
	Name          string
	FeedCount     int
	Stage         int
	Status        string
	Generation    int
	IsSick        int
	Money         int
	MinigamePlays int
	FoodHistory map[string]int
	IsMinigame    bool // ミニゲームをプレイ中かどうか
	mu            sync.Mutex
}

// Grave 構造体：死んだペットの記録
type Grave struct {
	Name       string `json:"name"`
	Stage      int    `json:"stage"`
	Generation int    `json:"generation"`
}

// グローバル変数としてペットの状態を管理
var egg = &Pet{
	Status:        "egg",
	Stage:         0,
	FeedCount:     0,
	Generation:    1,
	Money:         0,
	MinigamePlays: 0,
	IsMinigame: false,
	FoodHistory:   make(map[string]int),
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
	_ = os.Remove(graveyardFile)
	http.HandleFunc("/", statusHandler)
	http.HandleFunc("/name", nameHandler)
	http.HandleFunc("/feed/", feedHandler)
	http.HandleFunc("/next", nextHandler)
	http.HandleFunc("/graveyard", graveyardHandler)
	http.HandleFunc("/reset_graveyard", resetGraveyardHandler)
	http.HandleFunc("/images/", imageHandler)
	http.HandleFunc("/Audio/", audioHandler)
	http.HandleFunc("/minigame", minigameHandler)
	http.HandleFunc("/heal", healHandler)
	http.HandleFunc("/self_destruct", selfDestructHandler)
	log.Println("起動 → http://localhost:8090")
	if err := http.ListenAndServe(":8090", nil); err != nil {
		log.Fatalf("サーバー起動失敗: %v", err)
	}
}

func selfDestructHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	egg.mu.Lock()
	defer egg.mu.Unlock()

	if egg.Status != "dead" {
		egg.Status = "dead"
		saveToGraveyard(egg.Name, egg.Stage, egg.Generation)
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// statusHandler：メイン画面を表示
func statusHandler(w http.ResponseWriter, r *http.Request) {
	egg.mu.Lock()
	defer egg.mu.Unlock()

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintln(w, `<!DOCTYPE html><html><head><meta charset="UTF-8"><title>eggっち</title></head><body>`)

	if egg.IsMinigame == false {
		fmt.Fprintln(w, `<audio id="bgm" loop autoplay volume="0.5"><source src="/Audio/BGM/chiptune_sounds.mp3" type="audio/mpeg">お使いのブラウザはaudio要素をサポートしていません。</audio>`)
	} else {
		fmt.Fprintln(w, `<audio id="bgm" loop autoplay volume = "0.0"></audio>`)
	}

	egg.IsMinigame = false

	if egg.IsSick > 0 && egg.Status != "dead" {
		fmt.Fprintf(w, `<p style="color:red;">🤒 病気レベル %d：このまま成長すると死亡します！</p>`, egg.IsSick)
		cost := egg.IsSick * 10
		fmt.Fprintf(w, `<form action="/heal" method="POST"><input type="submit" value="治療する（%dぐっち）"></form>`, cost)
	}
	fmt.Fprintf(w, `<p>所持金: %d ぐっち</p>`, egg.Money)

	if egg.Status == "dead" {
		fmt.Fprintf(w, `<h2>%s は天に召されました🙏</h2><img src="/images/dead.png" alt="死んだeggっち" style="width:200px;height:200px;"><p>世代: 第%d世代</p><p>最終ステージ: %s</p><form action="/next" method="POST"><input type="submit" value="次の卵を生む"></form><a href="/graveyard">過去のeggっちたち</a>`, egg.Name, egg.Generation, stageNames[egg.Stage])
		return
	}

	if egg.Status != "dead" {
		fmt.Fprintln(w, `<form method="POST" action="/self_destruct"><input type="submit" value="💣 自爆する"></form>`)
	}

	if egg.Name == "" {
		fmt.Fprintf(w, `<h2>第%d世代の新しい命が誕生！名前をつけてね</h2><img src="/images/egg.png" alt="たまご" style="width:200px;height:200px;"><form action="/name" method="POST"><input type="text" name="name" required><input type="submit" value="決定"></form>`, egg.Generation)
		return
	}

	var baseImageName string
	switch egg.Stage {
	case 1: baseImageName = "baby"
	case 2: baseImageName = "child"
	case 3: // adult
		maxFood := ""
		maxCount := 0
		for food, count := range egg.FoodHistory {
			if count > maxCount {
				maxCount = count
				maxFood = food
			}
		}

		switch maxFood {
		case "ramen":
			baseImageName = "fat_adult"
		case "liver":
			baseImageName = "muscle_adult"
		default:
			baseImageName = "adult"
		}
	case 4: baseImageName = "elderly"
	default: baseImageName = "egg"
	}
	imageName := baseImageName + ".png"
	if egg.IsSick > 0 {
		imageName = baseImageName + "_sick.png"
	}

	fmt.Fprintf(w, `<h2>第%d世代 %s：%s</h2><img src="/images/%s" alt="%s" style="width:200px;height:200px;"><p>満腹度: %d / 5</p><h3>🍽️ 餌をあげる</h3>`, egg.Generation, egg.Name, stageNames[egg.Stage], imageName, stageNames[egg.Stage], egg.FeedCount%5)

	for _, key := range foodOrder {
		price := foodPrices[key]
		fmt.Fprintf(w, `<form action="/feed/%s" method="POST" style="display:inline; margin:5px;"><input type="submit" value="%s（%dぐっち）"></form>`, key, foods[key], price)
	}

	// ★ポップアップを開くボタンに変更
	fmt.Fprintln(w, `<hr><h3>🎲 ミニゲーム</h3><button onclick="window.open('/minigame', 'minigame', 'width=450,height=350');">サイコロを振ってお金を稼ぐ</button>`)
	remainingPlays := 3 - egg.MinigamePlays
	if remainingPlays < 0 {
		remainingPlays = 0
	}
	fmt.Fprintf(w, `<p>（この形態ではあと %d 回遊べます）</p>`, remainingPlays)
	fmt.Fprintln(w, `<p><a href="/graveyard">過去のeggっちたちを見る</a></p></body></html>`)
}

// minigameHandler：ミニゲームを実行して結果をポップアップに表示
func minigameHandler(w http.ResponseWriter, r *http.Request) {
	egg.mu.Lock()
	defer egg.mu.Unlock()

	egg.IsMinigame = true

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	// ★ロジックを修正：ゲーム実行前に回数制限をチェック
	if egg.MinigamePlays >= 3 {
		fmt.Fprintln(w, `<!DOCTYPE html><html><head><title>ミニゲーム</title></head><body>`)
		// ミニゲーム用BGM（音量調整）
		fmt.Fprintln(w, `<audio id="minigame-bgm" loop autoplay volume="0.5"><source src="/Audio/BGM/dice_game.mp3" type="audio/mpeg">お使いのブラウザはaudio要素をサポートしていません。</audio>`)
		fmt.Fprintln(w, `<h2>お知らせ</h2><p>この形態ではもう遊べません！</p><button onclick="window.close()">閉じる</button></body></html>`)
		return
	}

	// 回数を増やしてゲームを実行
	egg.MinigamePlays++
	dice1, dice2, reward := RollDice()
	egg.Money += reward

	// ★ポップアップ用のHTMLとJavaScriptを返す
	fmt.Fprintln(w, `<!DOCTYPE html><html><head><title>ミニゲーム結果</title></head><body>`)
	// fmt.Fprintln(w, `<audio id="bgm" loop controls autoplay muted> </audio>`)
	// ミニゲーム用BGM（音量調整）

	fmt.Fprintln(w, `<audio id="minigame-bgm" loop autoplay volume="0.5"><source src="/Audio/BGM/dice_game.mp3" type="audio/mpeg">お使いのブラウザはaudio要素をサポートしていません。</audio>`)
	fmt.Fprintf(w, "<h2>🎲 結果は... %d と %d！</h2>", dice1, dice2)
	if dice1 == dice2 {
		fmt.Fprintf(w, `<p style="color:red; font-weight:bold;">ゾロ目ボーナス！</p>`)
	}
	fmt.Fprintf(w, "<p><strong>%d ぐっち</strong> を手に入れた！</p>", reward)
	// fmt.Fprintln(w, `<hr><a href="/minigame">もう一度挑戦！</a><br><br><button onclick="window.close()">閉じる</button>`)

	// ★重要：親ウィンドウをリロードさせるJavaScript
	fmt.Fprintln(w, `<script>if (window.opener) { window.opener.location.reload(); }</script>`)
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
	price, ok := foodPrices[food]
	if !ok { price = 10 }
	if egg.Money < price {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, `<script>alert("所持金が足りません！"); window.location.href = "/";</script>`)
		return
	}
	egg.Money -= price

	egg.FoodHistory[food]++

	var sickChance float64
	switch food {
	case "ramen": sickChance = 0.15
	case "cake": sickChance = 0.1
	case "salad": sickChance = 0.2
	case "onigiri": sickChance = 0.3
	case "liver": sickChance = 0.7
	}

	if rand.Float64() < sickChance {
		if egg.IsSick < 3 { egg.IsSick++ }
	}
	growth, ok := foodGrowth[food]
	if !ok { growth = 1 }
	// 食べる
	egg.FeedCount += growth

	// 成長処理（病気でないことが前提）
	for egg.FeedCount >= 5 {
		if egg.IsSick > 0 {
			egg.Status = "dead"
			saveToGraveyard(egg.Name, egg.Stage, egg.Generation)
			break
		}
		if egg.Stage < len(stageNames)-1 {
			egg.Stage++
			egg.FeedCount = 0
			egg.MinigamePlays = 0

			// 分岐進化条件（大人になるときのみ）
			if egg.Stage == 3 {
				maxFood := ""
				maxCount := 0
				for food, count := range egg.FoodHistory {
					if count > maxCount {
						maxCount = count
						maxFood = food
					}
				}
				switch maxFood {
				case "ramen":
					egg.Status = "fat_adult"
				case "liver":
					egg.Status = "muscle_adult"
				default:
					egg.Status = stageNames[egg.Stage] // 通常大人
				}
			} else {
				egg.Status = stageNames[egg.Stage]
			}
		} else {
			egg.Status = "dead"
			saveToGraveyard(egg.Name, egg.Stage, egg.Generation)
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
	egg.MinigamePlays = 0
	egg.FoodHistory = make(map[string]int)
	egg.mu.Unlock()
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// saveToGraveyard：死んだペットをJSONファイルに保存
func saveToGraveyard(name string, stage int, generation int) {
	var graves []Grave
	_ = loadJSON(&graves)
	graves = append(graves, Grave{Name: name, Stage: stage, Generation: generation})
	data, _ := json.MarshalIndent(graves, "", "  ")
	_ = os.WriteFile(graveyardFile, data, 0644)
}

// graveyardHandler：墓地の一覧を表示
func graveyardHandler(w http.ResponseWriter, r *http.Request) {
	var graves []Grave
	_ = loadJSON(&graves)
	sort.Slice(graves, func(i, j int) bool { return graves[i].Generation < graves[j].Generation })
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintln(w, `<!DOCTYPE html><html><head><meta charset="UTF-8"><title>墓地</title></head><body><h2>過去のeggっちたち</h2><ul>`)
	for _, g := range graves {
		fmt.Fprintf(w, `<li>第%d世代 %s（%s）</li>`, g.Generation, g.Name, stageNames[g.Stage])
	}
	fmt.Fprintln(w, `</ul><form action="/reset_graveyard" method="POST" onsubmit="return confirm('本当に墓地データを消去しますか？');"><input type="submit" value="墓地をリセット"></form><a href="/">戻る</a></body></html>`)
}

func healHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	egg.mu.Lock()
	defer egg.mu.Unlock()

	if egg.IsSick == 0 {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	cost := egg.IsSick * 10
	if egg.Money < cost {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, `<script>alert("所持金が足りません！"); window.location.href = "/";</script>`)
		return
	}

	// 治療前の病気画像を取得
	var baseImageName string
	switch egg.Stage {
	case 1: baseImageName = "baby"
	case 2: baseImageName = "child"
	case 3: // adult
		maxFood := ""
		maxCount := 0
		for food, count := range egg.FoodHistory {
			if count > maxCount {
				maxCount = count
				maxFood = food
			}
		}

		switch maxFood {
		case "ramen":
			baseImageName = "fat_adult"
		case "liver":
			baseImageName = "muscle_adult"
		default:
			baseImageName = "adult"
		}
	case 4: baseImageName = "elderly"
	default: baseImageName = "egg"
	}
	treatmentImageName := baseImageName + "_treatment.png"
	treatedImageName := baseImageName + "_treatmented.png" // 治療完了後は健康な画像
	
	egg.Money -= cost
	egg.IsSick = 0

	// 治療完了ポップアップを表示
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintln(w, `<!DOCTYPE html><html><head><meta charset="UTF-8"><title>治療完了</title></head><body>`)
	fmt.Fprintln(w, `<audio id="bgm" loop autoplay volume="0.5"><source src="/Audio/BGM/chiptune_sounds.mp3" type="audio/mpeg">お使いのブラウザはaudio要素をサポートしていません。</audio>`)
	fmt.Fprintln(w, `<h2 id="title">🏥 治療中・・・</h2>`)
	fmt.Fprintf(w, `<img id="treatmentImage" src="/images/%s" alt="治療中" style="width:200px;height:200px;">`, treatmentImageName)
	fmt.Fprintln(w, `<p id="message">病気を治療しています・・・</p>`)
	fmt.Fprintf(w, `<p>治療費: %dぐっち</p>`, cost)
	fmt.Fprintln(w, `<script>
		// 2秒後に治療完了に切り替え
		setTimeout(function() {
			document.getElementById('title').innerHTML = '✨ 治療完了！';
			document.getElementById('treatmentImage').src = '/images/` + treatedImageName + `';
			document.getElementById('treatmentImage').alt = '治療完了';
			document.getElementById('message').innerHTML = '病気が完全に治りました！';
		}, 2000);
		
		// さらに2秒後にメイン画面に戻る
		setTimeout(function() {
			window.location.href = "/";
		}, 4000);
	</script>`)
	fmt.Fprintln(w, `</body></html>`)
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
	if err != nil { return nil }
	return json.Unmarshal(data, target)
}

// imageHandler：画像ファイルを配信
func imageHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "."+r.URL.Path)
}

// audioHandler：音声ファイルを配信
func audioHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "."+r.URL.Path)
}