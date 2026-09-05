import { Plus, X } from "lucide-react";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";

type Props = {
  value: string[];
  onChange: (next: string[]) => void;
};

// chip 入力のタグエディタ (design.md D11 参照)。
// - 入力用テキストボックスに文字を入力
// - Enter または「＋」ボタンで確定 → chip として追加
// - 各 chip の「×」で削除
// - 空文字 / 重複は黙って捨てる
// - 30 字を超える入力は input 側で maxLength=30 で防ぐ
const TAG_MAX_LENGTH = 30;

export function RecipeTagsInput({ value, onChange }: Props) {
  const { t } = useTranslation();
  const [draft, setDraft] = useState("");

  const commit = () => {
    const trimmed = draft.trim();
    if (trimmed === "") return;
    if (value.includes(trimmed)) {
      setDraft("");
      return;
    }
    onChange([...value, trimmed]);
    setDraft("");
  };

  const remove = (tag: string) => {
    onChange(value.filter((t) => t !== tag));
  };

  return (
    <div className="flex flex-col gap-2">
      <div className="flex items-center gap-2">
        <Input
          type="text"
          value={draft}
          maxLength={TAG_MAX_LENGTH}
          placeholder={t("pages.recipes.edit.tag_placeholder")}
          onChange={(e) => setDraft(e.target.value)}
          onKeyDown={(e) => {
            // IME 変換中の Enter (変換確定) は無視する。isComposing が
            // false のときだけ「タグ確定」として扱う。これを見ないと
            // 日本語入力の変換確定が奪われて、変換途中の文字がタグ化される。
            if (e.key === "Enter" && !e.nativeEvent.isComposing) {
              e.preventDefault();
              commit();
            }
          }}
          aria-label={t("pages.recipes.edit.label.tags")}
        />
        <Button
          type="button"
          variant="outline"
          size="icon"
          onClick={commit}
          aria-label={t("pages.recipes.edit.add_tag")}
        >
          <Plus className="h-4 w-4" />
        </Button>
      </div>
      {value.length > 0 && (
        <ul className="flex flex-wrap gap-2">
          {value.map((tag) => (
            <li
              key={tag}
              className="flex items-center gap-1 rounded-full bg-neutral-100 px-3 py-1 text-sm text-neutral-800"
            >
              <span>{tag}</span>
              <button
                type="button"
                onClick={() => remove(tag)}
                aria-label={t("pages.recipes.edit.remove_tag")}
                className="rounded-full p-0.5 text-neutral-500 hover:bg-neutral-200 hover:text-neutral-800"
              >
                <X className="h-3 w-3" />
              </button>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
