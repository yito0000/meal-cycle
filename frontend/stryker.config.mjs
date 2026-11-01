/** @type {import('@stryker-mutator/api/core').PartialStrykerOptions} */
export default {
  testRunner: "vitest",
  vitest: { configFile: "vitest.config.ts" },
  mutate: [
    "src/**/*.{ts,tsx}",
    "!src/**/*.test.{ts,tsx}",
    "!src/components/ui/**",
    "!src/main.tsx",
    "!src/router.tsx",
    "!src/i18n.ts",
  ],
  // TypeScript 7 には JS API (parseConfigFileTextToJson) が無く、tsconfig の書き換え処理が落ちる。
  // 存在しないパスを渡すと書き換えがスキップされる。Vitest は tsconfig を読まないので影響しない。
  tsconfigFile: "stryker-skip-tsconfig-rewrite.json",
  ignoreStatic: true,
  reporters: ["clear-text", "progress", "html"],
  htmlReporter: { fileName: "reports/mutation/index.html" },
  concurrency: 4,
  tempDirName: ".stryker-tmp",
};
