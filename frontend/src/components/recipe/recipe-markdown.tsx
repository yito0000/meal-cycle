import Markdown, { type Components } from "react-markdown";

// 完全ローカル前提のため、画像は取りに行かず alt だけ、リンクは遷移しない文字列にする (design.md D6)。
// rehype-raw は入れない。生 HTML を通すと <img> 経由で外部読み込みが復活する。
const components: Components = {
  img: ({ alt }) => <span>{alt}</span>,
  a: ({ children }) => <span>{children}</span>,
  // ドロワーの欄見出し (h3) の下に入るので、h1/h2 を h4、h3 を h5 に下げる。
  h1: ({ children }) => <h4 className="mt-3 mb-1 text-base font-semibold">{children}</h4>,
  h2: ({ children }) => <h4 className="mt-3 mb-1 text-base font-semibold">{children}</h4>,
  h3: ({ children }) => <h5 className="mt-2 mb-1 text-sm font-semibold">{children}</h5>,
  p: ({ children }) => <p className="my-1 whitespace-pre-wrap">{children}</p>,
  ul: ({ children }) => <ul className="my-1 list-disc pl-5">{children}</ul>,
  ol: ({ children }) => <ol className="my-1 list-decimal pl-5">{children}</ol>,
  li: ({ children }) => <li className="my-0.5">{children}</li>,
  blockquote: ({ children }) => (
    <blockquote className="my-2 border-l-2 border-neutral-300 pl-3 text-neutral-600">
      {children}
    </blockquote>
  ),
  pre: ({ children }) => (
    <pre className="my-2 overflow-x-auto rounded bg-neutral-100 p-2 text-xs">{children}</pre>
  ),
  code: ({ children }) => (
    <code className="rounded bg-neutral-100 px-1 font-mono text-[0.9em]">{children}</code>
  ),
};

export function RecipeMarkdown({ text }: { text: string }) {
  return (
    <div className="text-sm leading-relaxed text-neutral-800">
      <Markdown components={components}>{text}</Markdown>
    </div>
  );
}
