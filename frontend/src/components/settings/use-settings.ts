import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import i18n from "@/i18n";
import type { Values } from "../../../bindings/github.com/yito0000/meal-cycle/internal/domain/settings/models";
import { SettingsService } from "../../../bindings/github.com/yito0000/meal-cycle/internal/service";

export const SETTINGS_KEY = ["settings"] as const;

// 生成 bindings は配列を nullable で出すので、受け取った時点で空配列にそろえる。
export type SettingsValues = {
  [K in keyof Values]-?: NonNullable<Values[K]>;
};

export function toSettingsValues(v: Values): SettingsValues {
  return {
    ...v,
    leftoverCopyPresets: v.leftoverCopyPresets ?? [],
    shoppingPeriodDayPresets: v.shoppingPeriodDayPresets ?? [],
    measuringUnits: v.measuringUnits ?? [],
    unquantifiedWords: v.unquantifiedWords ?? [],
  };
}

// 保存するまで値は変わらないので取り直さない (add-settings-screen design.md D8)。
export function useSettings() {
  return useQuery({
    queryKey: SETTINGS_KEY,
    queryFn: async () => toSettingsValues(await SettingsService.Get()),
    staleTime: Number.POSITIVE_INFINITY,
  });
}

// S05 の期限状態や S07 の合算はサービスが設定値で計算した結果なので、設定以外の query をすべて取り直す。
function useApplySettings() {
  const queryClient = useQueryClient();
  return async (saved: Values) => {
    const next = toSettingsValues(saved);
    queryClient.setQueryData(SETTINGS_KEY, next);
    await i18n.changeLanguage(next.language);
    await queryClient.invalidateQueries({
      predicate: (q) => q.queryKey[0] !== SETTINGS_KEY[0],
    });
  };
}

export function useSaveSettings() {
  const apply = useApplySettings();
  return useMutation({
    mutationFn: (values: SettingsValues) => SettingsService.Save(values),
    onSuccess: apply,
  });
}

export function useResetSettings() {
  const apply = useApplySettings();
  return useMutation({
    mutationFn: () => SettingsService.Reset(),
    onSuccess: apply,
  });
}
