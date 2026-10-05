import type * as React from "react";

// IME の変換確定の Enter を送信や候補確定と取り違えないための判定。WebKit 系は
// compositionend 直後に isComposing=false の keydown が来ることがあり、keyCode 229 を併用する (add-fridge-inventory design.md D6)。
export function isImeComposing(e: React.KeyboardEvent): boolean {
  return e.nativeEvent.isComposing || e.keyCode === 229;
}
