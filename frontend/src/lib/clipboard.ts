export async function writeClipboardText(text: string): Promise<void> {
  if (navigator.clipboard?.writeText) {
    try {
      await navigator.clipboard.writeText(text);
      return;
    } catch {
      // secure context でない webview や権限の拒否では失敗するので、Wails の runtime に回す (add-shopping-copy-text design.md D2)。
    }
  }
  // 動的 import にして、runtime の無い E2E のビルドで起動時に読み込ませない。
  const { Clipboard } = await import("@wailsio/runtime");
  await Clipboard.SetText(text);
}
