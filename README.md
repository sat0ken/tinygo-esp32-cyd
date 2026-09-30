# tinygo-cyd: ESP32-4827S043 RGB液晶ドライバ（TinyGo）

Sunton ESP32-4827S043（ESP32-S3、4.3インチ 480×272 RGB565パラレル液晶）を TinyGo から駆動する。
液晶は LCD_CAM + GDMA の循環DMAで常時リフレッシュされる（CPU介入なし）。
同じアプリをブラウザ（wasm）でも動かせる。

## 必要なもの

- TinyGo 0.42 以降（`tinygo version`）
- Go 1.23 以降（ホストのテスト、`make serve`）
- 実機に書き込む場合：USB-C ケーブル（CH340C 経由で UART0 に接続される）
- 任意：Chromium / Google Chrome（`make test-browser`）

## 構成

```
board/        ピン番号・タイミング・タッチのキャリブレーション値（ボード固有の値はここだけ）
hal/          アプリが依存するインターフェース（Display, Touch）
framebuf/     RGB565フレームバッファへの描画（全バックエンド共通）
rgblcd/       ESP32-S3 LCD_CAM + GDMA ドライバ（Config でピン・タイミングを受け取る）
xpttouch/     XPT2046 タッチ → 画面座標（hal.Touch）
wasmlcd/      ブラウザ用バックエンド（canvas + pointer イベント）
memlcd/       メモリ上バックエンド（テスト用、PNG 出力）
platform/     ビルドタグで実機 / wasm / ホストを切り替える初期化
app/          アプリ本体（hal だけに依存）とゴールデン画像テスト
cmd/app/      実機・wasm 共通のエントリポイント
web/          index.html（app.wasm と wasm_exec.js は make wasm で生成）
examples/     01_backlight, 02_colorbars, 03_tinydraw, 04_touch
testdata/golden/  ゴールデン画像
targets/esp32-4827s043.json  カスタムターゲット（esp32s3-generic 継承、serial=uart）
tools/serve   wasm 用の簡易 HTTP サーバ（application/wasm で配信）
```

## 実機への書き込み

```sh
make flash                                   # cmd/app
make flash PKG=./examples/02_colorbars       # 例を指定
make flash PKG=./examples/02_colorbars PORT=/dev/ttyUSB0
make monitor                                 # シリアルモニタのみ（115200bps）
make examples                                # 全 example のビルド確認
```

中身は `tinygo flash -target=./targets/esp32-4827s043.json -monitor <pkg>`。
書き込めない場合は、BOOT ボタンを押したまま RST を押してダウンロードモードに入れる。

### examples

| 例 | 内容 | 確認すること |
|---|---|---|
| 01_backlight | GPIO2 の PWM でバックライトを 10% 刻みで上下させ、1秒ごとにログを出す | 明るさが変わる、ログが出る |
| 02_colorbars | 立ち上げ手順（フェーズ1〜5）を順に実行し、各段階のレジスタを読み戻して OK/NG をログに出す。8色のカラーバーと四辺の白枠を表示 | 下記「実機での確認項目」 |
| 03_tinydraw | アプリ画面（tinydraw 図形 + tinyfont 文字）を描いて描画時間をログに出す。wasm でも動く | 図形と文字が正しく出る |
| 04_touch | 4隅のターゲットをタッチしてキャリブレーションし、`board.go` 用の値をログに出す。その後タッチ位置に点を描く | 点がペン先の位置に出る |

02_colorbars はシリアルから1文字コマンドを受け付ける：
`c` カラーバー、`r`/`g`/`b`/`w`/`k` 単色（赤/緑/青/白/黒）、`d` レジスタを再ダンプ。
色がおかしいときは単色で1色ずつ確認する。

## ブラウザ（wasm）で動かす

```sh
make wasm     # TinyGo 付属の wasm_exec.js をコピーし web/app.wasm をビルド
make serve    # http://localhost:8080/ を開く
```

- マウスのクリック・ドラッグがタッチになる（右側の白い領域に描ける。下のパレットで色選択、CLEAR で消去）。
- `println` の出力はブラウザの開発者コンソールに出る。
- `file://` では動かない（`.wasm` を `application/wasm` で配信する必要がある）。
- `wasm_exec.js` は TinyGo 付属のものを使う（Go 付属のものは互換性がない）。TinyGo を更新したら `make wasm` でコピーし直される。

wasm 版で確認できるのは描画・UI・アプリのロジックだけ。RAM 容量、描画速度、ティアリング、液晶の発色、
抵抗膜タッチの精度、Wi-Fi などは実機で確認すること。

## テスト

```sh
make test            # go test ./...（ゴールデン画像テストを含む）
make test-browser    # 任意：headless Chromium で web/ を開き、canvas を同じゴールデン画像と比較
make update-golden   # UI を意図して変えたとき、ゴールデン画像を更新（コミット前に画像を目視確認）
```

ゴールデン画像と違うと、実際の画像が `testdata/golden/<name>.actual.png` に書き出される。

`rgblcd` の計算部分（PCLK 分周、タイミングレジスタ値、DMA ディスクリプタ）はホストでテストしている。
値は ESP-IDF の変換式から手計算した期待値と比較する。

## rgblcd ドライバ

```go
d, err := rgblcd.New(platform.RGBConfig(), platform.FrameBuffer())
d.Start()
d.FillRectangle(0, 0, 100, 50, color.RGBA{255, 0, 0, 255})
d.Dump() // 全レジスタを読み戻して期待値と比較
```

- `drivers.Displayer`（`Size` / `SetPixel` / `Display`）と、ST7789 ドライバと同じ `FillRectangle`、`DrawBitmap`（`pixel.Image[pixel.RGB565BE]`）、`DrawRGBBitmap8`、`FillScreen`、`DrawFastHLine`/`VLine` を持つ。
- `Display()` は何もしない（常時表示）。ティアリングを減らしたいときは描画前に `WaitVSync(timeout)` を呼ぶ（LCD の VSYNC フラグをポーリング）。
- フレームバッファは内部 SRAM（4バイト境界）に置くこと。PSRAM は未対応。
- レジスタ操作には ESP-IDF v5.4（`lcd_ll.h`、`esp_lcd_panel_rgb.c`、`gdma_ll.h` など）と TinyGo `device/esp`（SVD）の出典コメントを付けている。

### 設定値（board/board.go）

| 項目 | 値 |
|---|---|
| PCLK | 9MHz（PLL160M ÷ (8 + 8/9) ÷ 2、ちょうど 9.000MHz） |
| HSYNC | pulse 4, back porch 43, front porch 8, idle low |
| VSYNC | pulse 4, back porch 12, front porch 8, idle low |
| PCLK 極性 | 立ち下がりでデータ出力（pclk_active_neg） |
| リフレッシュ | 9MHz ÷ (535 × 296) ≒ 56.8Hz |
| DMA | 1ディスクリプタ = 4行（3840バイト）× 68個の循環リスト |

表示が不安定なら `PclkHz` を 8MHz → 7MHz と下げる。

### メモリ

`cmd/app` のビルドで、内部 DRAM（使えるのは 0x3FC88000〜0x3FCEB710）の使い方は次のとおり。

| 領域 | サイズ |
|---|---|
| スタック | 4KB |
| .bss（フレームバッファ 255KB を含む） | 257KB |
| .data（フォントなど） | 10.5KB |
| IRAM（DRAM と共有） | 2.4KB |
| **ヒープの残り** | **約 123KB** |

Wi-Fi（espradio）を併用するとヒープが足りなくなる可能性が高い。その場合は 8bit パレット＋バウンスバッファ方式（計画書フェーズ10）を検討する。

## 実機での確認項目

wasm 版やホストのテストで問題がなくても、次は実機で確認する。

1. **フェーズ0**（01_backlight）：書き込める、シリアルログが見える、バックライトの明るさが変わる。
2. **フェーズ1〜4**（02_colorbars のログ）：`self test: OK`、各ダンプが全行 `OK`、`=== register check: OK ===`、DMA ディスクリプタが `circular: true (68 steps)  total length: 261120`。
3. **フェーズ5**（02_colorbars の画面）：
   - カラーバーが左から 白・黄・シアン・緑・マゼンタ・赤・青・黒 の順か
   - 白枠が四辺すべてに見えるか（はみ出し・ずれ）
   - ちらつき・横ずれ・ノイズがないか
   - ログの `fps` が約 56〜57（LCD_CAM がフレームを出し続けている）
   - ログの `desc:` の6個の数字（1フレーム内で 3ms ごとに読んだディスクリプタ番号 0〜67）が進んで一周している（DMA が動いている）。
     `OUT_DSCR` は VSYNC 直後に読むので毎回同じ値になるのが正常。
4. **フェーズ6**（03_tinydraw）：図形と文字が `testdata/golden/app_initial.png` と同じに見える。
5. **フェーズ9**（04_touch）：キャリブレーション後、点がペン先の位置に出る。出力された値を `board/board.go` に書く。

### トラブルシューティング

| 症状 | 考えられる原因 | 対処 |
|---|---|---|
| 真っ暗 | バックライトが OFF | GPIO2 を確認（01_backlight） |
| 真っ白・無表示 | PCLK が出ていない、DE の極性が逆、DMA が止まっている | ダンプの NG 行、`LCD_START`、`OUT_DSCR` が変化しているかを確認 |
| 横方向のずれ・列のジッタ | 水平ポーチ値、PCLK が高すぎる | board.go の値に戻す、PCLK を下げる |
| 行が乱れる・ちらつく | 垂直ポーチ値、DMA の供給不足 | PCLK を下げる、ディスクリプタとバースト設定を確認 |
| 色がおかしい | データピンの順番、R と B の入れ替わり | 02_colorbars で `r`/`g`/`b` を送って1色ずつ確認 |
| 絵が1周ごとにずれる | ディスクリプタの長さ合計が1フレームと一致しない | ダンプの `total length` が 261120 か |
| 数秒後に表示が崩れる | 他の DMA や処理と帯域が競合 | PCLK を下げる |
| 起動しない | ストラッピングピン（GPIO3/45/46） | 初期化のタイミングを確認 |

## 未実装・今後

- フェーズ8：VSYNC は割り込みではなくポーリング（`WaitVSync`）。割り込み版は未実装。
- フェーズ10：メモリ節約モード（8bit パレット + バウンスバッファ）は未実装。
- タッチのキャリブレーション初期値は仮の値（要確認）。04_touch で求めた値に置き換えること。
