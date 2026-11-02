import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import type {
  Template,
  TemplateItem,
} from "../../../bindings/github.com/yito0000/meal-cycle/internal/domain/mealplan/models";
import { MealPlanService } from "../../../bindings/github.com/yito0000/meal-cycle/internal/service";

const TEMPLATES_KEY = ["mealplan-templates"] as const;

export type TemplateView = Omit<Template, "items"> & { items: TemplateItem[] };

export function useTemplates(enabled: boolean) {
  return useQuery({
    queryKey: TEMPLATES_KEY,
    queryFn: async (): Promise<TemplateView[]> =>
      ((await MealPlanService.ListTemplates()) ?? []).map((t) => ({ ...t, items: t.items ?? [] })),
    enabled,
    // レシピの削除で品の「削除済み」が変わるので、開くたびに取り直す。
    staleTime: 0,
  });
}

export function useSaveTemplate() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ weekStart, name }: { weekStart: string; name: string }) =>
      MealPlanService.SaveTemplate(weekStart, name),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: TEMPLATES_KEY }),
  });
}

export function useDeleteTemplate() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => MealPlanService.DeleteTemplate(id),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: TEMPLATES_KEY }),
  });
}
