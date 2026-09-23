import { useTranslation } from "react-i18next";

// 追加行 / 編集行の入力欄の直下に出すエラー。key は i18n key。
export function FieldError({ messageKey, id }: { messageKey?: string; id?: string }) {
  const { t } = useTranslation();
  if (!messageKey) return null;
  return (
    <p id={id} role="alert" className="mt-1 text-xs text-red-700">
      {t(messageKey)}
    </p>
  );
}
