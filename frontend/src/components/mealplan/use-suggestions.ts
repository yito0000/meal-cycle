import { useQuery } from "@tanstack/react-query";
import { SuggestionService } from "../../../bindings/github.com/yito0000/meal-cycle/internal/service";

// 在庫や献立の変更から invalidate せず、ダイアログを開くたびに取り直す (add-mealplan-suggest-by-stock design.md D4)。
export function useSuggestions(enabled: boolean) {
  return useQuery({
    queryKey: ["suggestions"],
    queryFn: async () => {
      const r = await SuggestionService.List();
      return {
        byStock: (r.byStock ?? []).map((c) => ({ ...c, stocks: c.stocks ?? [] })),
        byRecency: r.byRecency ?? [],
      };
    },
    enabled,
    staleTime: 0,
  });
}
