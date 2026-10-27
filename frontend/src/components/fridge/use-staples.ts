import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import type {
  Staple,
  StapleState,
} from "../../../bindings/github.com/yito0000/meal-cycle/internal/domain/fridge/models";
import { FridgeService } from "../../../bindings/github.com/yito0000/meal-cycle/internal/service";

export const STAPLE_LIST_KEY = ["fridge", "staples"] as const;

export function useStapleList() {
  return useQuery({
    queryKey: STAPLE_LIST_KEY,
    queryFn: async () => (await FridgeService.ListStaples()) ?? [],
  });
}

// 常備品の状態は S07 の常備品の欄と材料の寄せ先を変えるので、買い物リストも捨てる (add-fridge-staples design.md D9)。
function useInvalidateStaples() {
  const queryClient = useQueryClient();
  return () =>
    Promise.all([
      queryClient.invalidateQueries({ queryKey: STAPLE_LIST_KEY }),
      queryClient.invalidateQueries({ queryKey: ["shopping"] }),
    ]);
}

export function useAddStaple() {
  const invalidate = useInvalidateStaples();
  return useMutation({
    mutationFn: (name: string) => FridgeService.AddStaple(name),
    onSuccess: invalidate,
  });
}

export function useSetStapleState() {
  const invalidate = useInvalidateStaples();
  return useMutation({
    mutationFn: ({ id, state }: { id: string; state: StapleState }) =>
      FridgeService.SetStapleState(id, state),
    onSuccess: invalidate,
  });
}

export function useDeleteStaple() {
  const invalidate = useInvalidateStaples();
  return useMutation({
    mutationFn: (item: Staple) => FridgeService.DeleteStaple(item.id),
    onSuccess: invalidate,
  });
}

export function useRestoreStaple() {
  const invalidate = useInvalidateStaples();
  return useMutation({
    mutationFn: (item: Staple) => FridgeService.RestoreStaple(item),
    onSuccess: invalidate,
  });
}
