# tinygo-cyd: ESP32-4827S043 RGB液晶ドライバ（TinyGo）

Sunton ESP32-4827S043（ESP32-S3、4.3インチ 480×272 RGB565パラレル液晶）を TinyGo から駆動する。
液晶は LCD_CAM + GDMA の循環DMAで常時リフレッシュされる（CPU介入なし）。
同じアプリをブラウザ（wasm）でも動かせる。

ESP32-3248S035C（ESP32、3.5インチ 480×320 SPI 液晶、静電容量タッチ）と ESP32-2432S028R（ESP32、2.8インチ 320×240 SPI 液晶、抵抗膜タッチ）にも対応している（[3.5インチ](#esp32-3248s035c35インチ)・[2.8インチ](#esp32-2432s028r28インチ)）。

## 必要なもの

- TinyGo 0.43 以降（`tinygo version`、0.44.0 で確認）。0.42 は ESP32 でゴルーチンのスタックの位置が揃わないことがあり、値を黙って読み違える（docs/ARCHITECTURE.md 15章）
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
touchcal/     タッチの raw 値 → 画面座標の変換とキャリブレーション（xpttouch・gt911 共通）
board/s035/   ESP32-3248S035C のピン番号・設定値
board/s028/   ESP32-2432S028R のピン番号・設定値
spilcd/       SPI 液晶ドライバ（ST7796・ILI9341、フレームバッファなし。ESP32-3248S035C・ESP32-2432S028R）
gt911/        GT911 静電容量タッチ（I2C）→ 画面座標（hal.Touch）
viewport/     大きい画面の中の小さい窓（480×320 の中で 480×272 のゲームを動かす）
minijoy/      I2C のミニジョイスティックモジュール（nulllab、0x5A）のドライバ
pad/          ゲームコントローラー（hal.Pad）の入力・フォーカス移動、ジョイスティックの変換
picopad/      PicoLibSDK（PicoPad の C の SDK）の描画・ボタン・時間・乱数の関数（PicoPad のゲームの移植用。8bit パレットのフレームバッファ）
romfont/      フラッシュに置くフォント（tinyfont の freesans・proggy を tools/fontgen で変換。描画は同じ）
wasmlcd/      ブラウザ用バックエンド（canvas + pointer イベント）
memlcd/       メモリ上バックエンド（テスト用、PNG 出力）
platform/     ビルドタグで実機 / wasm / ホストを切り替える初期化
app/          アプリ本体（hal だけに依存）とゴールデン画像テスト
cmd/app/      実機・wasm 共通のエントリポイント
web/          index.html（app.wasm と wasm_exec.js は make wasm で生成）
examples/     01_backlight, 02_colorbars, 03_tinydraw, 04_touch
testdata/golden/  ゴールデン画像
targets/esp32-4827s043.json  カスタムターゲット（esp32s3-generic 継承、serial=uart）
targets/esp32-3248s035c.json ESP32-3248S035C のターゲット（esp32-generic 継承）
tools/serve   wasm 用の簡易 HTTP サーバ（application/wasm で配信）
tools/picoimg PicoLibSDK の画像・フォントの C の配列を Go の文字列定数に変換（フラッシュに置かれる）
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

### ゲームをまとめて遊ぶ（cmd/games）

```sh
make flash-noerase PKG=./cmd/games        # 実機
make wasm PKG=./cmd/games && make serve   # ブラウザ
```

起動するとゲームを選ぶ画面が出る（INVADERS / SUDOKU / OTHELLO / SOLITAIRE / POKER / KART / SPACE）。各ゲームの最初の画面にある「< GAMES」で選ぶ画面に戻る。
ゲーム本体は `games/invaders`・`games/sudoku`・`games/othello`・`games/solitaire`・`games/poker`・`games/kart`・`games/starguard` パッケージにあり（トランプの部品は `games/cards`）、examples の 08〜14 は1つずつ動かすだけの `main.go`。

### examples

| 例 | 内容 | 確認すること |
|---|---|---|
| 01_backlight | GPIO2 の PWM でバックライトを 10% 刻みで上下させ、1秒ごとにログを出す | 明るさが変わる、ログが出る |
| 02_colorbars | 立ち上げ手順（フェーズ1〜5）を順に実行し、各段階のレジスタを読み戻して OK/NG をログに出す。8色のカラーバーと四辺の白枠を表示 | 下記「実機での確認項目」 |
| 03_tinydraw | アプリ画面（tinydraw 図形 + tinyfont 文字）を描いて描画時間をログに出す。wasm でも動く | 図形と文字が正しく出る |
| 04_touch | 4隅のターゲットをタッチしてキャリブレーションし、`board.go` 用の値をログに出す。その後タッチ位置に点を描く | 点がペン先の位置に出る |
| 05_slideshow | 画像を5秒ごとに切り替える（ワイプ・ブラインドの切り替え効果）。タッチで右 1/3 = 次、左 1/3 = 前、中央 = 一時停止。wasm でも動く。**実機では先に `make flash-slides` が必要** | 画像が崩れずに出る、切り替えとタッチが効く |
| 06_sdslideshow | microSD カードの画像（480×272 の BMP か `.rgb565`）で 05 と同じスライドショー。タッチ操作も同じ | カードの画像が出る、タッチが効く |
| 07_breakout | ブロック崩し。指（ブラウザではマウス）でなぞるとパドルが動き、タップで発射。ライフ3、全消しで次のレベル（10% 速くなる）。wasm でも動く | パドルが指に付いてくる、ちらつかない、ログの fps |
| 08_invaders | インベーダーゲーム。なぞって砲台を移動、触れている間は自動で発射。55体が1体ずつ動き（元祖と同じく減るほど速くなる）、トーチカ・UFO・ウェーブ制。wasm でも動く | 砲台が指に付いてくる、ちらつかない、ログの fps |
| 09_sudoku | 数独。難易度3段階（ヒント 38/30/25 個）、解が必ず1つの盤面を生成。マスを選んで数字パッドで入力、ERASE・HINT・NEW、ハイライト、間違いは赤、経過時間とミス数。wasm でも動く。[Sparkadium/Sudoku-for-CYD](https://github.com/Sparkadium/Sudoku-for-CYD) を参考にした独自実装 | マスと数字パッドの位置がタッチと合う |
| 10_othello | オセロ（CPU 対戦）。先手/後手と強さ（EASY / NORMAL / HARD）を選ぶ。打てる場所に点、直前の手に赤い点、石が1つずつ裏返るアニメーション、UNDO。CPU はビットボード＋αβ探索（HARD は6手読み・終盤12マスは読み切り）。wasm でも動く | HARD の思考時間（右下とログの depth / nodes / ms） |
| 11_solitaire | ソリティア（クロンダイク）。カードをタップすると置ける場所へ自動で動く（1枚なら組札優先）。山札のタップでめくる、1枚/3枚めくり、UNDO（200手）、最後は自動で片付け。長い列は重なりを詰めて画面に収める。wasm でも動く | カードのタップ位置、3枚めくりの表示 |
| 12_poker | ビデオポーカー（ジャックス・オア・ベター、9/6 配当）。BET ONE / MAX BET で賭けて DEAL、残すカードをタップ（HELD）して DRAW。上の配当表で賭け数の列が赤、できている役の行が光る。クレジットが尽きたら DEAL で 100 補充。wasm でも動く | カードのタップで HELD が切り替わる、ボタンの位置 |
| 13_kart | カートレース（オリジナルのコース、3周、CPU 3台）。地面は昔の「モード7」と同じく、画面の1行ずつを平らなコースの地図の上の1本の線として描く。CPU の強さ（EASY / NORMAL / HARD）、カウントダウン、加速パネル、芝生では遅くなる、カート同士の押し合い、周回・順位・タイム・小さな地図、ゴールで順位とベストラップ。ジョイスティック（スティックでハンドル、A アクセル、B ブレーキ、Y でやめる）かタッチ（地面に触れている間アクセル、触れた側へ曲がる、地図をタップでやめる）。ログに fps。wasm でも動く | ESP32-2432S028R で 46 fps（DMA） |
| 14_starguard | 宇宙の防衛シューティング「STAR GUARD」（オリジナル）。操縦席から前を見て、飛んでくる岩とロボット艦を照準で撃つ。星・敵は 3D の位置を画面に投影して描き、遠いものから重ねる。強さ（EASY / NORMAL / HARD）、3ステージ（決まった数を倒すとクリア、シールド +20）のあと母艦（HP ゲージ、ドローンを出す）。シールドのカプセルを撃つとシールドが戻る。下の計器にシールド・スコア・ステージ・レーダー。ジョイスティック（スティックで照準、A を押している間撃つ、Y でやめる）かタッチ（触れた所を狙って撃つ、レーダーをタップでやめる）。ログに fps。wasm でも動く | ESP32-2432S028R で 37 fps（DMA） |
| 15_tetris | PicoPad（RP2040 のゲーム機）の SDK「PicoLibSDK」のテトリスを C から移植したもの（[games/tetris](games/tetris/tetris.go)）。左右で移動、UP か X で回転、DOWN か A で落とす、B で一時停止、Y でやめる。新しいゲームは一時停止で始まり、10秒ボタンを押さないとデモになる。画面は PicoPad と同じ 320×240（大きい液晶では中央）。ジョイスティックがなければ画面を 3×3 に分けたタッチのボタンで遊ぶ（[PicoPad のゲーム](#picopad-のゲーム)）。wasm でも動く | ブロックが動く・消える、ちらつかない、ボタンの反応 |
| 16_maze | 同じく PicoLibSDK の迷路（[games/maze](games/maze/maze.go)）。矢印で顔を動かして出口を探す。見えるのは通った道の周りだけ。A で出口が一瞬見える。出口に着くと迷路全体が見え、ボタンで次の迷路。Y でやめる。ジョイスティックがなければ 3×3 のタッチのボタン（[PicoPad のゲーム](#picopad-のゲーム)）。wasm でも動く | 顔が動く・壁で止まる、通った所が見えていく |
| 17_fifteen | 同じく PicoLibSDK の 15 パズル（[games/fifteen](games/fifteen/fifteen.go)）。矢印でタイルを穴に動かして 1〜15 に並べる。A で混ぜ直し、X で自動で解く（ボタンで止まる）、Y でやめる。揃うと点滅し、ボタンで次のゲーム。ジョイスティックがなければ 3×3 のタッチのボタン（[PicoPad のゲーム](#picopad-のゲーム)）。wasm でも動く | タイルが動く、X で最後まで解ける |
| 18_life | 同じく PicoLibSDK のライフゲーム（[games/life](games/life/life.go)）。50×50 の世界（端はつながる）の中央 20×15 を表示。矢印でカーソル、A でセルを反転、B で動かす（ボタンで止めると動かす前に戻る）、X で次のスロット（グライダー銃などの模様が入っている 10 個）、Y でやめる。ジョイスティックがなければ 3×3 のタッチのボタン（[PicoPad のゲーム](#picopad-のゲーム)）。wasm でも動く | セルが反転する、B で動く・止めると戻る |

02_colorbars はシリアルから1文字コマンドを受け付ける：
`c` カラーバー、`r`/`g`/`b`/`w`/`k` 単色（赤/緑/青/白/黒）、`d` レジスタを再ダンプ。
色がおかしいときは単色で1色ずつ確認する。

### スライドショー（05_slideshow）

実機では、画像データ（`examples/05_slideshow/slides.pack`）をプログラムとは別にフラッシュの 8MB 位置へ書き込む。

```sh
make flash-slides                                # slides.pack -> フラッシュ 0x800000（PORT 省略時 /dev/ttyUSB0）
make flash-noerase PKG=./examples/05_slideshow   # プログラムだけ書く（スライドは消さない）
make monitor
```

**注意：`make flash`（`tinygo flash`）は書き込み前にフラッシュ全体を消去するので、スライドも消える。**
スライドを残したままプログラムを書き直すには `make flash-noerase` を使う。
`make flash` を使った場合は、そのあとに `make flash-slides` をやり直す。

画像を差し替えるときは `examples/05_slideshow/images/` に PNG / JPEG / GIF を置いて `make slides` を実行する
（ファイル名順に表示）。`tools/img2rgb565` が 480×272 に切り抜き・縮小し、RGB565（ディザリングあり）のパックを作る。
切り抜かずに黒帯で収めたい場合は `go run ./tools/img2rgb565 -fit contain -pack examples/05_slideshow/slides.pack examples/05_slideshow/images`。

- 1枚 255KB。フラッシュ領域は 8MB なので最大 32 枚（(8MB − ヘッダ 4KB) ÷ 261,120 バイト。パック形式の上限は 127 枚）。
- 画像をプログラムに埋め込まない理由：TinyGo は2段目ブートローダを使わず ROM が直接プログラムを読み込むが、
  ROM は約 1MB の rodata セグメントを拒否する（`Invalid image block, can't boot`）。
  そこで別領域に書いたデータを実行時に MMU でマッピングして読む（`flashmap` パッケージ、ESP-IDF の `esp_mmu_map` 相当）。
- ブラウザ版とホストのテストでは同じ `slides.pack` を `go:embed` で埋め込む。
- `images/` の同梱画像はプログラムで生成したサンプル。

### SD カードのスライドショー（06_sdslideshow）

1. microSD カードを **FAT32** でフォーマットする（exFAT は未対応。32GB を超えるカードは exFAT のことが多い）。
2. カードに `slides` フォルダを作り（なければルートを探す）、画像を入れる。ファイル名順に表示される。
   - **BMP**：480×272、24bit または 32bit、無圧縮。多くの画像ソフトでこの形式で保存できる。
   - **.rgb565**：PNG / JPEG から変換する。
     ```sh
     go run ./tools/img2rgb565 -o /path/to/sdcard/slides ~/Pictures/*.jpg
     ```
     （切り抜かずに収めるなら `-fit contain` を付ける）
   - JPEG / PNG は実機では読めない（デコードに必要な RAM がない）ので、PC で変換しておく。
3. カードを挿して書き込む。
   ```sh
   make flash-noerase PKG=./examples/06_sdslideshow
   make monitor
   ```

- SD カードとタッチは SPI の SCK/MOSI/MISO を共有している（CS は SD=GPIO10、タッチ=GPIO38）。
  SD はハードウェア SPI、タッチはビットバングで使い、使う直前にピンの割り当てを切り替えている。
- SPI は 10MHz で読む。画像の一部の色がおかしい場合は `examples/06_sdslideshow/bus.go` の `sdFrequency` を 4MHz に下げる
  （ドライバがデータの CRC を検査しないため）。
- 読み込みに1枚1秒前後かかるので、切り替え効果は上から下・ブラインド・瞬時の3種類（左から右は遅いので使わない）。
- FAT を読む tinyfs/fatfs は C コードで `malloc` を使う。ESP32-S3 のターゲットは `malloc` を Wi-Fi ドライバ用に
  差し替える設定（`--wrap=malloc`）なので、`internal/espmalloc` で TinyGo の `malloc` に戻している。

## ESP32-3248S035C（3.5インチ）

Sunton ESP32-3248S035C：ESP32（ESP-32S モジュール。中身は ESP32-S3 ではなく初代 ESP32、PSRAM なし）、3.5インチ 320×480 の ST7796（SPI）、GT911 静電容量タッチ（I2C）。横向き 480×320 で使う。

```sh
make flash-s035 PKG=./examples/s035_check   # まずこれで液晶とタッチを確認
make flash-s035 PKG=./cmd/games             # ゲーム
make examples-s035                          # この基板で動く全プログラムのビルド確認
```

- RAM が足りないのでフレームバッファは持たず、描画はその都度 SPI で液晶に送る（`spilcd`）。
- 480×272 用に作ったプログラム（03、07〜12、cmd/app、cmd/games）は、`platform.Init()` が返す**画面中央の 480×272 の窓**で動く（上下 24px は黒）。タッチも窓の座標に直す。
- 01・02・04・05・06 は S043 専用（RGB 液晶、PWM、SD とタッチの共有バスを使う）。
- バックライトは点灯・消灯のみ（TinyGo に ESP32 の PWM がないため）。

**s035_check で確かめること**（シリアル 115200bps でコマンドを送れる）

| 確認 | 期待 | 違ったら |
|---|---|---|
| ログの `gt911: address ... product 911` | タッチ IC が見つかる | 配線・ピン（board/s035）を確認 |
| 色の帯 | 左から赤・緑・青・シアン・マゼンタ・黄・白・黒 | `o`（赤青の入れ替え）、`i`（反転）で直る方を board/s035 の `MADCTL`・`Invert` に書く |
| 向き | 「TOP LEFT」が左上で読める | `r`（180度回転）。直ったら `MADCTL = 0xE8` にし、`k` でタッチを測り直す |
| タッチ | 黄色い点が指の位置に出る | `k`（4隅のタップ）で測り、ログの値を board/s035 の `TouchRaw*` に書く |
| 速さ | `f`・`t` の時間がログに出る | 参考値（全画面塗りつぶしの時間など） |

## ESP32-2432S028R（2.8インチ）

Sunton ESP32-2432S028R（いわゆる Cheap Yellow Display）：ESP32（PSRAM なし）、2.8インチ 240×320 の ILI9341（SPI）、XPT2046 抵抗膜タッチ。横向き 320×240 で使う。micro USB が1つの版を前提にしている（USB が2つの版は色の反転が要る、または ST7789 という報告がある）。

```sh
make flash-s028 PKG=./examples/s028_check   # まずこれで液晶とタッチを確認
make flash-s028 PKG=./cmd/app               # お絵描きアプリ（320×240 用の配置）
make flash-s028 PKG=./cmd/games             # ゲーム（320×240 用の配置）
make examples-s028                          # この基板で動く全プログラムのビルド確認
```

- LCDwiki の「2.8inch ESP32-32E Display」（E32R28T、USB Type-C・電池端子つき）も、液晶・タッチ・SD のピンが同じなので同じターゲットで動く。違うのは RGB LED の赤（IO22）、IO4（スピーカーアンプの有効化、Low で有効）、IO34（電池電圧）だけ（board/s028 の `E32R28T*`）。
- 液晶は 3.5インチ版と同じ `spilcd`（初期化手順だけ ILI9341 用）。タッチは S043 と同じ `xpttouch`（ピンは別）。
- `platform.Init()` は画面全体（320×240）を返す。ゲーム（cmd/games・08〜12）とお絵描きアプリは画面の大きさから配置を選び、320×240 用の配置で動く（ゴールデン画像は `*_small_*`）。インベーダーはスプライトを 1.5 倍で描く（480×272 では 2 倍）。07_breakout は 480×272 のままなので動かない。
- 表示（色・向き）は ESP32-2432S028R と E32R28T の両方で確認済み。速さは全画面の塗りつぶし 39ms、文字 300 字 139ms（E32R28T、SPI 40MHz）。
- タッチのキャリブレーション（`TouchRaw*`）は E32R28T で s028_check の `k` で測った値。ずれるときは同じように測って board/s028 に書く。
- 液晶の ID の読み出しは、MISO がつながっているのに全部 0 だった。TinyGo の ESP32 の SPI で読み出しができていないらしい（SD カードを使うときに調べる）。
- バックライト（GPIO21）は点灯・消灯のみ。

**s028_check で確かめること**（シリアル 115200bps。コマンドは s035_check と同じ）

| 確認 | 期待 | 違ったら |
|---|---|---|
| 起動時のログ `RDDID(04)` | ILI9341 なら `00 93 41` 前後（MISO が液晶につながっている） | 全部 0（今はこれ）なら TinyGo の SPI の読み出しが効いていない |
| 色の帯 | 左から赤・緑・青・シアン・マゼンタ・黄・白・黒 | `o`（赤青）、`i`（反転） |
| 向き | 「TOP LEFT」が左上 | `r`（180度）。直ったら `MADCTL = 0xE8` |
| タッチ | 黄色い点がペン先の位置に出る | `k`（4隅のタップ）で測り、ログの値を board/s028 の `TouchRaw*` に書く |

## ミニジョイスティックモジュール（joystick_check）

[nulllaborg/mini-joystick-module](https://github.com/nulllaborg/mini-joystick-module)（ジョイスティックとボタン4つ、CH32V003、I2C アドレス 0x5A）を ESP32 の基板につなぐ確認用プログラム。ドライバは [minijoy](minijoy/minijoy.go)。

- **3.3V で給電する**（5V にしない）。モジュールの SDA・SCL は 4.7kΩ でモジュールの電源に引き上げられているので、5V で給電すると ESP32 の端子に 5V がかかる。CH32V003 は 2.7V から動き、ESP32-2432S028R の 3.3V で動いた。
- 配線（examples/joystick_check の `pins_*.go`）：

| 基板 | 配線 | 書き込み |
|---|---|---|
| ESP32-2432S028R | CN1：GND、3.3V、IO27=SDA、IO22=SCL | `make flash-s028 PKG=./examples/joystick_check` |
| E32R28T | SPI 端子 IO27=SDA・IO18=SCL、拡張端子 3.3V・GND | `make flash-s028 PKG=./examples/joystick_check TAGS=e32r28t` |
| ESP32-3248S035C | P4：GND、3V3、IO17=SDA、IO18=SCL（モジュールと同じ並び。IO18 は microSD の SCK と共用で、SD とは同時に使えない） | `make flash-s035 PKG=./examples/joystick_check` |

- 起動時のログ：I2C のスキャン結果、全レジスタの値、1回の読み取り時間。そのあと画面にスティックの位置とボタンの状態（押している間は緑）を出し、変化を時刻（ms）付きでログに出す。まとめ読み・リピートスタートでの読み出しは、スティックを斜め以外に倒すかボタンを押したときに比べる（中央で何も押していないと全レジスタが同じ値で、比べても分からないため）。

ESP32-3248S035C の P4 はまだ実機で確かめていない。ESP32-2432S028R（CN1、100kHz）で確かめたこと：

| 項目 | 結果 |
|---|---|
| スキャン | 0x5A だけが見つかる |
| 読み取り時間 | 1レジスタずつ（Arduino ライブラリと同じ）2.8ms、まとめ読み 1.2ms。まとめ読み・リピートスタートとも同じ値が返る（レジスタ番号は1バイトごとに進む） |
| スティック | X・Y とも 0〜255、中央 128。右に倒すと X が増え、上に倒すと Y が増える。中央付近の小さな動きは 128 になる |
| ボタンの名前 | 印字は上 Y、下 A、左 X、右 B。レジスタは A 0x21、Y 0x22、B 0x23、X 0x24、スティック押し込み 0x20。Arduino ライブラリの BUTTON_A〜D（0x22、0x23、0x21、0x24）は印字と合わない |
| ボタンのイベント | 押している間 press down (0)、1秒押し続けると long press hold (6)。離すと press up (1)、0.3秒後に single click (3) か double click (4)（0.3秒より長く押したときは出ない）、そのあと none (8)。long press start (5) と press repeat (2) は 30ms ごとの読み取りでは見えなかった。押しているかどうかは `Event.Down()`（0、2、5、6） |

### ゲームコントローラーとして使う

cmd/games と 08〜14 は、起動時にモジュールを探し（`platform.InitPad`、起動後に挿しても見つからない）、見つかればコントローラーでも遊べる（タッチもそのまま使える）。スティックで黄色い枠（ポーカーは水色）を動かし、A で枠のものを押す。枠はコントローラーを一度使うと出る。

| 画面 | スティック | A（またはスティック押し込み） | B | X | Y |
|---|---|---|---|---|---|
| ランチャー | タイルを選ぶ | 始める | | | |
| メニュー（数独・オセロ） | ボタンを選ぶ | 押す | < GAMES | | |
| インベーダー | 砲台を左右に動かす | 押している間撃つ・始める | < GAMES（タイトルとゲームオーバー） | | |
| 数独 | マスを選ぶ（盤の右へ倒すと数字・ERASE など） | マス：数字を選ぶへ。数字：入れてマスに戻る | マスに戻る | ERASE | HINT |
| オセロ | 盤の上の枠（右へ倒すと UNDO・NEW） | 石を置く・ボタンを押す | | UNDO | |
| ソリティア | 山札・捨て札・組札・表向きのカード・上のボタン | カード（とその上のカード）を置ける所へ動かす | | UNDO | 山札をめくる |
| ポーカー | カードと下のボタン | カード：HOLD。ボタン：押す | | DEAL/DRAW | |
| カート（メニュー・結果） | ボタンを選ぶ | 押す | メニュー・< GAMES | | |
| カート（レース） | ハンドル | アクセル | ブレーキ | | レースをやめる |
| スターガード（メニュー・結果） | ボタンを選ぶ | 押す | < GAMES・メニュー | | |
| スターガード（ゲーム） | 照準を動かす | 押している間撃つ | | | ゲームをやめる |

- 配線は上の表のとおり（3.5インチ版の P4 は未確認）。E32R28T は `TAGS=e32r28t` でビルドする。
- 読み取りは毎フレーム 1.2ms（まとめ読み、100kHz）。モジュールが答えないフレームは何も押していない扱い。
- ブラウザ版では矢印キーがスティック、Z が A、X が B、A が X、S が Y、Enter が押し込み。

## PicoPad のゲーム

[PicoLibSDK](https://github.com/Panda381/PicoLibSDK) は PicoPad（RP2040、320×240 の液晶、十字キーと A・B・X・Y）の C の SDK で、ゲームのサンプルが付いている（`PicoPad/GAME/`。作者の自作分は Unlicense）。[picopad](picopad/picopad.go) パッケージは、そのゲームが使う SDK の関数を同じ名前・同じ引数で持つ。C のゲームを関数ごとに Go に書き換えれば、ゲームのループもそのまま動く。移植したのはテトリス（[games/tetris](games/tetris/tetris.go)、15_tetris）、迷路（[games/maze](games/maze/maze.go)、16_maze）、15 パズル（[games/fifteen](games/fifteen/fifteen.go)、17_fifteen）、ライフゲーム（[games/life](games/life/life.go)、18_life）。

```sh
make flash-s028 PKG=./examples/15_tetris    # ESP32-2432S028R（画面がちょうど 320×240）
make flash-s035 PKG=./examples/15_tetris    # ESP32-3248S035C（中央の 320×240）
make wasm PKG=./examples/15_tetris && make serve
make flash-s028 PKG=./examples/16_maze      # 迷路、15 パズル（17_fifteen）、ライフゲーム（18_life）も同じ
```

| PicoPad | この基板 |
|---|---|
| 十字キー | ジョイスティックのスティック（ブラウザでは矢印キー） |
| A・B・X・Y | ジョイスティックの A・B・X・Y（押し込みも A。ブラウザでは Z・X・A・S） |
| （ボタンだけ） | タッチ：画面を 3×3 に分けて、上・左・右・下の真ん中が十字キー、中央と右下が A、左下が B、右上が X、左上が Y |

- PicoPad のフレームバッファは 16bit（150KB）。ESP32（PSRAM なし）でこれを持つと、ヒープが約 37KB しか残らない（TinyGo 0.44 のビルドで確認）。そこで 8bit のパレット番号（75KB）で持ち、`DispUpdate` が変わった範囲を 8 行ずつ RGB565 に直して送る（ESP32 では DMA）。色には最初に描かれたときに番号を付け、256 色を超えたら一番近い色にする。テトリスは 80 色。
- RAM：ESP32-2432S028R で起動直後のヒープ約 186KB のうち、フレームバッファ 75KB と送信用のバッファ 10KB を使う（15〜17。18_life はゲームの分の 30KB も使い、残りは約 60KB。ESP32-4827S043 では起動直後のヒープ約 130KB のうち約 127KB を使うので、残りは約 5KB しかない）。
- 画像とフォントは、PicoLibSDK の C の配列（`img/*.cpp`、`_font/*.c`）を [tools/picoimg](tools/picoimg/main.go) で Go の文字列定数にしたもの（フラッシュに置かれる）。作り直すときは PicoLibSDK を取ってきて `PICOLIBSDK=~/PicoLibSDK go generate ./picopad ./games/...`。
- 元のゲームの画面写真と比べた：テトリスはゴールデン画像（`tetris_start`）の動かない部分（壁・枠・題字・統計のブロック）が `TETRIS.PNG` と1ピクセルも違わない。迷路は `MAZE.PNG` の 8×8 のマス 1,131 個、15 パズルは `FIFTEEN.PNG` の 48×48 のマス 16 個、ライフゲームは `LIFE.PNG` の 16×16 のマス 300 個がどれも、こちらで描いたタイルのどれかと1ピクセルも違わない。
- 音はまだない（C の `PLAYSOUND` はコメントにしてある）。`ResetToBootLoader`（PicoPad では SD カードのメニューに戻る）は `Run` から戻ること。cmd/games（ランチャー）にはまだ入れていない。
- 迷路のメインループは C のとおり待たずに回り、100 周ごとに画面全体を描き直す（同じ絵を送り続けるが、見た目は変わらない）。
- 15 パズルの X は、C と同じ探索で解く（タイルの組ごとに、近づける探索と揃える探索を交互に、届かなければ深くして）。探索の間はボタンも画面も止まる（PicoPad と同じ）。ブラウザでは 30 秒以内に解けた。
- 仕組みと、ほかのゲームを移植するときの手順は [docs/ARCHITECTURE.md 17章](docs/ARCHITECTURE.md#17-picopad-のゲームの移植picopad15)。

## ブラウザ（wasm）で動かす

```sh
make wasm     # TinyGo 付属の wasm_exec.js をコピーし web/app.wasm をビルド
make serve    # http://localhost:8080/ を開く
```

- マウスのクリック・ドラッグがタッチになる（右側の白い領域に描ける。下のパレットで色選択、CLEAR で消去）。
- キーボードがゲームコントローラーになる（矢印キー、Z＝A、X＝B、A＝X、S＝Y、Enter＝押し込み）。
- `println` の出力はブラウザの開発者コンソールに出る。
- `file://` では動かない（`.wasm` を `application/wasm` で配信する必要がある）。
- `wasm_exec.js` は TinyGo 付属のものを使う（Go 付属のものは互換性がない）。TinyGo を更新したら `make wasm` でコピーし直される。

wasm 版で確認できるのは描画・UI・アプリのロジックだけ。RAM 容量、描画速度、ティアリング、液晶の発色、
抵抗膜タッチの精度、Wi-Fi などは実機で確認すること。

## テスト

```sh
make test            # go test ./...（ゴールデン画像テストを含む）
make test-browser    # 任意：headless Chromium で web/ を開き、canvas を同じゴールデン画像と比較
make test-browser PKG=./examples/05_slideshow GOLDEN=slideshow_caption DUMP_MS=2500
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
- タッチのキャリブレーション値は1台の実測値。個体差があるので、ずれる場合は 04_touch で測り直す。
- PicoPad のゲーム：音、ランチャーへの追加、ほかのゲームの移植（docs/ARCHITECTURE.md 17章）。15〜18 は実機では未確認。
