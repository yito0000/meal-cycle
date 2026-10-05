import { useQuery } from "@tanstack/react-query";
import { FridgeService } from "../../../bindings/github.com/yito0000/meal-cycle/internal/service";

export const INGREDIENT_SUGGESTIONS_KEY = ["ingredient", "suggestions"] as const;

export function useIngredientSuggestions() {
  return useQuery({
    queryKey: INGREDIENT_SUGGESTIONS_KEY,
    queryFn: async () => {
      const s = await FridgeService.ListSuggestions();
      return { names: s.names ?? [], units: s.units ?? [] };
    },
  });
}
