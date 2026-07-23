# gplay — Google Play Developer Publishing API CLI

Google Play Developer Publishing API（Android Publisher API v3）を操作するCLIツール。AWS CLIと同様のプロファイル方式でクレデンシャルを管理します。

## インストール

```bash
go build ./cmd/gplay          # カレントに gplay を出力
# または
go install github.com/ideamans/google-play-developer-publishing-cli/cmd/gplay@latest
```

## セットアップ

Google Cloud Consoleで作成したサービスアカウントJSONキーを登録します。

```bash
gplay configure --key ~/Downloads/play-publisher-abc123.json --package com.example.app
```

事前に必要な作業（いずれもWeb UI、初回のみ）:

1. Google Cloud Console で **Google Play Android Developer API** を有効化
2. IAM & 管理 > サービスアカウント で JSON キーを作成
3. Play Console > ユーザーと権限 でそのサービスアカウントを招待し、対象アプリへのアクセス権を付与

> Play Console の「API アクセス」ページから GCP プロジェクトをリンクする手順が案内されることがありますが、
> このページはアカウントの状態によっては開けません（ホームにリダイレクトされます）。
> その場合でも上記 1〜3 だけで完結します。サービスアカウントの招待は「ユーザーと権限」から行えます。

登録されるもの:

- キーは `~/.config/google-play-developer-publishing/keys/` にコピーされ（パーミッション0600）、プロファイルが `config.toml` に登録されます
- `--package` / `--developer-id` はプロファイルの既定値になり、以降のコマンドで省略できます
- 最初に登録したプロファイルがデフォルトになります
- 別アカウントのキーは `--profile <name>` を付けて登録します

```bash
gplay configure --profile client-a --key ~/Downloads/client-a.json --package com.client.app
```

## プロファイル管理

```bash
gplay profiles list          # 一覧（デフォルトに印が付く）
gplay profiles use client-a  # デフォルトを切り替え
gplay profiles remove NAME   # プロファイル削除（キーファイルは残る）
```

## 使い方

```bash
gplay details get                       # 認証・権限の確認を兼ねる
gplay --profile client-a tracks list    # プロファイル指定
gplay tracks list --json                # 生のAPI JSONを出力

# 未対応のエンドポイントは汎用コマンドで
gplay api /applications/com.example.app/reviews
gplay api -X POST /applications/com.example.app/edits
gplay api --paginate --items reviews "/applications/com.example.app/reviews?maxResults=100"

# curl等で使うアクセストークンを発行
curl -H "Authorization: Bearer $(gplay token)" \
  https://androidpublisher.googleapis.com/androidpublisher/v3/applications/com.example.app/reviews
```

## edit（編集トランザクション）モデル

ストア掲載情報・画像・バイナリ・トラックの変更は、すべて **edit** というステージング領域を経由します。editをcommitするまで、変更はGoogle Playに反映されません。

単発の操作ではCLIが自動でedit作成→変更→commitまで行うので意識不要です。複数の変更を1回のcommitにまとめたい場合はeditを明示的に引き回します。

```bash
EDIT=$(gplay edits create)
gplay listings set --edit $EDIT --language ja --title "..." --full-description @desc-ja.txt
gplay images replace --edit $EDIT --language ja --type phoneScreenshots --file 01.png --file 02.png
gplay tracks set --edit $EDIT --track internal --version-code 42
gplay edits commit $EDIT
```

- `--edit <id>` … 既存のeditに追記する（commitしない）
- `--no-commit` … editを作って変更を適用し、edit idを標準出力に出して開いたままにする
- `--dry-run` … 送信せずにリクエスト内容を標準エラーへ出力（ネットワーク書き込みは一切行わない）
- 参照系のedit対象コマンドは使い捨てeditを作り、読んだあと削除します

edit対象のコマンド: `listings` `details` `images` `bundles` `apks` `mapping` `expansion` `tracks` `testers` `release`。それ以外は即時反映です。

## リリースフロー用コマンド

```bash
# ビルドをアップロードしてトラックに載せ、commitまで一括実行
gplay release --track internal --aab app-release.aab
gplay release --track production --aab app-release.aab --mapping mapping.txt \
  --user-fraction 0.1 --notes ja=@notes-ja.txt --notes en-US=@notes-en.txt

# 段階的リリースの操作
gplay tracks rollout --track production --user-fraction 0.5 --changes-not-sent-for-review
gplay tracks halt --track production          # 一時停止
gplay tracks complete --track production      # 100%配信

# トラック間の昇格（元トラックは空になる = Play Consoleと同じ挙動）
gplay tracks promote --from beta --to production --user-fraction 0.1

# ストア掲載情報
gplay listings set --language ja --title "レシート読取" --short-description "..." --full-description @desc-ja.txt
gplay images replace --language ja --type phoneScreenshots --file 01.png --file 02.png
gplay details set --contact-email support@example.com --contact-website https://example.com/support

# テスター（Googleグループ）
gplay testers set --track alpha --group qa@example.com
```

## 全コマンド一覧（APIドメイン別）

Android Publisher API v3（ディスカバリドキュメント revision 20260722）の **全143メソッドを網羅** しています。書き込み系はすべて `--dry-run` 対応。詳細は `gplay <command> --help` / `gplay --llm` を参照。

| コマンド | 対応ドメイン |
|---------|-------------|
| `edits` | edit作成・取得・削除・検証・commit |
| `listings` / `details` | 言語別ストア掲載情報（タイトル・説明・動画）、連絡先・デフォルト言語 |
| `images` | アイコン・機能グラフィック・TVバナー・各種スクリーンショット（アップロード前に寸法検証） |
| `bundles` / `apks` | .aab / .apk のアップロードと一覧、外部ホスト型APKの登録 |
| `mapping` / `expansion` | ProGuardマッピング・ネイティブシンボル、拡張ファイル（OBB） |
| `tracks` / `testers` | トラックとリリース（段階的リリース・昇格・停止・完了・国別指定）、クローズドトラックのテスターグループ |
| `release` | アップロード→トラック設定→commit の一括実行 |
| `internal-sharing` | 内部アプリ共有（審査もトラックも介さない共有リンク） |
| `products` | 管理対象アプリ内商品（inappproducts、価格は `JPY:480` 形式で指定） |
| `subscriptions` | 定期購入・基本プラン・オファー（monetization API、一括更新・状態変更・価格移行含む） |
| `one-time-products` | 1回限りの商品・購入オプション・オファー（新モデル、一括操作含む） |
| `pricing` | 価格の全地域変換（Play Consoleの価格換算と同じ計算） |
| `purchases` | 購入の検証・承認・消費・解約・返金・取り消し（v1 / v2）、無効化された購入の一覧 |
| `orders` | 注文の取得・一括取得・返金・返金リクエストの審査 |
| `reviews` | レビューの一覧・取得・返信 |
| `generated-apks` / `system-apks` | Playが生成したAPKのダウンロード、システムイメージ用APKバリアント |
| `device-tier-configs` | デバイスティア設定（Play Asset Delivery向け） |
| `data-safety` | データセーフティ（プライバシー）申告のCSV投入 |
| `app-recovery` | アプリ復旧アクション（不具合バージョンの救済） |
| `users` / `grants` | デベロッパーアカウントのユーザーとアプリ単位の権限 |
| `external-transactions` | 代替課金システムの取引報告 |
| `apps` | アクセス可能なアプリの一覧（後述のとおりReporting APIを使用） |
| `app-store` | **Google Play以外のアプリストア運営者向け**：ホスト対象アプリの登録・審査提出・APK/画像/ポリシー宣言ファイルのアップロード、Playカタログエクスポートの参照 |

`app-store` はGoogleが承認したアプリストア事業者のみが利用でき、それ以外のアカウントでは403になります。アプリ開発者としての利用には不要です。

複雑なリソース（基本プラン、オファー、デバイスティア設定、外部取引など）は `--from-json @file.json` でリソースJSONをそのまま渡せます。将来APIに追加されたエンドポイントは `gplay api` で呼び出せます。

### APIでは操作できず人間が行う必要がある工程

- **アプリの新規作成と初回バイナリの公開**: Play Consoleで行う必要があります。1本もアップロードされていないパッケージに対しては、APIは常に404を返します。
- **コンテンツのレーティング審査・ターゲット層・広告の申告・アプリアクセス権（審査用ログイン情報）**: Web UIのみ。
- **デベロッパー配布契約の同意、お支払いプロファイル・税務情報**: Web UIのみ（未了だと課金機能が有効になりません）。
- **クローズドトラックの個別テスター（メールアドレス単位）**: APIはGoogleグループのみ扱えます。
- **データセーフティの設問への回答**: APIはPlay Consoleがエクスポート/インポートするCSVのみ受け付けます（`gplay data-safety --file data-safety.csv`）。
- **統計・Android vitals・売上レポート**: このAPIには含まれません。Play Developer Reporting APIおよびCloud Storageのレポートバケットにあります。

### 実運用で踏みやすい落とし穴（対策を組み込み済み）

- **アプリ一覧を返すエンドポイントが存在しない**。`gplay apps list` は Play Developer Reporting API を使うため、そのAPIの有効化が別途必要です（未有効なら手順を示すエラーになります）。他のコマンドは影響を受けません。
- **バージョンコードは増加必須**。同じバージョンコードの再アップロードは失敗します。
- **トラックの更新はリリース一覧の置き換え**。`gplay tracks set` は指定した1つのリリースだけを送ります（通常のデプロイはこれが正しい挙動）。既存リリースを残したい場合は `--keep-existing`。
- **userFractionは0 < f < 1 かつ inProgress/halted のみ**。draft/completedでは自動的に除去し、0や1は事前にエラーにします（100%配信は `tracks complete`）。
- **`--changes-not-sent-for-review` は審査の抜け道ではない**。審査不要な変更（段階的リリースの割合など）にのみ使えます。
- **掲載情報の文字数制限**（タイトル30 / 簡単な説明80 / 詳細な説明4000）を送信前に検証します。
- **画像は寸法が厳密**（アイコン512×512 / 機能グラフィック1024×500 / TVバナー1280×720、スクリーンショットは各辺320〜3840px）。アップロード前にローカル検証するので、どのファイルが原因か即座に分かります。
- **スクリーンショットは追加される**（uploadは追加、`images replace` が冪等）。
- **50GBまでのAABは再開可能アップロード**で送信し、進捗を標準エラーに表示します。
- **購入は3日以内にacknowledgeが必要**（未承認だと自動返金されます）。新規実装では v2 エンドポイント（`purchases subscription-v2` など）を推奨します。
- **権限とAPI有効化の反映には数分かかる**ことがあります。403が出たら少し待って再試行してください。
- **`403 PERMISSION_DENIED` は「認証は成功、権限が無い」の意味**です。認証失敗（401）やAPI未有効化とは別物なので、切り分けて対処してください。トークン自体が取れているかは `gplay token` で確認できます。
- **`users list` は `pageSize=-1` が必須**です（APIが他の値を拒否します）。ディスカバリドキュメントからは読み取れない挙動で、gplay 側で対応済みです。

## ヘルプ

- `gplay --help` / `gplay <command> --help` — 人間向けの通常のヘルプ
- `gplay --llm` — LLMエージェント向けの詳細リファレンスを一括出力（クレデンシャルモデル、解決順序、editモデル、全コマンド・フラグ、`gplay api` のページネーションの注意点まで含む）。どのサブコマンドに付けても同じ全文が出ます

## クレデンシャルの解決順序

1. 環境変数 `GPLAY_SERVICE_ACCOUNT_BASE64`（CI向け、JSONキーのbase64）→ `GPLAY_SERVICE_ACCOUNT_JSON`（パス）→ `GOOGLE_APPLICATION_CREDENTIALS`（パス）
2. `--profile` フラグ → 環境変数 `GPLAY_PROFILE` → `config.toml` の `default_profile`

パッケージ名は `--package` / `-p` → `GPLAY_PACKAGE` → プロファイルの `package` の順に解決します。

`.env` ファイルは読み込みません。direnv等でシェル側で環境変数化してください。

## 設定ファイル

```
~/.config/google-play-developer-publishing/
├── config.toml        # プロファイル定義（0600）
└── keys/              # サービスアカウントJSONキー（0700 / 各ファイル0600）
    └── default.json
```

```toml
default_profile = "default"

[profiles.default]
service_account = "keys/default.json"
package = "com.example.app"
developer_id = "1234567890123456789"
```

## CI での利用

```yaml
env:
  GPLAY_SERVICE_ACCOUNT_BASE64: ${{ secrets.PLAY_SERVICE_ACCOUNT_BASE64 }}
  GPLAY_PACKAGE: com.example.app
run: |
  gplay release --track internal --aab app-release.aab --mapping mapping.txt \
    --notes ja=@notes-ja.txt
```
