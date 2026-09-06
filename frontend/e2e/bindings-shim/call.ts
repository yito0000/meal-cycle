import { CancellablePromise } from "@wailsio/runtime";

// cmd/e2e-server が公開する呼び出し口。Wails の /wails/runtime は webview の
// 外から到達できないため、E2E ではこちらを叩く (design.md D1)。
const API_PREFIX = "/api";

// call は生成 bindings の $Call.ByID と同じ位置に立つ。戻り値を
// CancellablePromise にするのは、生成物と型を一致させて type-check で
// ずれを検出するため (design.md D5)。
export function call<T>(
  service: string,
  method: string,
  ...args: unknown[]
): CancellablePromise<T> {
  return new CancellablePromise<T>((resolve, reject) => {
    fetch(`${API_PREFIX}/${service}/${method}`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(args),
    })
      .then(async (res) => {
        const text = await res.text();
        if (!res.ok) {
          reject(toError(text));
          return;
        }
        resolve(text === "" ? (undefined as T) : (JSON.parse(text) as T));
      })
      .catch(reject);
  });
}

// Go 側は {"kind": ..., "message": ...} を返す。kind を name に載せて、
// 呼び出し側が errors.Is 相当の分岐をできるようにする。
function toError(body: string): Error {
  try {
    const parsed = JSON.parse(body) as { kind?: string; message?: string };
    const err = new Error(parsed.message ?? body);
    if (parsed.kind) {
      err.name = parsed.kind;
    }
    return err;
  } catch {
    return new Error(body);
  }
}
