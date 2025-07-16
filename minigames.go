package main

import (
    "math/rand"
    "time"
)

// サイコロを2個投げて合計値を返す関数
// ゾロ目の場合は合計値の2倍を返す
func rollDice() int {
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
    
    return sum
}