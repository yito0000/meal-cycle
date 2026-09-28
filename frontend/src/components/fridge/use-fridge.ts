import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import type { StockItem } from "../../../bindings/github.com/yito0000/meal-cycle/internal/domain/fridge/models";
import { FridgeService } from "../../../bindings/github.com/yito0000/meal-cycle/internal/service";

const LIST_KEY = ["fridge", "list"] as const;
const SUGGESTIONS_KEY = ["fridge", "suggestions"] as const;

// 「元に戻す」トーストの表示時間。ユーザー可変値ではないので config には置かない (design.md D9)。
export const UNDO_TOAST_MS = 5000;

export function useStockList() {
  return useQuery({
    queryKey: LIST_KEY,
    queryFn: async () => (await FridgeService.List()) ?? [],
  });
}

export function useStockSuggestions() {
  return useQuery({
    queryKey: SUGGESTIONS_KEY,
    queryFn: async () => {
      const s = await FridgeService.ListSuggestions();
      return { names: s.names ?? [], units: s.units ?? [] };
    },
  });
}

// 在庫名の増減で候補も変わるので、どの mutation でも一覧と候補の両方を捨てる (design.md D12)。
export function useInvalidateFridge() {
  const queryClient = useQueryClient();
  return () =>
    Promise.all([
      queryClient.invalidateQueries({ queryKey: LIST_KEY }),
      queryClient.invalidateQueries({ queryKey: SUGGESTIONS_KEY }),
    ]);
}

export function useAddStock() {
  const invalidate = useInvalidateFridge();
  return useMutation({
    mutationFn: (input: StockItem) => FridgeService.Add(input),
    onSuccess: invalidate,
  });
}

export function useUpdateStock() {
  const invalidate = useInvalidateFridge();
  return useMutation({
    mutationFn: ({ id, input }: { id: string; input: StockItem }) =>
      FridgeService.Update(id, input),
    onSuccess: invalidate,
  });
}

export function useDeleteStock() {
  const invalidate = useInvalidateFridge();
  return useMutation({
    mutationFn: (item: StockItem) => FridgeService.Delete(item.id),
    onSuccess: invalidate,
  });
}

export function useRestoreStock() {
  const invalidate = useInvalidateFridge();
  return useMutation({
    mutationFn: (item: StockItem) => FridgeService.Restore(item),
    onSuccess: invalidate,
  });
}
