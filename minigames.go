package main

import (
    "math/rand"
    "time"
)

// サイコロを2個投げて合計値を返す関数
// ゾロ目の場合は合計値の2倍を返す
func rollDice() (int, int, int) {
    // 乱数の種を設定
    rand.Seed(time.Now().UnixNano())
    
    // 1〜6の範囲でサイコロを2個投げる
    dice1 := rand.Intn(6) + 1
    dice2 := rand.Intn(6) + 1
    
    sum := dice1 + dice2
    
    // ゾロ目の場合は合計値の2倍を返す
    if dice1 == dice2 {
        return sum * 2
    }
    
    return dice1, dice2, sum
}

// 仮に構造体を用いている

// 丁半博打の結果を表す構造体
type ChouhanResult struct {
    Dice1    int    // 1つ目のサイコロ
    Dice2    int    // 2つ目のサイコロ
    Sum      int    // 合計値
    IsEven   bool   // 偶数（丁）かどうか
    Result   string // "丁" または "半"
}


// 丁半博打を実行する関数
// prediction: true=丁（偶数）, false=半（奇数）
func playChouhanBakuchi(prediction bool) (ChouhanResult, bool) {
    // 乱数の種を設定
    rand.Seed(time.Now().UnixNano())
    
    // 1〜6の範囲でサイコロを2個投げる
    dice1 := rand.Intn(6) + 1
    dice2 := rand.Intn(6) + 1
    
    sum := dice1 + dice2
    isEven := sum%2 == 0
    
    var result string
    if isEven {
        result = "丁"
    } else {
        result = "半"
    }
    
    // 予想が当たったかどうか
    isWin := prediction == isEven
    
    return ChouhanResult{
        Dice1:  dice1,
        Dice2:  dice2,
        Sum:    sum,
        IsEven: isEven,
        Result: result,
    }, isWin
}

// 丁半博打の結果を文字列で返す関数
func getChouhanResultString(result ChouhanResult, prediction bool, isWin bool) string {
    var predictionStr string
    if prediction {
        predictionStr = "丁（偶数）"
    } else {
        predictionStr = "半（奇数）"
    }
    
    var winStr string
    if isWin {
        winStr = "勝ち！"
    } else {
        winStr = "負け..."
    }
    
    return fmt.Sprintf(
        "🎲 サイコロ: %d, %d\n合計: %d\n結果: %s\n予想: %s\n%s",
        result.Dice1, result.Dice2, result.Sum, result.Result, predictionStr, winStr,
    )
}