import type * as React from "react";
import { Toaster as Sonner, type ToasterProps } from "sonner";

// shadcn 版から next-themes の useTheme を外している。本アプリはライトテーマ固定で、
// shadcn の CSS 変数 (--popover 等) も持たないため Tailwind の neutral 色を直接渡す。
export function Toaster(props: ToasterProps) {
  return (
    <Sonner
      theme="light"
      className="toaster group"
      style={
        {
          "--normal-bg": "var(--color-white)",
          "--normal-text": "var(--color-neutral-900)",
          "--normal-border": "var(--color-neutral-200)",
        } as React.CSSProperties
      }
      {...props}
    />
  );
}
