import { CalendarDays } from "lucide-react";
import { useState } from "react";
import { enUS, ja } from "react-day-picker/locale";
import { useTranslation } from "react-i18next";
import { formatDisplayDate } from "@/lib/format-date";
import { cn } from "@/lib/utils";
import { Button } from "./button";
import { Calendar } from "./calendar";
import { Popover, PopoverContent, PopoverTrigger } from "./popover";

type Props = {
  id?: string;
  // YYYY-MM-DD (ローカル日付)。
  value: string;
  onChange: (next: string) => void;
  className?: string;
};

function toLocalDate(isoDate: string): Date {
  const [y, m, d] = isoDate.split("-").map(Number);
  return new Date(y, m - 1, d);
}

function toIsoDate(d: Date): string {
  const pad = (n: number) => String(n).padStart(2, "0");
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`;
}

// WebView のネイティブ日付入力は大きさも表示形式も環境任せになるため、アプリ内の Calendar で置き換える (design.md D15)。
export function DatePicker({ id, value, onChange, className }: Props) {
  const { i18n } = useTranslation();
  const [open, setOpen] = useState(false);
  const selected = toLocalDate(value);

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>
        <Button
          id={id}
          type="button"
          variant="outline"
          className={cn("w-40 justify-start gap-2 font-normal", className)}
        >
          <CalendarDays className="h-4 w-4 text-neutral-500" aria-hidden="true" />
          {formatDisplayDate(value)}
        </Button>
      </PopoverTrigger>
      <PopoverContent align="start" className="w-auto">
        <Calendar
          mode="single"
          required
          locale={i18n.language.startsWith("ja") ? ja : enUS}
          selected={selected}
          defaultMonth={selected}
          onSelect={(d) => {
            onChange(toIsoDate(d));
            setOpen(false);
          }}
        />
      </PopoverContent>
    </Popover>
  );
}
