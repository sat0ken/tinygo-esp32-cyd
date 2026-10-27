# tinygo-cyd コード解説

ESP32-4827S043（ESP32-S3 + 4.3インチ 480×272 RGB 液晶）を TinyGo で動かすためのコード一式の解説。
「どこに何があり、なぜそうなっているか」を、今後自分で直せることを目標に説明する。

- 図は Mermaid で書いている。GitHub や VS Code（Markdown Preview Mermaid Support 拡張）で図として表示される。
- リンクはリポジトリ内のファイル（行番号付きのものもある）。行番号は 2026-10 時点のもの。
- 理解度の確認は [QUIZ.md](QUIZ.md)、解答は [QUIZ_ANSWERS.md](QUIZ_ANSWERS.md)（どちらも手元だけに置き、git では管理しない）。

## 目次

1. [用語](#1-用語)
2. [全体像](#2-全体像)
3. [ビルドの切り替え（実機・ブラウザ・ホスト）](#3-ビルドの切り替え実機ブラウザホスト)
4. [液晶表示の仕組み（rgblcd）](#4-液晶表示の仕組みrgblcd)
5. [メモリの使い方](#5-メモリの使い方)
6. [描画 API とバックエンド](#6-描画-api-とバックエンド)
7. [ブラウザ版（wasmlcd）](#7-ブラウザ版wasmlcd)
8. [テスト](#8-テスト)
9. [タッチパネル](#9-タッチパネル)
10. [フラッシュとスライドショー（05）](#10-フラッシュとスライドショー05)
11. [SD カードのスライドショー（06）](#11-sd-カードのスライドショー06)
12. [ゲーム（07〜12）](#12-ゲーム0712)
13. [運用：コマンド・よくある作業・トラブル](#13-運用コマンドよくある作業トラブル)
14. [未解決・注意事項](#14-未解決注意事項)
15. [ESP32-3248S035C（3.5インチ・SPI 液晶）](#15-esp32-3248s035c35インチspi-液晶)
16. [ESP32-2432S028R（2.8インチ・320×240）](#16-esp32-2432s028r28インチ320240)

---

## 1. 用語

| 用語 | 意味 |
|---|---|
| RGB565 | 1画素 16bit の色形式。上位から R 5bit・G 6bit・B 5bit。 |
| RGB565BE | RGB565 を「上位バイト → 下位バイト」の順にバイト列で並べたもの（ビッグエンディアン）。`DrawRGBBitmap8` と画像ファイルがこの形式。 |
| フレームバッファ | 画面全体の画素を入れたメモリ。ここでは `[]uint16`（RGB565、行優先、480×272）。 |
| PCLK | ピクセルクロック。1クロックで1画素を液晶に送る。ここでは 9MHz。 |
| HSYNC / VSYNC | 水平・垂直同期信号。行の始まり・フレームの始まりを液晶に知らせる。 |
| DE | Data Enable。「今送っているのは表示データ」という信号。 |
| ポーチ | 同期パルスの前後の空白期間（front porch / back porch）。 |
| LCD_CAM | ESP32-S3 の液晶・カメラ用ペリフェラル。RGB パラレル出力を作る。 |
| GDMA | ESP32-S3 の汎用 DMA。メモリから LCD_CAM へデータを流す。 |
| ディスクリプタ | DMA に「どのメモリを何バイト送るか、次はどれか」を教える 12 バイトの構造体。 |
| GPIO マトリクス / IO_MUX | ペリフェラルの信号を任意の GPIO ピンにつなぐ仕組み。 |
| DROM / IROM | フラッシュをキャッシュ経由でメモリとして見せるアドレス空間（データ用 0x3C000000〜 / 命令用 0x42000000〜）。 |
| MMU | 上の仮想アドレスとフラッシュの物理位置を対応付ける表。64KB ページ単位。 |
| hal | このリポジトリのアプリが依存するインターフェース（`hal.Display`、`hal.Touch`）。 |
| ゴールデン画像 | 「正しい画面」として保存した PNG。テストで描画結果とピクセル単位で比較する。 |

---

## 2. 全体像

### ディレクトリ

| ディレクトリ | 役割 | 動く場所 |
|---|---|---|
| [board/](../board/board.go) | ピン番号・タイミング・キャリブレーション値など**ボード固有の値をすべて**ここに置く | 全部 |
| [hal/](../hal/hal.go) | アプリが依存するインターフェース `Display`・`Touch` | 全部 |
| [framebuf/](../framebuf/framebuf.go) | RGB565 フレームバッファへの描画（点・矩形・ビットマップ） | 全部 |
| [rgblcd/](../rgblcd/) | **LCD_CAM + GDMA の液晶ドライバ**（このリポジトリの中心） | 実機（計算部分はホストでもテスト） |
| [wasmlcd/](../wasmlcd/wasmlcd.go) | ブラウザの canvas に表示・マウス/タッチを入力にする | ブラウザ |
| [memlcd/](../memlcd/memlcd.go) | メモリ上の表示（テスト用、PNG 出力） | ホスト |
| [platform/](../platform/) | ビルド先に応じて上の3つを選んで初期化 | 全部 |
| [xpttouch/](../xpttouch/) | XPT2046 タッチの読み取り | 実機 |
| [touchcal/](../touchcal/calib.go) | タッチの raw 値 → 画面座標、4隅からのキャリブレーション | 全部 |
| [romfont/](../romfont/romfont.go) | フォントを文字列にしてフラッシュに置く（tinyfont と同じ描画）。`go generate ./romfont` で [tools/fontgen](../tools/fontgen/main.go) が作る | 全部 |
| [spilcd/](../spilcd/spilcd.go) | SPI 液晶（ST7796・ILI9341）のドライバ。フレームバッファなし | 実機（ホストでもテスト） |
| [board/s035/](../board/s035/s035.go)・[gt911/](../gt911/gt911.go)・[viewport/](../viewport/viewport.go) | ESP32-3248S035C 用（[15章](#15-esp32-3248s035c35インチspi-液晶)） | 実機（ドライバはホストでもテスト） |
| [board/s028/](../board/s028/s028.go) | ESP32-2432S028R 用（[16章](#16-esp32-2432s028r28インチ320240)） | 全部 |
| [internal/panelcheck/](../internal/panelcheck/panelcheck.go) | SPI 液晶基板の確認プログラム（s035_check・s028_check）の共通部分 | 実機 |
| [flashmap/](../flashmap/flashmap.go) | フラッシュの任意の領域を MMU でメモリとして読めるようにする | 実機 |
| [slidepack/](../slidepack/slidepack.go) | 複数の全画面画像を1ファイルにまとめる形式 | 全部 |
| [slideshow/](../slideshow/) | スライドショー本体（切り替え効果・キャプション・タッチ操作）と SD 用の読み込み | 全部 |
| [app/](../app/app.go) | お絵描きアプリ（`cmd/app` の中身） | 全部 |
| [games/](../games/) | ゲーム本体（invaders・sudoku・othello・solitaire・poker）とトランプの部品（cards）。hal だけに依存 | 全部 |
| [cmd/app/](../cmd/app/main.go) | 実機・ブラウザ共通のエントリポイント | 全部 |
| [cmd/games/](../cmd/games/main.go) | ゲームを選んで遊ぶランチャー | 全部 |
| [examples/](../examples/) | 01〜08 のサンプル | 例ごとに異なる |
| [internal/](../internal/) | テスト補助（goldentest）、C 用 malloc 補助（espmalloc） | — |
| [tools/](../tools/) | 画像変換（img2rgb565）、HTTP サーバ（serve）、ブラウザ検査（wasmcheck） | ホスト |
| [targets/](../targets/esp32-4827s043.json) | TinyGo のカスタムターゲット | — |
| [web/](../web/index.html) | ブラウザ版の HTML | ブラウザ |
| [testdata/golden/](../testdata/golden/) | ゴールデン画像 | ホスト |

### 依存関係

アプリ（`app`、`slideshow`、ゲーム）は `hal` にしか依存しない。だから同じコードが実機でもブラウザでもテストでも動く。

```mermaid
flowchart TB
    subgraph Apps["アプリ（hal だけに依存）"]
        app["app<br/>お絵描き"]
        ss["slideshow"]
        games["examples/07, 08<br/>ゲーム"]
    end
    hal["hal<br/>Display / Touch"]
    Apps --> hal

    subgraph Backends["hal の実装"]
        rgblcd["rgblcd<br/>実機の液晶"]
        wasmlcd["wasmlcd<br/>canvas"]
        memlcd["memlcd<br/>テスト用"]
        xpt["xpttouch<br/>実機のタッチ"]
    end
    hal -.実装.- rgblcd
    hal -.実装.- wasmlcd
    hal -.実装.- memlcd
    hal -.実装.- xpt
    rgblcd --> framebuf
    wasmlcd --> framebuf
    memlcd --> framebuf

    platform["platform<br/>ビルド先で実装を選ぶ"]
    platform --> rgblcd
    platform --> wasmlcd
    platform --> memlcd
    platform --> xpt
    board["board<br/>ボード固有の値"]
    platform --> board
    cmd["cmd/app, examples"] --> platform
    cmd --> Apps
```

---

## 3. ビルドの切り替え（実機・ブラウザ・ホスト）

同じソースから3種類のプログラムを作る。どのファイルを使うかは **Go のビルドタグ**（ファイル先頭の `//go:build ...`）で決まる。

| ビルド | コマンド | 有効なタグ | `platform.Init()` の実体 |
|---|---|---|---|
| 実機 | `tinygo build -target=./targets/esp32-4827s043.json` | `esp32s3` ほか | [platform_esp32s3.go](../platform/platform_esp32s3.go)：rgblcd + xpttouch |
| ブラウザ | `tinygo build -target=wasm` | `js`, `wasm` | [platform_wasm.go](../platform/platform_wasm.go)：wasmlcd（表示とタッチを兼ねる） |
| ホスト | `go test` / `go run` | （なし） | [platform_other.go](../platform/platform_other.go)：memlcd + 押されないタッチ |

- `machine` パッケージ（GPIO など）を使うファイルは必ず `//go:build esp32s3` を付ける。付けないとホストの `go test ./...` が「package machine is not in std」で失敗する。
- `rgblcd` は「計算だけのファイル（タグなし）」と「レジスタを触るファイル（esp32s3）」に分かれている。計算部分（[calc.go](../rgblcd/calc.go)）はホストでテストできる。

### カスタムターゲット [targets/esp32-4827s043.json](../targets/esp32-4827s043.json)

```json
{
  "inherits": ["esp32s3-generic"],   // machine パッケージが SPI のピン定数を要求するため generic を継承
  "build-tags": ["esp32_4827s043"],
  "serial": "uart",                  // USB-C は CH340C 経由で UART0 につながっている
  "flash-method": "esp32flash",      // UART 経由の書き込み
  "ldflags": ["--defsym=Cache_Invalidate_Addr=0x400016b0", ...]  // flashmap が使う ROM 関数の番地
}
```

---

## 4. 液晶表示の仕組み（rgblcd）

### 4.1 データの流れ

**CPU は画面を「送らない」**。フレームバッファに書くだけで、あとは GDMA と LCD_CAM が毎秒約57回、勝手に液晶へ送り続ける。

```mermaid
flowchart LR
    CPU["CPU<br/>SetPixel / FillRectangle"] -->|書く| FB["フレームバッファ<br/>内部 SRAM 255KB<br/>fbMem"]
    FB -->|読む| DMA["GDMA OUT ch0<br/>68個の循環<br/>ディスクリプタ"]
    DMA -->|16bit ずつ| LCD["LCD_CAM<br/>RGB モード<br/>PCLK 9MHz"]
    LCD -->|DATA0-15<br/>HSYNC VSYNC DE PCLK| MUX["GPIO マトリクス"]
    MUX -->|20本| Panel["液晶パネル<br/>480x272"]
```

- フレームバッファの1画素（`uint16`）の bit0〜15 が、そのまま LCD_CAM の `DATA_OUT0〜15` に出る。
- [board.DataPins()](../board/board.go) が `DATA_OUT0〜4 = B0〜B4`、`5〜10 = G0〜G5`、`11〜15 = R0〜R4` の順にピンを並べる。これで普通の RGB565（上位 5bit が赤）がそのまま正しい色になる。

### 4.2 パネルの信号タイミング

1行 = 4（HSYNC）+ 43（バックポーチ）+ 480（表示）+ 8（フロントポーチ）= **535 PCLK**。
1フレーム = 4 + 12 + 272 + 8 = **296 行**。9MHz ÷ (535 × 296) ≒ **56.83 fps**。

```
1行（PCLK 単位）
HSYNC  ‾‾‾‾|____|‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾|____|‾‾  ← ※実際の極性は下表
            4      43           480（DE=1、データ）          8
DE     ________________________|‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾|_______________
           パルス  バックポーチ           表示            フロントポーチ
```

| 設定（[board.go](../board/board.go)） | 値 | 意味 |
|---|---|---|
| `PclkHz` | 9_000_000 | 不安定なら 8MHz → 7MHz に下げる |
| `HSyncIdleLow` / `VSyncIdleLow` | true | 待機時 Low（Arduino_GFX の polarity 0） |
| `PclkActiveNeg` | true | PCLK の立ち下がりでデータを出す |
| `DEIdleHigh` | false | 待機時 DE は Low |

### 4.3 クロックの計算（[CalcClock](../rgblcd/calc.go#L27)）

ESP-IDF の `lcd_hal_cal_pclk_freq` をそのまま移植したもの。

```
lcd_clk   = ソース ÷ (N + B/A)
pixel_clk = lcd_clk ÷ MO
```

9MHz の場合：ソース PLL_F160M = 160MHz。`PclkActiveNeg` があると `MO = 1` は使えない（極性を変えられないため）ので MO = 2。
160 ÷ 18 = 8 余り 16 → **N=8、A=9、B=8、MO=2**。lcd_clk = 18MHz、pclk = 9MHz ちょうど。
実機のレジスタ値 `LCD_CLOCK=e4901101` はこの値を含む。

### 4.4 タイミングレジスタ（[CalcTiming](../rgblcd/calc.go#L109)）

ESP-IDF `lcd_ll_set_horizontal_timing` / `lcd_ll_set_vertical_timing` の変換式（レジスタには「値 − 1」を書く）。

| フィールド | 式 | 値 |
|---|---|---|
| `LCD_HSYNC_WIDTH` | hsw − 1 | 3 |
| `LCD_HB_FRONT` | hbp + hsw − 1 | 46 |
| `LCD_HA_WIDTH` | 幅 − 1 | 479 |
| `LCD_HT_WIDTH` | hsw + hbp + 幅 + hfp − 1 | 534 |
| `LCD_VSYNC_WIDTH` | vsw − 1 | 3 |
| `LCD_VB_FRONT` | vbp + vsw − 1 | 15 |
| `LCD_VA_HEIGHT` | 高さ − 1 | 271 |
| `LCD_VT_HEIGHT` | vsw + vbp + 高さ + vfp − 1 | 295 |

各フィールドにはビット幅の上限があり（例：`VT_HEIGHT` は 10bit = 1023 まで）、超えると `ErrBadTiming`。

### 4.5 DMA ディスクリプタ（[gdma.go](../rgblcd/gdma.go)、[DescLayout](../rgblcd/calc.go#L173)）

1つのディスクリプタで送れるのは最大 4095 バイト。1行 = 960 バイトなので、**4行（3840 バイト）× 68 個**に分ける。最後のディスクリプタの `next` を先頭に向けて**輪**にする。

```mermaid
flowchart LR
    D0["desc[0]<br/>fb+0<br/>3840B"] --> D1["desc[1]<br/>fb+3840"] --> D2["desc[2]<br/>fb+7680"] --> Dots["…"] --> D67["desc[67]<br/>fb+257280<br/>suc_eof=1"]
    D67 -->|next| D0
```

- ディスクリプタのワード0：`size`（bit0-11）、`length`（bit12-23）、`suc_eof`（bit30）、`owner`（bit31 = DMA）。[DescWord0](../rgblcd/calc.go#L158) が作る。
- `descs` は**パッケージのグローバル変数**。DMA が動いている間に GC で回収されないこと、内部 SRAM にあることが必要なため（`OUTLINK_ADDR` はアドレスの下位 20bit しか持たない）。
- オーナーチェック（`OUT_CHECK_OWNER`）を切っているので、DMA が一周してもディスクリプタを書き戻す必要がない。
- LCD_CAM の `LCD_NEXT_FRAME_EN` を立てているので、1フレーム送り終わると LCD_CAM が自分で次のフレームを要求する。**CPU の介入なしで表示が続く**のはこの2つの組み合わせのため。

### 4.6 GPIO マトリクス（[gpio.go](../rgblcd/gpio.go)）

20本のピンそれぞれに「LCD_CAM のどの信号を出すか」を書く。信号番号は ESP-IDF `gpio_sig_map.h` から。

| 信号 | 番号 | GPIO |
|---|---|---|
| LCD_DATA_OUT0〜15 | 133〜148 | B0-4: 8,3,46,9,1 / G0-5: 5,6,7,15,16,4 / R0-4: 45,48,47,21,14 |
| LCD_H_ENABLE（DE） | 150 | 40 |
| LCD_H_SYNC | 151 | 39 |
| LCD_V_SYNC | 152 | 41 |
| LCD_PCLK | 154 | 42 |

`routePin` は「IO_MUX の機能を GPIO（MCU_SEL=1）に → `FUNCn_OUT_SEL_CFG` に信号番号 → 出力を有効化」の順。

### 4.7 初期化の順序（[New](../rgblcd/display.go#L46)・[Start](../rgblcd/display.go#L109)）

ESP-IDF の `esp_lcd_new_rgb_panel` → `esp_lcd_panel_reset` → `esp_lcd_panel_init` → `lcd_rgb_panel_start_transmission` と同じ順序。

```mermaid
sequenceDiagram
    participant App as platform.InitLCD
    participant D as rgblcd
    participant SYS as SYSTEM
    participant LCD as LCD_CAM
    participant DMA as GDMA
    App->>D: New(cfg, fb)
    D->>D: 設定の検査・計算（CalcClock / CalcTiming / DescLayout）
    D->>SYS: LCD_CAM のクロック有効化・リセット
    D->>LCD: クロック源 PLL160M、FIFO/コントローラのリセット、割り込み無効
    D->>SYS: GDMA のクロック有効化（使用中ならリセットしない）
    D->>DMA: チャネルのリセット、接続先=LCD_CAM(5)、バースト有効
    D->>D: ディスクリプタの循環リストを作る
    D->>D: GPIO マトリクスに 20 本をつなぐ
    D->>LCD: PCLK 分周・極性・RGB モード・タイミング・NEXT_FRAME_EN
    App->>D: Start()
    D->>DMA: リセット → 先頭ディスクリプタで開始（DMA が先）
    D->>D: 1µs 待つ（FIFO を満たす）
    D->>LCD: UPDATE → START（LCD が後）
    App->>App: バックライト 100%
```

### 4.8 デバッグ用の仕組み

- [Dump()](../rgblcd/display.go#L172)：設定した全レジスタを**読み戻し**、計算した期待値と比べて `OK/NG` を表示する。最後に `=== register check: OK ===`。
- [SelfTest()](../rgblcd/clock.go#L65)：LCD_CAM と GDMA のレジスタに値を書いて読み戻せるかを試す（フェーズ1）。
- [examples/02_colorbars](../examples/02_colorbars/main.go)：上の2つを順に実行し、カラーバーを出す。シリアルで `c r g b w k d` を受け付ける。tick 行に `regs OK`、`fps`、DMA が今どのディスクリプタを読んでいるか（`desc:`）を出す。

### 4.9 ティアリング対策（[WaitVSync](../rgblcd/display.go#L150)）

LCD_CAM はフレームを送り終わるたびに `LC_DMA_INT_RAW` の bit0（VSYNC）を立てる。`WaitVSync` はこのビットを消して、また立つまで**ポーリング**で待つ。割り込みハンドラは使っていない。
`Display()` は何もしない（常に表示されているため）。ゲームは毎フレーム `WaitVSync` の直後に描くことで、描きかけの画面が映りにくくしている。

---

## 5. メモリの使い方

ESP32-S3 の内部 SRAM のうち、TinyGo が使えるのは `0x3FC88000〜0x3FCEB710`（約 398KB、上は ROM と Wi-Fi が使う）。この領域は IRAM（命令）と共有。

```
0x3FC88000 ┌────────────────────────┐
           │ スタック 4KB            │
0x3FC89000 ├────────────────────────┤
           │ .bss                   │ ← descs（1.5KB）と fbMem（255KB）はここ
           │   descs  0x3FC890E4〜   │
           │   fbMem  0x3FC896E4〜   │
           ├────────────────────────┤
           │ .data（フォント等 ~10KB）│
           ├────────────────────────┤
           │ IRAM の影（~2.4KB）      │ ← mapPages など IRAM に置いたコード
           ├────────────────────────┤
           │ ヒープ 約 123KB          │ ← make / new / goroutine のスタック
0x3FCEB710 └────────────────────────┘
```

- **フレームバッファはグローバル配列** [`fbMem`](../platform/platform_esp32s3.go#L17)。`[480*272/2]uint32` と宣言しているのは **4 バイト境界に揃えるため**（`uint16` の配列だと 2 バイト境界の可能性がある）。`rgblcd.New` は境界と「DMA が読める範囲（0x3FC88000〜0x3FD00000）にあるか」を検査する。
- PSRAM（8MB）は使っていない。TinyGo の PSRAM 対応が不明なため。
- Wi-Fi（espradio）を足すとヒープが足りなくなる可能性が高い。

---

## 6. 描画 API とバックエンド

### hal.Display と framebuf

[hal.Display](../hal/hal.go) は tinygo drivers の `Displayer`（`Size`、`SetPixel`、`Display`）に `FillRectangle` を足したもの。tinydraw・tinyfont は `Displayer` しか使わないので、そのまま動く。

```mermaid
classDiagram
    class Displayer {
        <<interface tinygo drivers>>
        Size()
        SetPixel(x, y, c)
        Display()
    }
    class Display {
        <<interface hal>>
        FillRectangle(x, y, w, h, c)
    }
    Displayer <|-- Display
    class Buffer {
        <<framebuf>>
        Pix []uint16
        SetPixel / FillRectangle
        DrawRGBBitmap8 / DrawBitmap
    }
    class rgblcd_Device {
        Display() 何もしない
        WaitVSync()
        Dump()
    }
    class wasmlcd_Display {
        Display() dirty を立てる
        ReadTouch()
    }
    class memlcd_Display {
        Display() 回数を数える
        Image() / SavePNG()
    }
    Buffer <|-- rgblcd_Device : 埋め込み
    Buffer <|-- wasmlcd_Display : 埋め込み
    Buffer <|-- memlcd_Display : 埋め込み
    Display <.. rgblcd_Device
    Display <.. wasmlcd_Display
    Display <.. memlcd_Display
```

- 3つのバックエンドは**同じ形式のフレームバッファ**（`[]uint16`、RGB565、行優先）を `framebuf.Buffer` として埋め込んでいる。違うのは「バッファがどう画面に届くか」だけ。
- `FillRectangle` は1行目を埋めてから `copy` で下の行に複製する（ループより速い）。
- `DrawRGBBitmap8` は **RGB565BE のバイト列**を受け取る。画像ファイル（.rgb565、slidepack）とゲームのスプライト描画がこれを使う。

---

## 7. ブラウザ版（wasmlcd）

[wasmlcd.go](../wasmlcd/wasmlcd.go) は canvas に描く。

```mermaid
sequenceDiagram
    participant Go as Go（アプリ）
    participant W as wasmlcd
    participant JS as ブラウザ
    Go->>W: SetPixel / FillRectangle（バッファに書いて dirty=true）
    JS->>W: requestAnimationFrame（毎フレーム）
    W->>W: dirty なら RGB565→RGBA 変換
    W->>JS: CopyBytesToJS → putImageData
    JS->>W: pointerdown / move / up
    W->>W: 座標を 480x272 に換算して保持
    Go->>W: ReadTouch()（最新の座標を返す）
```

- `wasm_exec.js` は **TinyGo 付属のもの**を使う（Go 付属のものは互換性がない）。`make wasm` が毎回コピーする。
- `.wasm` は `application/wasm` の MIME で配信する必要があり、`file://` では動かない。`make serve`（[tools/serve](../tools/serve/main.go)）がそうする。
- `main` から return するとプログラムが終わるので、ループで回し続ける。
- headless Chromium での検査用に、`window.wasmlcdFlush_lcd()` で即座に canvas へ転送できる。headless の仮想時間モードでは requestAnimationFrame が来ないため。

---

## 8. テスト

| 種類 | 場所 | 何を確かめるか |
|---|---|---|
| 計算 | [rgblcd/calc_test.go](../rgblcd/calc_test.go) | PCLK 分周・タイミング値・ディスクリプタを、ESP-IDF の式から手計算した値と比較 |
| ゴールデン画像 | [app](../app/app_test.go)、[05](../examples/05_slideshow/main_test.go)、[07](../examples/07_breakout/game_test.go)、[08](../games/invaders/game_test.go) | memlcd で描いた画面を `testdata/golden/*.png` とピクセル単位で比較 |
| ロジック | 各 `_test.go` | タッチ操作、スライドの切り替え、ゲームの自動プレイ、BMP 変換など |
| ブラウザ | [tools/wasmcheck](../tools/wasmcheck/wasmcheck_test.go) | headless Chromium で canvas を読み、同じゴールデン画像と比較 |

```mermaid
flowchart LR
    T["go test"] --> M["memlcd に描く"] --> I["Image()"] --> C{"goldentest.Check"}
    C -->|一致| OK["合格"]
    C -->|不一致| NG["失敗<br/>name.actual.png を出力"]
    C -->|"-update"| U["ゴールデン画像を上書き<br/>→ 目視確認してからコミット"]
```

- 画面を意図して変えたら `make update-golden` で更新し、**画像を目で見てから**コミットする。
- 実機でしか確かめられないもの：レジスタ設定、RAM 容量、描画速度、ティアリング、液晶の発色、タッチの精度、SD カード、Wi-Fi。

---

## 9. タッチパネル

- XPT2046（抵抗膜）。tinygo drivers の [xpt2046](https://github.com/tinygo-org/drivers/tree/release/xpt2046) を使う。このドライバは **SPI を GPIO で1ビットずつ操作（ビットバング）** しており、読み取りに時間がかかる。
- ピン：SCK=12、MOSI=11、MISO=13、CS=38、INT=18（押されると Low）。SCK/MOSI/MISO は microSD と共有。
- [xpttouch.ReadRaw](../xpttouch/xpttouch.go#L30)：ドライバが返す 16bit 値を 12bit に戻す。[Calibration.Map](../touchcal/calib.go#L18)：raw 値を画面座標に線形変換し、画面内に収める。
- キャリブレーション値は [board.go](../board/board.go) の `TouchRaw*`（実測：X 172〜3940、Y 3884〜329、軸の入れ替えなし）。測り直しは [04_touch](../examples/04_touch/main.go)。
- [FromCorners](../touchcal/calib.go)：4隅のタップから値を求める。軸の入れ替わりを検出し、範囲が狭すぎる・同じ辺の2点が食い違う・向きが逆、のときは失敗にする。

---

## 10. フラッシュとスライドショー（05）

### 10.1 なぜ画像をプログラムに埋め込まないのか

TinyGo の ESP32-S3 は **ESP-IDF の2段目ブートローダを使わず、ROM が直接プログラムを読み込む**。ROM は約 1MB の rodata セグメントを「Invalid image block, can't boot」と拒否した。
そこで、画像は**プログラムとは別にフラッシュの 8MB 位置に書き**、実行時に MMU でメモリとして見えるようにして読む。

### 10.2 フラッシュの配置（16MB）

```
0x000000 ┌──────────────────────────┐
         │ プログラム（数十〜数百KB）  │ ← make flash / make flash-noerase
         │                          │
0x800000 ├──────────────────────────┤ ← board.SlidesFlashOffset
         │ slides.pack（1枚 255KB）   │ ← make flash-slides
         │                          │
0xFFFFFF └──────────────────────────┘
```

| コマンド | 何をするか | スライドは |
|---|---|---|
| `make flash` | `tinygo flash`。**フラッシュ全体を消去**してからプログラムを書く | **消える** |
| `make flash-noerase` | `tinygo build` で .bin を作り、espflasher で 0 番地に書く | 残る |
| `make flash-slides` | espflasher で `slides.pack` を 0x800000 に書く（`-fs 16MB` が必要） | 書かれる |

`-fs 16MB` がないと、espflasher（の書き込み用スタブ）は 4MB までしか書かせず、エラー 0xC4 になる。フラッシュサイズの自動検出はプログラムのイメージ（先頭 0xE9）でしか行われないため。

### 10.3 MMU マッピング（[flashmap.Map](../flashmap/flashmap.go#L70)）

ESP-IDF の `esp_mmu_map` と同じ手順。データキャッシュを止めている間はフラッシュを読めないので、その間に動くコード（[mapPages](../flashmap/flashmap.go#L92)）は `//go:section .iram1...` で **IRAM に置く**。

```mermaid
sequenceDiagram
    participant Go as flashmap.Map
    participant I as mapPages（IRAM）
    participant ROM as ROM 関数
    participant MMU as MMU 表 0x600C5000
    Go->>Go: 割り込み禁止
    Go->>I: 呼ぶ
    I->>ROM: rom_Cache_Suspend_DCache
    I->>MMU: エントリ[page] と [256+page] に page を書く
    I->>ROM: Cache_Invalidate_Addr
    I->>ROM: Cache_Resume_DCache
    Go->>Go: 割り込み復帰
    Go-->>Go: 0x3C000000+offset を []byte として返す
```

- ROM 関数の番地は TinyGo のリンカスクリプトにないので、ターゲットの `ldflags` に `--defsym` で書いている。
- エントリを2か所に書いているのは、ESP-IDF の解釈と TinyGo の起動コードの解釈が違い、どちらが正しいか決められなかったため（[14章](#14-未解決注意事項)）。

### 10.4 slidepack 形式（[slidepack.go](../slidepack/slidepack.go)）

```
0x0000  "SLD1"  幅(u16) 高さ(u16) 枚数(u32) 画像の開始位置(u32=4096) 1枚のサイズ(u32)
0x0014  名前 32 バイト × 枚数
0x1000  画像0（RGB565BE、480×272×2 = 261,120 バイト）
        画像1 …
```

作り方：`make slides`（= `go generate` → [tools/img2rgb565](../tools/img2rgb565/main.go) `-pack`）。変換は CatmullRom で縮小、`cover`（切り抜き）/`contain`（黒帯）、Floyd–Steinberg ディザリング。

05 は読み込み元をビルドタグで切り替える：実機は [source_flash.go](../examples/05_slideshow/source_flash.go)（flashmap）、ブラウザ・ホストは [source_embed.go](../examples/05_slideshow/source_embed.go)（`go:embed`）。

### 10.5 slideshow パッケージ

```mermaid
classDiagram
    class Source {
        <<interface>>
        Len() int
        Name(i) string
        Open(i) io.ReaderAt
    }
    class PackSource {
        slidepack.Pack
    }
    class DirSource {
        FS, dir
        BeforeIO func
    }
    class BMPReader {
        BMP → RGB565BE を1行ずつ変換
    }
    class Show {
        Transitions []Transition
        Interval / Duration
        Start() / Step() / Next() / Draw()
        ShowCaption() / HideCaption()
    }
    Source <|.. PackSource
    Source <|.. DirSource
    DirSource ..> BMPReader : .bmp のとき
    Show --> Source
```

- 画像は**全部をメモリに読まない**。`ReadAt` で 8 行ずつ（15KB）読んで `DrawRGBBitmap8` で描く。
- 切り替え効果：Cut（一度に）、WipeDown（8行ずつ上から）、WipeRight（40列ずつ左から）、Blinds（8本の帯を1行ずつ）。
- キャプションは2秒後、**その部分の画像をもう一度読み直して**消す（`HideCaption`）。そのため `Source.Open` が返したリーダーは、次の `Open` まで有効である必要がある。
- タッチは**押し始めた瞬間**だけ反応する（`newPress`）。押し続けても繰り返さない。

---

## 11. SD カードのスライドショー（06）

- SD カード：CS=10、SCK/MOSI/MISO はタッチと共有。**FAT32、MBR** のみ（exFAT・GPT は tinyfs の FatFs 設定で無効）。
- 読める画像：480×272 の `.bmp`（24/32bit、無圧縮）と `.rgb565`。JPEG/PNG は RAM 不足で実機では展開しない。
- 名前が `.` で始まるファイル（macOS の `._xxx`）は無視。

### 共有 SPI バス（[bus.go](../examples/06_sdslideshow/bus.go)）

```mermaid
flowchart LR
    Pins["共有ピン<br/>SCK=12 MOSI=11 MISO=13"]
    SPI["ハードウェア SPI2<br/>SD カード（CS=10）"] ---|useSD: spi.Configure| Pins
    GPIO["GPIO ビットバング<br/>XPT2046（CS=38）"] ---|useTouch: Pin.Configure| Pins
```

- SD を読む直前（`DirSource.BeforeIO`）に `useSD`：ピンを SPI に戻し、タッチの CS を High。
- タッチを読む直前（`busTouch.ReadTouch`）に `useTouch`：ピンを GPIO に戻し、SD の CS を High。
- 今どちらか（`mode`）を覚えているので、切り替えは必要なときだけ。

### C の malloc（[internal/espmalloc](../internal/espmalloc/espmalloc.go)）

tinyfs の fatfs は C コードで `malloc` を呼ぶ。ESP32-S3 のターゲットは `--wrap=malloc` でリンクするので、`malloc` の呼び出しは `__wrap_malloc` に向かう。普通は Wi-Fi ドライバ（espradio）が用意するが、使っていないと未定義になる。espmalloc は `__wrap_malloc` → `__real_malloc`（TinyGo のランタイムの malloc）と転送するだけ。**espradio と一緒に使うとシンボルが重複する。**

---

## 12. ゲーム（07〜12）

### ゲームループ（[07 main.go](../examples/07_breakout/main.go)、[08 main.go](../examples/08_invaders/main.go)）

```mermaid
flowchart TB
    A["dt = 前回からの時間（最大 0.05 秒）"] --> B["タッチを読む"]
    B --> C["ゲームを dt 進める"]
    C --> D{"実機？"}
    D -->|はい| E["WaitVSync"]
    D -->|いいえ| F["60fps になるよう Sleep"]
    E --> G["変わった部分だけ描く"]
    F --> G
    G --> H["5秒ごとに fps をログ"] --> A
```

### 差分描画

毎フレーム全画面（261KB）を描き直さず、変わった所だけ描く。

| 対象 | 方法 |
|---|---|
| 07 パドル | 動いた分の「はみ出た部分」だけ背景色で消し、新しい位置に描く（消してから描くとちらつく） |
| 07 ボール・弾 | 古い位置を背景色で消し、新しい位置に描く（小さいので問題にならない） |
| 08 スプライト | 古い位置と新しい位置を合わせた矩形を**1枚の画像として作り**、`DrawRGBBitmap8` で1回で描く（`blit`） |
| 消えたブロック・敵 | その矩形だけ背景色にする |
| スコア | 変わったときだけ上の帯を描き直す |

### 07 ブロック崩し（[game.go](../examples/07_breakout/game.go)）

- ボールは1回に**最大 2px** ずつ、**横と縦を別々に**動かす。ブロックに重なったら、その軸だけ位置を戻して速度の符号を反転する。速くなってもすり抜けない。
- パドルに当たった位置で角度が決まる（中央 = 真上、端 = 60°）。
- 速さ：220 px/s × (1 + 0.1 × (レベル − 1))。

### 09 数独（[sudoku.go](../games/sudoku/sudoku.go)、[game.go](../games/sudoku/game.go)）

- 盤面は「乱数で完成した盤を作る → ランダムな順にマスを空ける → 解が2つ以上になるマスは戻す」で作るので、**解は必ず1つ**。間違いの判定は解と比べるだけでよい。
- 解を数える探索は再帰を使わず、配列のスタックで書いている（実機のスタックを使い切らないため）。候補が最も少ないマスから試す。
- 描画は、マスごとの「見た目」（背景色・文字色・数字）を覚えておき、変わったマスだけ描き直す。
- 元にした Arduino スケッチ（Sparkadium/Sudoku-for-CYD）は GPL-3.0 のため、コードは移さず、遊び方だけを参考にして書いた。

### 10 オセロ（[othello.go](../games/othello/othello.go)、[game.go](../games/othello/game.go)）

- 盤面は 64bit 整数2つ（手番側と相手側）。合法手は「8方向にシフトして、相手の石の並びの先が空いている所」をビット演算でまとめて求める。正しさは perft（初期局面から n 手先までの局面数の既知の値）で確かめている。
- CPU はαβ探索（negamax）。評価は「マスの価値（隅が高く、隅の隣は低い）」＋「打てる手の数の差」。1手、2手…と深くしていき（反復深化）、**調べた局面数の上限**を超えたら直前の深さの結果を使う。時間ではなく局面数で打ち切るのは、テストで同じ結果を再現するため。
- 探索中はヒープを使わない（手の並べ替えは呼び出し側のスタック上の配列に書く）。
- 手番の進行は状態の切り替え（あなたの番 → 裏返しアニメーション → CPU の思考 → …、パス、終局）。CPU が考える前の更新で「THINKING...」を描いておく。

### 11 ソリティア（[solitaire.go](../games/solitaire/solitaire.go)、[game.go](../games/solitaire/game.go)、[cards](../games/cards/cards.go)）

- トランプの部品（カードの型、切り混ぜ、表・裏・空き枠の描画、スートの 7×7 ドット絵）は `games/cards` にあり、ほかのカードゲームでも使える。
- 操作は「タップしたカードを置ける場所へ自動で動かす」だけ（ドラッグは抵抗膜タッチでは使いにくいため）。行き先は `AutoDest` が決める（1枚なら組札を優先、キングは意味のない空き列への移動をしない）。
- UNDO は状態を約80バイトのバイト列に詰めて最大200手まで保存する（スライスの深いコピーより省メモリ）。
- 描き直しは領域（山札・めくった札・組札・各列）ごとに「中身を表すキー」を覚えておき、変わった領域だけ。しかも盤面が変わったフレームだけ比較する（毎フレームキーを作るとGCの負担になる）。
- 長い列は重なり幅を詰めて画面に収め、詰めすぎて大きな角の文字が隠れるときは小さい角（`DrawFaceSmall`）で描く。

### 12 ビデオポーカー（[poker.go](../games/poker/poker.go)、[game.go](../games/poker/game.go)）

- ルールは「ジャックス・オア・ベター」、配当は 9/6（フルハウス 9・フラッシュ 6）。5枚賭けのロイヤルフラッシュだけ 1枚あたり 800（4000）になるのは実機のマシンと同じ。
- 役の判定（`Evaluate`）はランクごとの枚数を数え、「同じランクが2・3・4枚ある組の数」「全部同じスート」「5つ連続（A は 1 としても 14 としても使える）」から決める。正しさは 5枚の組み合わせ全 2,598,960 通りを数えて、各役の既知の数と一致することで確かめている。
- 1回の勝負（`Round`）は切り混ぜた山を持ち、DRAW では HELD でないカードを山の続きから配る。
- 状態は「待ち → 配る（1枚ずつめくれる）→ 選ぶ → 引く（1枚ずつめくれる）→ 待ち」。めくっている間のタップは無視する。選んでいる間は今できている役の行を水色、引いたあとは勝った役の行を黄色で光らせる。
- 描き直しはソリティアと同じ「領域ごとのキー」方式。

### 08 インベーダー（[game.go](../games/invaders/game.go)）

- 元祖と同じく、**1フレームに1体ずつ**動かす（[stepInvader](../games/invaders/game.go#L659)）。全員が1歩動くのに「生きている数」フレームかかるので、数が減るほど速くなる。
- 1周（全員が1歩）の間に誰かが端に着いたら、次の1周は全員が下に 8px 移動し、向きが反転する。
- 弾は 4px ずつ動かして当たり判定（すり抜け防止）。爆弾は同時に3発まで、乱数（xorshift、テストでは固定シード）で落とす。
- トーチカは 4×4px のセルの集まりで、当たったセルだけ消える。

---

### ランチャー（[cmd/games](../cmd/games/launcher.go)）

- ゲームは `games/` のパッケージ。各ゲームの `OnExit` に関数を入れると、ゲームの最初の画面に「< GAMES」ボタンが出て、押すとその関数が呼ばれる（examples から単体で動かすときは `nil` なので出ない）。
- ランチャーは選ばれたゲームを作り、毎フレーム `Update(dt)` を呼ぶだけ。`OnExit` が呼ばれたらゲームを捨てて（GC が回収）、選ぶ画面を描き直す。
- **タップの持ち越しを防ぐ**：タイルをタップした指はまだ触れているので、そのままゲームに渡すとゲームの最初の画面でもタップとして扱われる（数独なら難易度ボタンが押される）。ゲームを始めた直後と戻った直後は、指が離れるまで入力を渡さない（`waitRelease`）。テストはタイルの「ゲームのボタンと重なる位置」（数独は重なるボタンがないので除く）をタップして、これを確かめている。
- タイルは 3列×2段（1枠空き）。5列ではゲーム名が入らなかった。

## 13. 運用：コマンド・よくある作業・トラブル

### Makefile

| コマンド | 内容 |
|---|---|
| `make flash PKG=...` | 書き込み＋モニタ（**フラッシュ全消去**） |
| `make flash-noerase PKG=...` | 消去せずに書き込み（スライドを残す） |
| `make flash-slides` | slides.pack を 0x800000 に書く |
| `make monitor` | シリアルモニタ（115200bps） |
| `make examples` | 全 example と cmd/app を実機向けにビルド |
| `make wasm PKG=...` / `make serve` | ブラウザ版のビルドと配信 |
| `make test` | ホストのテスト |
| `make test-browser PKG=... GOLDEN=... DUMP_MS=...` | headless Chromium で canvas とゴールデン画像を比較 |
| `make update-golden` | ゴールデン画像の更新 |
| `make slides` | images/ → slides.pack |

### よくある作業

| やりたいこと | 変える場所 |
|---|---|
| PCLK・ポーチを変える | [board.go](../board/board.go) の定数。`rgblcd/calc_test.go` は board を参照しない（式の検証）ので変更不要。実機で 02_colorbars の Dump が OK になることを確かめる |
| ピンを変える（別の基板） | `board` と `platform.RGBConfig()`。`rgblcd` 本体は変えない |
| タッチがずれる | 04_touch で測り直し、`board.TouchRaw*` を書き換える |
| スライドの画像を変える | `examples/05_slideshow/images/` → `make slides` → `make flash-slides`（テストのゴールデン画像は1枚目に依存） |
| 新しい example を作る | `examples/NN_name/main.go`。`machine` を使うならビルドタグ。`Makefile` の `EXAMPLES` に追加 |
| 画面の見た目を変えた | `make test` → 失敗 → `*.actual.png` を確認 → `make update-golden` → 目視 → コミット |

### トラブルシューティング

| 症状 | 原因と対処 |
|---|---|
| 真っ暗 | バックライト（GPIO2）。01_backlight で確認 |
| 色がおかしい | データピンの順番。02_colorbars で `r`/`g`/`b` を送って1色ずつ確認 |
| 横ずれ・ジッタ・ちらつき | ポーチ値、PCLK が高い。PCLK を下げる |
| 絵が1周ごとにずれる | ディスクリプタの長さの合計が1フレームと違う。Dump の `total length` |
| `=== register check: NG ===` | NG の行のフィールドを見る。計算値と書き込み手順を確認 |
| 05 で `bad magic` | スライドが書かれていない／`make flash` で消えた → `make flash-slides` |
| `fatal error: deadlocked: no event source` | 実機で `select {}`。`platform.Halt()` を使う |
| `undefined symbol: __wrap_malloc` | C コードを使うのに espmalloc を import していない |
| `package machine is not in std`（go test） | `machine` を使うファイルにビルドタグがない |
| ブラウザで真っ白 | `file://` で開いている、wasm_exec.js が Go 付属、MIME が違う |
| `make flash-slides` が 0xC4 | `-fs 16MB` がない |
| `Invalid image block, can't boot` | プログラムに大きなデータを埋め込んだ（rodata が大きい） |

---

## 14. 未解決・注意事項

- **起動時の `SHA-256 comparison failed`**：ROM が計算したハッシュとイメージのハッシュが違う。起動は続くので実害はないが、原因は未調査。
- **MMU エントリを2か所に書いている**：ESP-IDF（共有の表、エントリ = 仮想アドレス >> 16）と TinyGo の起動コード（DROM はエントリ 256〜）の解釈の違い。どちらで動いているかは未確認。
- **06 は実機で未確認**：SD カードの動作確認がまだ。SPI 10MHz で画素化けする場合は `sdFrequency` を 4MHz に。
- **タッチの読み取りが遅い**：xpt2046 ドライバがビットバングで、各クロックで `time.Sleep` を呼ぶ。ゲームの fps が下がるならサンプル数を減らす。
- **PSRAM と Wi-Fi は未使用**：使う場合はメモリ計画（[5章](#5-メモリの使い方)）を見直す。
- **フェーズ8（VSYNC 割り込み）・フェーズ10（メモリ節約モード）は未実装**。

---

## 15. ESP32-3248S035C（3.5インチ・SPI 液晶）

ESP32（無印、PSRAM なし）、ST7796（320×480、SPI）、GT911（静電容量タッチ、I2C）。横向き 480×320 で使う。ピンと設定値は [board/s035](../board/s035/s035.go) にまとめてある。

### S043 との違い

| | ESP32-4827S043 | ESP32-3248S035C |
|---|---|---|
| CPU・RAM | ESP32-S3、内蔵 SRAM 512KB | ESP32、TinyGo が使える DRAM は 192KB |
| 液晶 | RGB パラレル、DMA が常にフレームバッファを送る | SPI（40MHz）、液晶側が画面のメモリを持つ |
| 描画 | フレームバッファに書くだけ（速い） | 描くたびに SPI で送る（[spilcd](../spilcd/spilcd.go)） |
| タッチ | XPT2046（抵抗膜、SPI ビットバング） | GT911（静電容量、I2C 400kHz） |
| ターゲット | esp32s3-generic を継承 | [esp32-generic](../targets/esp32-3248s035c.json) を継承（I2C のピン定数が要るため） |

### 描画（spilcd）

- コントローラごとに違うのは初期化手順だけ（`spilcd.InitST7796`・`InitILI9341`）。CASET・RASET・RAMWR・MADCTL などの描画のコマンドは共通（MIPI DCS）。初期化手順のデータは文字列なのでフラッシュに置かれる。
- 480×320×2 = 300KB のフレームバッファは RAM に入らないので持たない。`FillRectangle` は「アドレス窓（CASET・RASET）→ RAMWR → 色を繰り返し送る」。`DrawRGBBitmap8` のバイト列（ビッグエンディアン RGB565）はそのまま液晶の形式なので、そのまま送る。
- **SetPixel のまとめ送り**：1点ごとに窓を設定すると遅い。tinyfont は文字を左上から1行ずつ1点ずつ描くので、同じ行で右に続く同じ色の点を「ラン」にまとめ、続かなくなったときに1本の矩形として送る。まだ送っていないランは `Display()` か次の描画で送る。
- 前回と同じ CASET・RASET は送らない（同じ行のランは列だけ変わる）。
- 速さ（実測、SPI 40MHz）：480×320 の全画面塗りつぶし 77ms（理論値 61ms）、freesans Bold 9pt の文字 300 字で 137ms（1字 約 0.5ms）。
- 液晶からの読み出しはできない（ID が常に 0。MISO が液晶につながっていないらしい）。描画には不要。
- テストは液晶の側を真似る [panel](../spilcd/spilcd_test.go)（コマンドを解釈して画素を持つ）を作り、フレームバッファに同じものを描いた結果と画素単位で比べる。

### 480×272 のプログラムを動かす（viewport）

- `platform.Init()` は画面中央（y = 24〜295）の 480×272 の [viewport](../viewport/viewport.go) を返す。描画は 24px 下にずらして窓の外を切り、タッチは逆にずらす（窓の外のタッチは近い辺に寄せる：上の帯を少し外れて押してもボタンに当たる）。
- `platform.Init()` のタッチは、読む前に液晶の未送信のランを送る（`flushTouch`）。ゲームは毎フレーム必ずタッチを読むので、`Display()` を呼ばないコードでも最後に描いた文字が画面に出る。
- 画面全体を使うときは `platform.InitScreen()`（480×320）と `platform.InitTouch()`。

### タッチ（gt911）

- GT911 は約 100 回/秒測る。状態レジスタ 0x814E の bit7 が「新しい値がある」、下位4ビットが点の数。読んだら 0 を書いて次の測定を受け取る。新しい値がない間は前の状態を返す（毎フレーム読んでもよい）。
- I2C アドレスはリセット時に 0x5D か 0x14 に決まる。INT ピンは資料で 21 と 36 に分かれているので使わず、両方のアドレスを試す。見つからなければ RST（GPIO25）でリセットしてもう一度試す。
- GT911 の座標は縦向きで、解像度は液晶と同じ 320×480。横向き（MADCTL 0x28）では GT911 の Y が画面の X、GT911 の X が画面の下→上になる。変換は [touchcal](../touchcal/calib.go)（`SwapXY` と左右・上下の raw 値）で、拡大縮小はしない（raw 0〜479 / 319〜0）。
- 実測では、端から 20px の目標のうち下の2つだけ 12〜19px 下にずれて読めた。4隅から求めた補正値を使うと画面の下 16px に触れなくなるので、GT911 の座標をそのまま使っている（[board/s035](../board/s035/s035.go) のコメント）。
- **TinyGo の I2C の不具合への対処**：ESP32 の I2C ドライバはピンの出力を `GPIO.ENABLE_W1TS.Set(1 << pin)` で有効にするが、32 番以上のピンは ENABLE1 側にあるので 0 になり、SCL（32）・SDA（33）が駆動されない。[platform_s035.go](../platform/platform_s035.go) の `enableHighPins` で ENABLE1 のビットを立てている。

### 起動時に止まった原因：UART0 の初期化が後回しになる（TinyGo の不具合）

- **症状**：cmd/games を書き込むと、何も表示されず、ログも出ない。ROM の表示（`entry 0x40080058`）のあと、ランタイムが UART を設定し直すところ（`machine.InitSerial`）で止まっていた。
- **原因**：
  - TinyGo はパッケージの初期化（大域変数の初期値など）をコンパイル時に実行して、結果を `.data` に入れる（interp）。実行できなかった初期化は、起動時に `runtime.initAll` で実行する。
  - games/invaders のスプライト（`parseSprite` で文字列から組み立てる）はコンパイル時に実行できず、起動時に回っていた。
  - その結果、invaders より後に初期化される `machine` の `_UART0`（UART0 のレジスタを指す構造体）まで起動時に回され、`.bss`（ゼロ）に置かれた。
  - ESP32 のランタイム（`runtime_esp32.go` の `main`）は、パッケージの初期化より**前**に `machine.InitSerial()` を呼ぶ。このため、空の `_UART0`（レジスタのポインタが nil）を使って止まる。例外の表示も出ない。
- **見分け方**：`go tool nm <elf> | grep machine._UART0` が `d`（.data）なら正常、`b`（.bss）なら起動時に止まる。`make examples-s035` が全プログラムでこれを確かめる。
- **対策**：ライブラリのパッケージに、起動時に組み立てる大域変数を置かない。invaders のスプライトは文字列のまま持ち、最初の `NewGame` で組み立てる（[sprites.go](../games/invaders/sprites.go)）。main パッケージの初期化は最後に実行されるので影響しない。
- **調べ方**：TinyGo の `src/runtime/runtime_esp32.go` の写しを作り、`runtime.main` の各段階で UART0 の FIFO に1文字ずつ書いて（`TINYGOROOT` を写しに向けてビルド）、どこまで進むかを見た。クロックを上げてから UART を設定し直すまでは 230400bps で出力される（APB が 80MHz になり、分周比は ROM の 40MHz 用のまま）ので、モニタを 230400bps にすると読める。
- 途中で疑った「`.data` の大きさ」「フラッシュ 80MHz の読み出し」「コードの2ページ目の MMU」は、どれも原因ではなかった（2ページ目の割り当ては正しいことを実機で確認）。

### フォントはフラッシュに置く（romfont）

- tinyfont のフォント（`freesans.Bold9pt7b` など）はスライスを持つ変数なので、TinyGo は書き換えられる可能性があると見て、字形の表もビットマップも `.data`（RAM）に置く。cmd/games では5書体で 28KB あった。ESP32 の DRAM は全部で 192KB しかない。
- [romfont](../romfont/romfont.go) は字形を2本の文字列（索引とデータ）に入れる。文字列の中身は rodata（フラッシュ）に置かれるので、`.data` は cmd/games で 960 バイトになった（S043 でも RAM が 28KB 空く）。描画は tinyfont と1ピクセルも違わないことを全書体・全文字で確かめている（[romfont_test.go](../romfont/romfont_test.go)）。ゴールデン画像も変わらない。
- 使い方は import を `tinygo.org/x/tinyfont/freesans` から `github.com/sat0ken/tinygo-cyd/romfont/freesans` に変えるだけ（名前は同じ）。新しい書体が要るときは [tools/fontgen](../tools/fontgen/main.go) の一覧に足して `go generate ./romfont`。
- [boot_s035.go](../platform/boot_s035.go)：前回の実行が例外で再起動していたら、起動時にその原因と番地を表示する（TinyGo の例外ハンドラが RTC のレジスタに残す）。電源投入以外のリセットなら、その理由も表示する。

### 実機で確かめること（s035_check）

[s035_check](../examples/s035_check/main.go) を書き込み、ログ・色の帯・向き・タッチを見る。シリアルのコマンドで MADCTL（`r` 180度、`o` 赤青）・反転（`i`）を切り替えられ、`k` で4隅のキャリブレーション、`f`・`t` で描画速度を測る。合っていたら値を board/s035 に書く。テスト画面・コマンド・キャリブレーションは s028_check と共通で、[internal/panelcheck](../internal/panelcheck/panelcheck.go) にある。

## 16. ESP32-2432S028R（2.8インチ・320×240）

ESP32（PSRAM なし）、ILI9341（240×320、SPI）、XPT2046（抵抗膜タッチ）。横向き 320×240 で使う。ピンと設定値は [board/s028](../board/s028/s028.go)。CPU と描画の仕組みは 3.5インチ版（15章）と同じ。

| | ESP32-3248S035C | ESP32-2432S028R |
|---|---|---|
| 液晶 | ST7796、480×320 | ILI9341、320×240（`spilcd.InitILI9341`） |
| タッチ | GT911（I2C） | XPT2046（S043 と同じ [xpttouch](../xpttouch/xpttouch.go)、ピンは CLK 25・MOSI 32・MISO 39・CS 33・IRQ 36） |
| `platform.Init()` | 中央の 480×272 の窓 | 画面全体の 320×240 |
| 動くプログラム | 03、07〜12、cmd/app、cmd/games | s028_check、cmd/app |

- 480×272 より小さいので、480×272 で配置を決めているゲームは動かない（窓に入らない）。[app](../app/app.go) は `New` で画面の大きさから配置を選び、480×272 未満ならデモを省いて紙・パレット・CLEAR だけにする（ゴールデン画像 `app_small`）。
- 液晶の MISO（GPIO12）は液晶の SDO につながっている（3.5インチ版はつながっていなかった）。s028_check は起動時に ID を読み、TinyGo の SPI で読み出しができるかも確かめる。
- 共通部分：タッチを読む前に未送信のランを送る `flushTouch`（[spilcd_esp32.go](../platform/spilcd_esp32.go)）、起動時の診断（[boot_esp32.go](../platform/boot_esp32.go)）。`make examples-s028` も `machine._UART0` の置き場所を確かめる（15章）。
