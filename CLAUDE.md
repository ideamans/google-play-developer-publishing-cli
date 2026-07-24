# CLAUDE.md — google-play-developer-publishing-cli

Google Play Developer Publishing API（Android Publisher v3）の非対話 CLI。
**バイナリ名は `gplay`**、goreleaser のプロジェクト名は
`google-play-developer-publishing`（`-cli` なし）、リポジトリ名は
`google-play-developer-publishing-cli`。**3つとも違う**ので、リリースアセット名や
インストール手順を書くときは取り違えないこと。

姉妹プロジェクトは [apple-app-store-connect-cli](https://github.com/ideamans/apple-app-store-connect-cli)
（同じ設計思想の iOS 側）。

## 変更時の必須手順

**機能を追加した、フラグを増やした、既存の挙動を変えた — このいずれかをしたら、
3か所すべてを更新してから終わること。**

| 更新先 | 対象 | やり方 |
| --- | --- | --- |
| ① ドキュメント | `README.md` | 使い方が変わったときのみ |
| ② ヘルプ | cobra の `Short` / `Long` / `Example` / フラグ説明 | コード内。**カタログはここから生成される**ので、ここを厚くすると③も良くなる |
| ③ **LLMナレッジ** | `internal/llmdocs/00-guide.md` | 認証・**編集モデル**・`gplay api`・代表フローが変わったら |
| | `internal/llmdocs/10-pitfalls.md` | **実リリースで判明した罠を見つけたら必ず追記** |
| | `internal/llmdocs/90-commands.md` | **生成物。手編集しない** → `go generate ./...` |
| | `plugins/google-play-developer-publishing-cli/skills/*/SKILL.md` | 手順や前提が変わったとき |
| | `context7.json` の `rules` | 新しい落とし穴が生まれたとき |

③ を忘れやすい。ドキュメントとヘルプは人間が読んで気づくが、**LLMナレッジが
古いことには誰も気づかない**（エージェントが黙って間違えるだけ）。

判断に迷ったときの目安:

- **実リリースで罠を踏んだ** → `10-pitfalls.md` に追記する。Google のドキュメントにも
  コマンド一覧にも書かれていない知識が溜まる場所で、この CLI で最も価値がある
- **編集（edit）モデルの扱いを変えた** → `00-guide.md`。commit し忘れると
  「変更したのに反映されない」という、エラーにならない失敗になる
- 段階的ロールアウトやトラック昇格の挙動を変えた → `gplay-usage` の SKILL.md。
  **本番ロールアウトは実ユーザーに即座に届く**ため、事前提示と同意の手順を保つこと
- 新しいコマンドを足した → ②の `Short` / `Long` / `Example` を書いてから
  `go generate ./...`

## リリース

`PluginVersion`（`cmd/root.go`）と `plugin.json` の `version` と git タグの3つを
揃える。テストとリリースワークフローが不一致を検出する。手順は
`plugins/google-play-developer-publishing-cli/PUBLISH.md`。

## 秘密情報

サービスアカウントの JSON 鍵は**内容を出力しない・コマンドラインに書かない**。

## 確認

```bash
go generate ./...     # 生成物を作り直す
git diff --exit-code  # 差分が出たらコミット漏れ
go test ./...         # SKILL.md 検証とバージョン整合を含む
go run ./cmd/gplay llm | head
```

## 参照

- 標準: <https://github.com/ideamans/go-llm-cli-kit/blob/main/LLM.md>
- 生成物と原本の対応: `.claude/rules/ai-artifacts-policy.md`
- 再生成: `/regen-ai`
