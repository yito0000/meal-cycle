import { useQuery } from "@tanstack/react-query";
import { RecipeService } from "../../../bindings/github.com/yito0000/meal-cycle/internal/service";

// S03 と S04 で同じ query key を共有し、S04 の保存時の invalidate が S03 にも効くようにする (design.md D5)。
export function useRecipeDetail(id: string | null) {
  return useQuery({
    queryKey: ["recipes", "detail", id],
    queryFn: () => RecipeService.GetByID(id as string),
    enabled: id !== null,
  });
}
