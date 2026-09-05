import * as React from "react";
import { cn } from "@/lib/utils";

// shadcn の Label プリミティブは htmlFor / 子テキストを持たない汎用ラッパー。
// 各利用箇所で htmlFor を指定するのが規約なので、a11y の関連付けルールは
// このコンポーネント定義側では無効化する (呼び出し側で担保する)。
export const Label = React.forwardRef<
  HTMLLabelElement,
  React.LabelHTMLAttributes<HTMLLabelElement>
>(({ className, ...props }, ref) => (
  // biome-ignore lint/a11y/noLabelWithoutControl: shadcn のラッパープリミティブは htmlFor を呼び出し側で付ける。
  <label
    ref={ref}
    className={cn(
      "text-sm font-medium leading-none peer-disabled:cursor-not-allowed peer-disabled:opacity-70",
      className,
    )}
    {...props}
  />
));
Label.displayName = "Label";
