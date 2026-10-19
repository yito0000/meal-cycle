# リリースとバージョン運用

meal-cycle のバージョン番号の上げ方、ブランチの使い方、リリースの手順をまとめる。
1.0.0 で GitHub Releases に Windows 版の配布を始めたのに合わせて定めた。

## バージョン番号の決め方

バージョンは [Semantic Versioning](https://semver.org/lang/ja/) の `MAJOR.MINOR.PATCH` に従う。
ただし meal-cycle はライブラリではないので、互換性を測る対象は API ではない。
利用者の手元にあるもの（SQLite の DB と、インポート用の Markdown ファイル）が、新しいバージョンでそのまま使えるかで判断する。

- **MAJOR**：既存の DB やインポート用ファイルが、新しいバージョンでそのままでは使えなくなる変更。
  マイグレーションで移行できないスキーマ変更や、[レシピのインポート形式](recipe-import.md)を後方互換なしに変える場合が当たる。
- **MINOR**：機能の追加と、マイグレーションを伴う変更。
  OpenSpec の proposal を 1 件 archive したら、通常は MINOR を上げる。
- **PATCH**：不具合の修正と小さな改善で、マイグレーションを伴わないもの。

マイグレーションを含む変更を PATCH にしないのは、goose のマイグレーションが前進のみで、古いバージョンに戻すと新しいスキーマの DB を開けなくなるためである。
PATCH であれば、いつでも 1 つ前に戻せる。
利用者がそう期待できるよう、DB に触れる変更は必ず MINOR 以上にする。

## ブランチ

開発は main ブランチで行い、リリースは main のコミットに `vX.Y.Z` 形式のタグを打って作る。
個人開発なので、develop ブランチやリリースごとのブランチは常設しない。

例外は、古いバージョンにだけ修正を出す必要が生じた場合である。
そのときは該当タグから `release/X.Y` ブランチを切り、修正をコミットして PATCH のタグを打つ。
同じ修正が main にも必要なら、cherry-pick で main に取り込む。

## リリース手順

1. `build/config.yml` の `info.version` を新しいバージョンに書き換える。
2. `wails3 task common:update:build-assets` を実行し、Windows / macOS / Linux 向けの各マニフェストにバージョンを反映する。
   `build/ios/` にも差分が出るが、モバイルは MVP 対象外なので `git checkout -- build/ios` で戻す。
3. [CLAUDE.md のテスト実行ルール](../CLAUDE.md#テスト実行ルール)にあるテストと `task test:e2e`、`task lint` をすべて通す。
4. `task build:win` で `bin/meal-cycle.exe` を作り、Windows 実機で起動と日本語入力を確認する。
5. バージョン更新をコミットして main に push する（例：`chore: release v1.0.0`）。
6. タグを打って push する。

   ```bash
   git tag -a v1.0.0 -m "v1.0.0"
   git push origin v1.0.0
   ```

7. チェックサムを作り、exe と一緒に GitHub Release へ載せる。

   ```bash
   cd bin && sha256sum meal-cycle.exe > SHA256SUMS.txt
   gh release create v1.0.0 bin/meal-cycle.exe bin/SHA256SUMS.txt --title "v1.0.0" --notes-file <リリースノート>
   ```

## 配布物

GitHub Releases に置くのは Windows（amd64）版の `meal-cycle.exe` だけである。
macOS 版は WSL からクロスビルドできず、Linux 版は利用者の WebKit2GTK 環境への依存が大きいため、どちらもソースからのビルドを案内する。

exe にはコード署名をしていない。
個人で証明書を維持する費用に見合わず、署名しても SmartScreen の評判が貯まるまでは警告が消えないためである。
代わりに `SHA256SUMS.txt` を添え、README で SmartScreen の警告が出たときの起動方法を案内する。
