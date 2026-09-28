import { useQuery } from "@tanstack/react-query";
import { ShoppingService } from "../../../bindings/github.com/yito0000/meal-cycle/internal/service";

// 他画面の mutation からは invalidate しない。S07 はマウントのたびに再取得するので、開き直した時点で反映される
// (add-shopping-list design.md D8)。S07 自身の一括登録だけは invalidate する (add-shopping-purchase design.md D6)。
export function useShoppingList(periodStart: string, periodDays: number) {
  return useQuery({
    queryKey: ["shopping", periodStart, periodDays],
    queryFn: async () => {
      const l = await ShoppingService.List(periodStart, periodDays);
      return {
        ...l,
        dishes: (l.dishes ?? []).map((d) => ({ ...d, dates: d.dates ?? [], meals: d.meals ?? [] })),
        items: (l.items ?? []).map((it) => ({ ...it, recipes: it.recipes ?? [] })),
      };
    },
  });
}
