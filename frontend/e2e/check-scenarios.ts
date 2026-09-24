// E2E のテスト名が openspec の spec.md に実在するかを検査する。
// 対応規約: ファイル名の先頭語 = capability、最外の describe = Requirement 名、test 名 = Scenario 名。
// spec.md のシナリオすべてに E2E を求めはしない (サービス層で確かめるシナリオもあるため)。
import { spawnSync } from "node:child_process";
import { existsSync, readdirSync, readFileSync } from "node:fs";
import { join, resolve } from "node:path";

type Spec = Map<string, Map<string, Set<string>>>; // capability → Requirement → Scenario

type Suite = { title: string; specs?: SpecEntry[]; suites?: Suite[] };
type SpecEntry = { title: string; tests: { expectedStatus: string }[] };

const frontendDir = resolve(import.meta.dirname, "..");
const openspecDir = resolve(frontendDir, "..", "openspec");

function specFiles(): [string, string][] {
  const files: [string, string][] = [];
  const add = (specsDir: string) => {
    if (!existsSync(specsDir)) return;
    for (const cap of readdirSync(specsDir)) {
      const f = join(specsDir, cap, "spec.md");
      if (existsSync(f)) files.push([cap, f]);
    }
  };
  add(join(openspecDir, "specs"));
  // 実装中の change の delta も対象にする。archive 前に E2E を書くため。
  const changesDir = join(openspecDir, "changes");
  for (const name of readdirSync(changesDir)) {
    if (name !== "archive") add(join(changesDir, name, "specs"));
  }
  return files;
}

function loadSpec(): Spec {
  const spec: Spec = new Map();
  for (const [cap, file] of specFiles()) {
    const reqs = spec.get(cap) ?? new Map<string, Set<string>>();
    spec.set(cap, reqs);
    let current: Set<string> | undefined;
    for (const line of readFileSync(file, "utf8").split("\n")) {
      const req = line.match(/^### Requirement:\s*(.+?)\s*$/);
      if (req) {
        current = reqs.get(req[1]) ?? new Set();
        reqs.set(req[1], current);
        continue;
      }
      const sc = line.match(/^#### Scenario:\s*(.+?)\s*$/);
      if (sc && current) current.add(sc[1]);
    }
  }
  return spec;
}

function listTests(): { file: string; path: string[]; title: string; skipped: boolean }[] {
  const res = spawnSync("npx", ["playwright", "test", "--list", "--reporter=json"], {
    cwd: frontendDir,
    encoding: "utf8",
    maxBuffer: 64 * 1024 * 1024,
  });
  if (res.status !== 0) {
    process.stderr.write(res.stderr);
    process.exit(1);
  }
  const out: { file: string; path: string[]; title: string; skipped: boolean }[] = [];
  const walk = (s: Suite, file: string, path: string[]) => {
    for (const sp of s.specs ?? []) {
      const skipped = sp.tests.every((t) => t.expectedStatus === "skipped");
      out.push({ file, path, title: sp.title, skipped });
    }
    for (const c of s.suites ?? []) walk(c, file, [...path, c.title]);
  };
  for (const file of JSON.parse(res.stdout).suites as Suite[]) walk(file, file.title, []);
  return out;
}

const spec = loadSpec();
const errors: string[] = [];
const seen = new Set<string>();
const covered = new Map<string, { run: number; skip: number }>();

for (const t of listTests()) {
  const where = `${t.file} > ${[...t.path, t.title].join(" > ")}`;
  const cap = t.file.match(/^([a-z]+)/)?.[1] ?? "";
  const reqs = spec.get(cap);
  if (!reqs) {
    errors.push(`${where}: capability "${cap}" の spec.md が無い`);
    continue;
  }
  const req = t.path[0];
  const scenarios = req === undefined ? undefined : reqs.get(req);
  if (!scenarios) {
    errors.push(`${where}: 最外の describe "${req ?? ""}" が ${cap} の Requirement に無い`);
    continue;
  }
  if (!scenarios.has(t.title)) {
    errors.push(`${where}: Requirement "${req}" に Scenario "${t.title}" が無い`);
    continue;
  }
  const key = `${cap}\0${req}\0${t.title}`;
  if (seen.has(key)) errors.push(`${where}: 同じ Scenario のテストが複数ある`);
  seen.add(key);
  const c = covered.get(cap) ?? { run: 0, skip: 0 };
  c[t.skipped ? "skip" : "run"]++;
  covered.set(cap, c);
}

for (const [cap, reqs] of [...spec].sort(([a], [b]) => a.localeCompare(b))) {
  const total = [...reqs.values()].reduce((n, s) => n + s.size, 0);
  const c = covered.get(cap) ?? { run: 0, skip: 0 };
  console.log(`${cap}: E2E ${c.run} 件 (skip ${c.skip} 件) / Scenario ${total} 件`);
}

if (errors.length > 0) {
  console.error(`\nspec.md と対応しない E2E テストが ${errors.length} 件ある:`);
  for (const e of errors) console.error(`  ${e}`);
  process.exit(1);
}
