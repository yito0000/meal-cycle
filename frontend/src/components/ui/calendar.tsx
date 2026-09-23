import { ChevronLeft, ChevronRight } from "lucide-react";
import type * as React from "react";
import { DayPicker } from "react-day-picker";
import { cn } from "@/lib/utils";

export type CalendarProps = React.ComponentProps<typeof DayPicker>;

// shadcn の Calendar を手動コピーし、WebView のネイティブ日付ピッカーより大きく見せるため
// 日付セルを 40px 四方 / text-base に広げている (add-shopping-list design.md D15)。
export function Calendar({
  className,
  classNames,
  showOutsideDays = true,
  ...props
}: CalendarProps) {
  return (
    <DayPicker
      showOutsideDays={showOutsideDays}
      className={cn("p-1", className)}
      classNames={{
        months: "relative flex flex-col gap-4",
        month: "flex flex-col gap-3",
        nav: "absolute inset-x-0 top-0 flex items-center justify-between",
        button_previous:
          "inline-flex h-9 w-9 items-center justify-center rounded-md text-neutral-700 hover:bg-neutral-100 disabled:opacity-40",
        button_next:
          "inline-flex h-9 w-9 items-center justify-center rounded-md text-neutral-700 hover:bg-neutral-100 disabled:opacity-40",
        month_caption: "flex h-9 items-center justify-center",
        caption_label: "text-base font-semibold",
        month_grid: "border-collapse",
        weekdays: "flex",
        weekday: "w-10 text-sm font-normal text-neutral-500",
        week: "mt-1 flex",
        day: "h-10 w-10 p-0 text-center text-base",
        day_button:
          "h-10 w-10 rounded-md hover:bg-neutral-100 focus-visible:outline-2 focus-visible:outline-blue-600",
        selected: "[&>button]:bg-blue-600 [&>button]:text-white [&>button]:hover:bg-blue-700",
        today: "[&>button]:font-bold [&>button]:ring-1 [&>button]:ring-blue-300",
        outside: "text-neutral-400",
        disabled: "opacity-40",
        hidden: "invisible",
        ...classNames,
      }}
      components={{
        Chevron: ({ orientation, className: chevronClassName }) =>
          orientation === "left" ? (
            <ChevronLeft className={cn("h-5 w-5", chevronClassName)} />
          ) : (
            <ChevronRight className={cn("h-5 w-5", chevronClassName)} />
          ),
      }}
      {...props}
    />
  );
}
