import type { Plugin } from "vite";

// 生成 bindings のサービスモジュール群。ここへの import だけを shim に振り替える。
// domain/*/models は型のみで実行時コードを持たないので対象外 (design.md D2)。
const SERVICE_DIR = "bindings/github.com/yito0000/meal-cycle/internal/service";

/**
 * e2eBindings は Wails 生成 bindings を fetch 版 shim に解決し直す。
 *
 * コンポーネントは bindings を相対パスで import しているため resolve.alias が
 * 効かない。解決後の絶対パスで判定できる resolveId フックを使う。
 *
 * @param shimDir プロジェクトルートからの shim ディレクトリ相対パス
 */
export default function e2eBindings(shimDir: string): Plugin {
  // vite の id は OS を問わず / 区切りに正規化される。
  const relShimDir = shimDir.replace(/^\.\//, "").replace(/\/$/, "");
  let shimRoot = relShimDir;

  return {
    name: "e2e-bindings",
    enforce: "pre",

    configResolved(config) {
      shimRoot = `${config.root.replace(/\\/g, "/").replace(/\/$/, "")}/${relShimDir}`;
    },

    async resolveId(source, importer, options) {
      // shim 自身が生成 bindings の型を参照するので、ここを振り替えると循環する。
      if (importer?.startsWith(shimRoot)) {
        return null;
      }

      const resolved = await this.resolve(source, importer, { ...options, skipSelf: true });
      if (!resolved || resolved.external) {
        return null;
      }

      const id = resolved.id.replace(/\\/g, "/");
      const slash = id.lastIndexOf("/");
      if (slash < 0 || !id.slice(0, slash).endsWith(SERVICE_DIR)) {
        return null;
      }

      const base = id.slice(slash + 1).replace(/\.(ts|js)$/, "");
      if (base === "models") {
        return null;
      }
      return `${shimRoot}/${base}.ts`;
    },
  };
}
